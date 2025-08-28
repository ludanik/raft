package main

import (
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Peer struct {
	id   int32
	addr string
	conn *grpc.ClientConn
	stub RaftServiceClient
}

func NewPeer(id int32, addr string) *Peer {
	// only try to connect to peer when u are candidate
	return &Peer{
		id:   id,
		addr: addr,
		conn: nil,
		stub: nil,
	}
}

func (p *Peer) Connect() error {
	conn, err := grpc.NewClient(p.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Could not connect to peer", err)
		return err
	}

	client := NewRaftServiceClient(conn)
	p.conn = conn
	p.stub = client
	return nil
}
