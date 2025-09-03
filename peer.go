package main

import (
	"context"
	"log/slog"
	"time"

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

// we separate this into its own method so we can use gRPC deadlines
func (p *Peer) RequestVoteFromPeer(msg *RequestVoteMessage, timeout time.Duration) (*RequestVoteReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout/10)
	defer cancel()
	slog.Info("Trying to request vote", "addr", p.addr, "timeout", timeout/10)
	reply, err := p.stub.RequestVote(ctx, msg)
	if err != nil {
		return nil, err
	}
	return reply, err
}

// we separate this into its own method so we can use gRPC deadlines
func (p *Peer) AppendEntriesToPeer(msg *AppendEntriesMessage, timeout time.Duration) (*AppendEntriesReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout/10)
	defer cancel()
	slog.Info("Trying to append entries", "addr", p.addr, "timeout", timeout/10)
	reply, err := p.stub.AppendEntries(ctx, msg)
	if err != nil {
		return nil, err
	}
	return reply, err
}
