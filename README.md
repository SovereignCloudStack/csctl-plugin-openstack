# CSCTL plugin for OpenStack

## Table of Contents

- [CSCTL plugin for OpenStack](#csctl-plugin-for-openstack)
  - [Table of Contents](#table-of-contents)
  - [Introduction](#introduction)
  - [Features of csctl plugin for OpenStack](#features-of-csctl-plugin-for-openstack)
  - [Docs](#docs)

## Introduction

Cluster Stacks are intended to be well-tested bundles of Kubernetes manifests designed to bootstrap productive Kubernetes clusters using the CAPI approach.

In the case of OpenStack as the infrastructure layer, several custom components, in addition to the CAPI and CAPO (Cluster API provider OpenStack) operators, are involved in the Kubernetes cluster lifecycle management (LCM):

- CSO (Cluster Stack Operator)
- CSPO (Cluster Stack Provider OpenStack)
- CSCTL (CLI for Cluster Stacks management)

**CSO** is the provider-agnostic component that handles the core processes.

**CSPO** is the provider-specific component responsible for uploading the node images to the OpenStack project, for later consumption by the CAPO.

**CSCTL** facilitates the Cluster Stack creation and versioning process.

This project facilitates building node images that can be used with the Cluster Stack Operator.

## Features of csctl plugin for OpenStack

1. The fully automated building and uploading process for node images, which can be referenced in the Cluster Stack.
2. Generating `node-images.yaml` file, which is needed when you want to use images in the Cluster Stack that are not in your OpenStack Glance service.

## Docs
[Docs](./docs/README.md)
