# State

Persist currentTerm, votedFor, log[] on machine

## Persistent state 

### Log persistence
    - persist log using text file, new line is new entry
    - read whole log into memory on startup
    - line n is entry n
    - then each value is comma separated
    - first value is term
    - second value can be command
    - first line of log is (currentTerm, votedFor)
    - ex.
    -     0: 1,192.168.0.0.1
    -     1: 0,SET?key=foo&val=baz 
    -     2: 1,SET?key=meow&val=cow
    -     3: 2,SET?key=meow&val=now

## Variables on all servers
    - commitIndex: index of highest log entry known to be
                    committed (initialized to 0, increases
                    monotonically)
                    
    - lastApplied:  index of highest log entry applied to state
                    machine (initialized to 0, increases
                    monotonically)

## Variables on leaders
    -nextIndex[]: for each server, index of the next log entry
                to send to that server (initialized to leader
                last log index + 1)

    -matchIndex[]:for each server, index of highest log entry
                known to be replicated on server
                (initialized to 0, increases monotonically)


# Election

## RPC

Keep peers in memory and keep track of current leader

Can't just connect once to peers at the start cuz they can go offline and back online

Need to keep all peers in memory and try to make a connection every heartbeat or whatever

As follower, at start of loop check connection to leader only, if time expires try reconnect to all nodes, then send out RequestVote

Apparently connection autoreconnects anyway, so maybe we could try the original method of just keeping the connections in a hashmap

### RequestVote RPC

Just implement this as its written in the slides, what does the server do when it receives the RPC?

### AppendEntries RPC

Implement this same way it's written in the slides, what does the server do when it receives the RPC?




 