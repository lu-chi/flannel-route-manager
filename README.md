# flannel-route-manager

The flannel route manager syncs the [flannel](https://github.com/coreos/flannel) routing table to the specified backend.

## Overview

* [Usage](#usage)
* [Backends](#backends)
* [Build](#build)
* [Single Node Demo](#single-node-demo)
## Usage

```
Usage of ./flannel-route-manager:
  -backend="google": backend provider
  -etcd-endpoint="http://127.0.0.1:4001": etcd endpoint
  -etcd-prefix="/coreos.com/network": etcd prefix
  -network="default": google compute network
  -project="": google compute project id
  -sync-interval=30: sync interval
```

### Example

```
flannel-route-manager -project hightower-labs
```

## Backends

flannel-route-manager has been designed to support multiple backends, but only ships a single backend today -- the google backend.

### google

The google backend syncs the flannel route table from etcd to GCE for a specific GCE project and network. Currently routes are only created or updated for each subnet managed by flannel.

Routes are created naming scheme:

```
default-route-flannel-10-0-63-0-24
```

#### Requirements

* [enabled IP forwarding for instances](https://developers.google.com/compute/docs/networking#canipforward) 
* [instance service account](https://developers.google.com/compute/docs/authentication#using)
* [project ID](https://developers.google.com/compute/docs/overview#projectids)

The google backend relies on instance service accounts for authenitcation. See [Preparing an instance to use service accounts](https://developers.google.com/compute/docs/authentication#using) for more details.

Creating a compute instance with the right permissions and IP forwarding enabled:

```
$ gcloud compute instances create INSTANCE --can-ip-forward --scopes compute-rw
```

## Build

```
mkdir -p "${GOPATH}/src/github.com/kelseyhightower"
cd "${GOPATH}/src/github.com/kelseyhightower"
git clone https://github.com/kelseyhightower/flannel-route-manager.git
cd flannel-route-manager
godep go build .
```

## Single Node Demo

Add your GCE project ID to the cloud-config.yaml file:

```
write_files:
  - path: /etc/flannel-route-manager.conf
    permissions: 0644
    owner: root
    content: |
      GOOGLE_PROJECT_ID=""
```

The following command will create a GCE instance with flannel and the flannel-route-manager up and running. 

```
$ gcloud compute instances create flannel-route-manager-test \
--image-project coreos-cloud \
--image coreos-alpha-440-0-0-v20140915 \
--machine-type g1-small \
--can-ip-forward \
--scopes compute-rw \
--metadata-from-file user-data=cloud-config.yaml \
--zone us-central1-a
```

Once the instance is fully booted you should see a new route added under the default network.
