# Using the csctl plugin for OpenStack

## What does the csctl plugin for OpenStack do?

As a user, you can create clusters based on Cluster Stacks with the help of the Cluster Stack Operators (CSO and CSPO). The operators need certain files, such as those required to apply the necessary Helm charts and to obtain information about the versions in the cluster stack.

To avoid generating these files manually, you can use [CSCTL](https://github.com/SovereignCloudStack/csctl). In the case of provider-specific Cluster Stacks, the `CSCTL` tool invokes the provider-specific CSCTL plugin. Therefore, the CSCTL plugin for OpenStack is essential if the user intends to build, upload node images to an S3 bucket, and then import them into Glance.

## Different methods of csctl plugin for OpenStack

The csctl plugin for OpenStack offers two methods that can be used for different use cases. You can configure them in `csctl.yaml` at `config.provider.config.method`, see [example of the csctl.yaml](../example/cluster-stacks/openstack/ferrol/csctl.yaml) file.

> [!NOTE]
> Please note that in both methods you need to specify the `config.yaml` file in the `node-images` folder similar to a provided [example](../example/cluster-stacks/openstack/ferrol/node-images/config.yaml).

### Get method

This method can be used when the creator of the cluster-stacks has already built and stored image(s) in some S3 storage. Then, they need to insert the URL to those image(s) in the `config.yaml`. The plugin, based on the configuration file, then generates `node-images.yaml` file in the release directory.

### Build method

The use case for this method is the opposite of the `Get` method. It means that the cluster-stack creator intends to use an image that has not yet been built. The plugin then builds image(s) based on Packer scripts in the `node-images` folder and pushes these image(s) to an S3 bucket. In this mode, you need to provide the path to your S3 storage credentials using the `--node-image-registry` flag, see [registry.yaml](../example/cluster-stacks/openstack/ferrol/node-images/registry.yaml). The URL does not need to be set in `config.yaml`. The plugin can create it for you based on the following patterns:

- for an `S3` type registry:

  ```bash
  <endpoint>/<bucket-name>/<image-dir-name>
  ```

- for a `Swift` type registry:

  ```bash
  <endpoint>/swift/v1/AUTH_<project-ID>/<bucket-name>/<image-dir-name>
  ```

Be aware of that in this method you need to specify `imageDir` in `config.yaml` file.

> [!NOTE]
> If you want to use URL creation for OpenStack Swift registry, please change the `registry.yaml` file accordingly:

```yaml
type: Swift
config:
  endpoint: <endpoint>
  bucket: <bucket_name>
  accessKey: <access_key>
  secretKey: <secret_key>
  projectID: <openstack_project_id>
  # verify: false  # Only if you want to disable SSL certificate verification and use `http` url in endpoint
  # cacert: <path/to/cacert> # Use this field only if the S3 storage endpoint certificate is signed by a custom(non-public) authority
```

## Installing csctl plugin for OpenStack

You can click on the respective release of the csctl plugin for OpenStack on GitHub and download the binary.

Assuming, you have downloaded the `<release-name>` binary in your Downloads directory, you will need the following commands to rename the binary and to give it executable permissions.

```bash
sudo chmod u+x ~/Downloads/<release-name>
sudo mv ~/Downloads/<release-name> /usr/local/bin/csctl-openstack # or use any bin directory from your PATH
```

Then you can check whether everything worked by printing the version of csctl-openstack.

```bash
$ csctl-openstack version
csctl-openstack version: v0.0.1-10-gb6fd166
commit: b6fd16665c5e28d122ce50063977a599d809f641
```

If you don't see a version there, then something has gone wrong. Re-check above steps and open an issue if it still does not work!

If you're using `gh` CLI then you can also use the following to download it.

```bash
gh release download -p <release-name> -R SovereignCloudStack/csctl-plugin-openstack
```

## Creating node-images file in release directory of cluster-stacks

The most important subcommand is `create-node-images`. This command takes a path to the directory where you configured your Cluster Stack and generates the `node-images.yaml` file in the output directory.

```bash
csctl-openstack create-node-images cluster-stack-directory cluster-stack-release-directory
```

If you choose `build` method you need to provide the path to your node image registry credentials like this:

```bash
csctl-openstack create-node-images cluster-stack-directory cluster-stack-release-directory node-image-registry-path
```

Then the plugin build and push created node image(s) to the appropriate S3 bucket.

## Use csctl plugin for OpenStack with csctl

[CSCTL](https://github.com/SovereignCloudStack/csctl) contains a plugin mechanism for providers. This means csctl automatically invokes the plugin for OpenStack if the `csctl.yaml` file contains a configuration for the OpenStack, i.e., `config.provider.config`. In this case, csctl looks for an executable (binary) with a certain name: `csctl- + config.provider.type`. Please take a look at the example of a [csctl.yaml](../example/cluster-stacks/openstack/ferrol/csctl.yaml) file to understand how the configuration for the OpenStack plugin should be set up for csctl to be able to invoke the plugin. Then, you can use basic csctl commands to create cluster stacks. See [csctl documentation](https://github.com/SovereignCloudStack/csctl/blob/main/docs/how_to_use_csctl.md#creating-cluster-stacks) for more details.
