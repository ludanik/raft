Raft
==================

My implementation of the Raft distributed consensus protocol in Go, as described by the [Raft paper](https://raft.github.io/raft.pdf#page=1&zoom=200,87,407). This implementation supports all features described in the paper, except for snapshotting and cluster membership changes.

## Getting Started

1. Clone the Git repository, build Raft
```shell script
git clone https://github.com/ludanik/raft.git
cd raft
make build
```
2. Start a Raft cluster.
Open three terminal windows and run the Raft node on each. 
```shell script
# Terminal window 1:
./bin/raft --cluster="1:3001,2:3002,3:3003" --node=1
# Terminal window 2:
./bin/raft --cluster="1:3001,2:3002,3:3003" --node=2
# Terminal window 3:
./bin/raft --cluster="1:3001,2:3002,3:3003" --node=3
```

3. Submit commands to the cluster via HTTP API (POST to any node on port 8090)
```shell script
# Submit a command to the leader (will be accepted)
curl -X POST http://localhost:8090/addentry \
  -H "Content-Type: application/json" \
  -d '{"command":"set x=1"}'

# If you POST to a follower, it will redirect you to the leader
# Response: {"success":false,"message":"Not the leader. Redirect to node 1","leader":1}
```

## Features

- **Leader Election**: Automatic leader election with randomized timeouts
- **Log Replication**: Commands are replicated from leader to followers
- **Persistent State**: Logs are persisted to disk (files named `log1`, `log2`, `log3`)
- **HTTP API**: Submit commands via POST /addentry endpoint
- **Fault Tolerance**: Cluster continues operating with majority of nodes available


## Kubernetes Deployment

1. Build the container and deploy it to a running Kubernetes cluster.
You will need to write a deployment.yaml for your cluster.
```shell script
docker build -t github.com/ludanik/raft .
kubectl apply -f deployment.yaml
```

2. Get the pod name and view its output
```shell script
kubectl get pods
kubectl logs -f POD_NAME
```

## Misc

To regenerate Protobuf files
```shell script
protoc --go_out=. --go-grpc_out=. raft.proto
```




