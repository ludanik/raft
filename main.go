package main

// make the log first ezpz
type Log struct {
}

type ServerState int

const (
	FOLLOWER ServerState = iota
	CANDIDATE
	LEADER
)

type PersistentConfig struct {
	log         Log
	currentTerm int
	votedFor    int
}

type Config struct {
	pConfig     PersistentConfig
	commitIndex int
	lastApplied int
	state       ServerState
}

func main() {

}
