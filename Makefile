run: build
	@./bin/raft --cluster="1:3001,2:3002,3:3003" --node=1

build:
	@go build -race -o bin/raft .
