/*
Copyright 2024 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package main provides the OpenStack plugin for csctl.
// The "create-node-images" command to create node images during
// a `csctl create` call.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	csctlclusterstack "github.com/SovereignCloudStack/csctl/pkg/clusterstack"
	yaml "github.com/goccy/go-yaml"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// RegistryConfig represents the structure of the registry.yaml file.
type RegistryConfig struct {
	Type   string `yaml:"type"`
	Config struct {
		Endpoint  string `yaml:"endpoint"`
		Bucket    string `yaml:"bucket"`
		AccessKey string `yaml:"accessKey"`
		SecretKey string `yaml:"secretKey"`
		Verify    *bool  `yaml:"verify,omitempty"`
		Cacert    string `yaml:"cacert,omitempty"`
	} `yaml:"config"`
}

// OpenStackNodeImage represents the structure of the OpenStackNodeImage.
type OpenStackNodeImage struct {
	URL        string      `json:"url" yaml:"url"`
	ImageDir   string      `json:"imageDir,omitempty" yaml:"imageDir,omitempty"`
	CreateOpts *CreateOpts `json:"createOpts" yaml:"createOpts"`
}

// CreateOpts represents options used to create an image.
type CreateOpts images.CreateOpts

// NodeImages represents the structure of the config.yaml file.
type NodeImages struct {
	APIVersion          string                `yaml:"apiVersion"`
	OpenStackNodeImages []*OpenStackNodeImage `yaml:"openStackNodeImages"`
}

const (
	provider        = "openstack"
	outputDirectory = "./output"
)

func usage() {
	fmt.Printf(`%s create-node-images cluster-stack-directory cluster-stack-release-directory [node-image-registry-path]
This command is a csctl plugin.
https://github.com/SovereignCloudStack/csctl
`, os.Args[0])
}

func main() {
	minArgs := 4
	maxArgs := 5
	if len(os.Args) < minArgs || len(os.Args) > maxArgs {
		fmt.Printf("Wrong number of arguments. Expected %d or %d, got %d\n", minArgs, maxArgs, len(os.Args))
		usage()
		os.Exit(1)
	}
	if os.Args[1] != "create-node-images" {
		usage()
		os.Exit(1)
	}
	clusterStackPath := os.Args[2]
	csctlConfig, err := csctlclusterstack.GetCsctlConfig(clusterStackPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	configFilePath := filepath.Join(clusterStackPath, "node-images", "config.yaml")
	config, err := GetConfig(configFilePath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if csctlConfig.Config.Provider.Type != provider {
		fmt.Printf("Wrong provider in %s. Expected %s\n", clusterStackPath, provider)
		os.Exit(1)
	}
	releaseDir := os.Args[3]
	_, err = os.Stat(releaseDir)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	registryConfigPath := ""
	if len(os.Args) == 5 {
		registryConfigPath = os.Args[4]
	}

	method := csctlConfig.Config.Provider.Config["method"]
	switch method {
	case "get":
		// Copy config.yaml to releaseDir as node-images.yaml
		dest := filepath.Join(releaseDir, "node-images.yaml")
		if err := copyFile(configFilePath, dest); err != nil {
			fmt.Printf("Error copying config.yaml to releaseDir: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("config.yaml copied to releaseDir as node-images.yaml successfully!")
	case "build":
		if registryConfigPath == "" {
			fmt.Printf("Error: Please specify <node-image-registry-path> when using `build` method in csctl.yaml\n")
			os.Exit(1)
		}

		for imageOrder, image := range config.OpenStackNodeImages {
			if image.ImageDir == "" {
				fmt.Printf("No images to build, image directory is not defined in config.yaml file")
				os.Exit(1)
			}

			// Construct the path to the image folder
			packerImagePath := filepath.Join(clusterStackPath, "node-images", image.ImageDir)

			if _, err := os.Stat(packerImagePath); err != nil {
				fmt.Printf("Image folder %s does not exist\n", packerImagePath)
				os.Exit(1)
			}
			fmt.Println("Running packer build...")
			// Warning: variables like build_name and output_directory must exist in packer variables file like in example
			// #nosec G204
			cmd := exec.Command("packer", "build", "-var", "build_name="+image.ImageDir, "-var", "output_directory="+outputDirectory, packerImagePath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("Error running packer build: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Packer build completed successfully.")

			_, err = os.Stat(registryConfigPath)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			// Get the current working directory
			currentDir, err := os.Getwd()
			if err != nil {
				fmt.Printf("Error getting current working directory: %v\n", err)
				os.Exit(1)
			}

			// Path to the image created by the packer
			// Warning: name of the image created by packer should have same name as the name of the image folder in node-images
			ouputImagePath := filepath.Join(currentDir, outputDirectory, image.ImageDir)

			// Push the built image to S3
			if err := pushToS3(ouputImagePath, image.ImageDir, registryConfigPath); err != nil {
				fmt.Printf("Error pushing image to S3: %v\n", err)
				os.Exit(1)
			}

			// Update URL in config.yaml if it is necessary
			if err := updateURLNodeImages(configFilePath, registryConfigPath, image.ImageDir, imageOrder); err != nil {
				fmt.Printf("Error updating URL in config.yaml: %v\n", err)
				os.Exit(1)
			}
		}
		// Copy config.yaml to releaseDir as node-images.yaml
		dest := filepath.Join(releaseDir, "node-images.yaml")
		if err := copyFile(configFilePath, dest); err != nil {
			fmt.Printf("Error copying config.yaml to releaseDir: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("config.yaml copied to releaseDir as node-images.yaml successfully!")
	default:
		fmt.Println("Unknown method:", method)
		os.Exit(1)
	}
}

func pushToS3(filePath, fileName, registryConfigPath string) error {
	// Load registry configuration from YAML file
	// #nosec G304
	registryConfigFile, err := os.Open(registryConfigPath)
	if err != nil {
		return fmt.Errorf("error opening registry config file: %w", err)
	}
	defer registryConfigFile.Close()

	var registryConfig RegistryConfig
	decoder := yaml.NewDecoder(registryConfigFile)
	if err := decoder.Decode(&registryConfig); err != nil {
		return fmt.Errorf("error decoding registry config file: %w", err)
	}

	if registryConfig.Type != "S3" {
		return fmt.Errorf("error, only S3 compatible registry is supported")
	}

	// Remove "http://" or "https://" from the endpoint if present cause Endpoint cannot have fully qualified paths in minioClient.
	registryConfig.Config.Endpoint = strings.TrimPrefix(registryConfig.Config.Endpoint, "http://")
	registryConfig.Config.Endpoint = strings.TrimPrefix(registryConfig.Config.Endpoint, "https://")

	// Requests are always secure (HTTPS) by default unless `verify: false` is defined in registry.yaml to enable insecure (HTTP) access.
	useSSL := true

	if registryConfig.Config.Vefiry != nil {
		useSSL = *registryConfig.Config.Vefiry
	}

	// TLS configuration
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if registryConfig.Config.Cacert != "" {
		config.RootCAs = x509.NewCertPool()
		data, err := os.ReadFile(registryConfig.Config.Cacert)
		if err != nil {
			return fmt.Errorf("failed to read the CA certificate: %w", err)
		}
		ok := config.RootCAs.AppendCertsFromPEM(data)
		if !ok {
			// If no certificates were successfully parsed, set RootCAs to nil
			config.RootCAs = nil
		}
	}

	// Create custom HTTP transport using the TLS configuration
	customTransport := &http.Transport{
		TLSClientConfig: config,
	}

	// Initialize Minio client
	minioClient, err := minio.New(registryConfig.Config.Endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(registryConfig.Config.AccessKey, registryConfig.Config.SecretKey, ""),
		Secure:    useSSL,
		Transport: customTransport,
	})
	if err != nil {
		return fmt.Errorf("error initializing Minio client: %w", err)
	}

	// Open file to upload
	// #nosec G304
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %w", err)
	}

	// Upload file to bucket
	_, err = minioClient.PutObject(context.Background(), registryConfig.Config.Bucket, fileName, file, fileInfo.Size(), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("error uploading file: %w", err)
	}
	return nil
}

func updateURLNodeImages(configFilePath, registryConfigPath, imageName string, imageOrder int) error {
	// Read the config.yaml file
	// #nosec G304
	nodeImageData, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config.yaml: %w", err)
	}

	// Unmarshal YAML data into NodeImages struct
	var nodeImages NodeImages
	if err := yaml.Unmarshal(nodeImageData, &nodeImages); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Check if the URL already exists for the given image
	imageURLExists := nodeImages.OpenStackNodeImages[imageOrder].URL != ""

	// If the URL doesn't exist, update it for the image
	if !imageURLExists {
		// Load registry configuration from YAML file
		// #nosec G304
		registryConfigFile, err := os.Open(registryConfigPath)
		if err != nil {
			return fmt.Errorf("error opening registry config file: %w", err)
		}
		defer registryConfigFile.Close()

		var registryConfig RegistryConfig
		decoder := yaml.NewDecoder(registryConfigFile)
		if err := decoder.Decode(&registryConfig); err != nil {
			return fmt.Errorf("error decoding registry config file: %w", err)
		}
		// Generate URL
		newURL := fmt.Sprintf("%s%s/%s/%s", "https://", registryConfig.Config.Endpoint, registryConfig.Config.Bucket, imageName)

		// Assign the generated URL to the correct node-image
		nodeImages.OpenStackNodeImages[imageOrder].URL = newURL

		// Marshal the updated struct back to YAML
		updatedNodeImageData, err := yaml.Marshal(&nodeImages)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}

		// Write the updated YAML data back to the file
		if err := os.WriteFile(configFilePath, updatedNodeImageData, os.FileMode(0o644)); err != nil {
			return fmt.Errorf("failed to write config.yaml: %w", err)
		}

		fmt.Printf("URL updated for image: %s\n", newURL)
	} else {
		fmt.Printf("URL already exists for the image\n")
	}
	return nil
}

func copyFile(src, dest string) error {
	// #nosec G304
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("error reading source file: %w", err)
	}

	if err := os.WriteFile(dest, data, os.FileMode(0o644)); err != nil {
		return fmt.Errorf("error writing to destination file: %w", err)
	}

	return nil
}

// GetConfig returns Config.
func GetConfig(configPath string) (*NodeImages, error) {
	configFileData, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	nd := NodeImages{}
	if err := yaml.Unmarshal(configFileData, &nd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config yaml: %w", err)
	}

	if nd.APIVersion == "" {
		return nil, fmt.Errorf("api version must not be empty")
	}

	if len(nd.OpenStackNodeImages) == 0 {
		return nil, fmt.Errorf("at least one node image needs to exist in OpenStackNodeImages list")
	}

	// Ensure all fields in OpenStackNodeImages are defined
	for _, image := range nd.OpenStackNodeImages {
		switch {
		case image.CreateOpts == nil:
			return nil, fmt.Errorf("field CreateOpts must not be empty")
		case image.CreateOpts.Name == "":
			return nil, fmt.Errorf("field 'name' in CreateOpts must be defined")
		case image.CreateOpts.DiskFormat == "":
			return nil, fmt.Errorf("field 'disk_format' in CreateOpts must be defined")
		case image.CreateOpts.ContainerFormat == "":
			return nil, fmt.Errorf("field 'container_format' in CreateOpts must be defined")
		}
	}

	return &nd, nil
}
