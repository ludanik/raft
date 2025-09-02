Raft
==================

My implementation of the Raft distributed consensus protocol, as described by the [Raft paper](https://raft.github.io/raft.pdf#page=1&zoom=200,87,407), in Golang.

This implementation supports all features described in the paper, except for snapshotting and cluster membership changes.

## Getting Started

1. Clone the Git repository 
```shell script
git clone https://github.com/ludanik/raft.git
cd raft
```

2. Build the container and deploy it to a running Kubernetes cluster. You will need to write a deployment.yaml for your cluster.
```shell script
docker build -t github.com/ludanik/raft .
kubectl apply -f deployment.yaml
```

3. Get the pod name and view its output
```shell script
kubectl get pods
kubectl logs -f <POD_NAME>
```




