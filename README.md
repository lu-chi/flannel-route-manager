# flannel-route-manager

The flannel route manager syncs the [flannel](https://github.com/coreos/flannel) routing table to the specified backend.

## Backends

### google

#### Requirements

* [instance service account](https://developers.google.com/compute/docs/authentication#using)
* [project ID](https://developers.google.com/compute/docs/overview#projectids)

The google backend relies on instance service accounts for authenitcation. See [Preparing an instance to use service accounts](https://developers.google.com/compute/docs/authentication#using) for more details.

Creating a compute instance with the right permissions:

```
$ gcloud compute instances create INSTANCE --scopes compute-rw
```

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

## Building

```
mkdir -p "${GOPATH}/src/github.com/kelseyhightower"
cd "${GOPATH}/src/github.com/kelseyhightower"
git clone https://github.com/kelseyhightower/flannel-route-manager.git
cd flannel-route-manager
godep go build .
```
