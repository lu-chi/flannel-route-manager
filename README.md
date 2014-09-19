# flannel-route-manager

The flannel route manager syncs the [flannel](https://github.com/coreos/flannel) routing table to the specified backend.

## Usage

```
Usage of ./flannel-route-manager:
  -backend="google": backend provider
  -etcd-endpoint="http://127.0.0.1:4001": etcd endpoint
  -etcd-prefix="/coreos.com/network": etcd prefix
  -network="default": google compute network
  -project="": google compute project name
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
