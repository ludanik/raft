# State

Persist currentTerm, votedFor, log[] on machine

## Persistent state 

### currentTerm, votedFor
    - one file for both
    - ex. line
    -     v
    -     0:currentTerm
    -     1:votedFor

### Log persistence
    - persist log using text file, new line is new entry
    - read whole log into memory on startup
    - line n is entry n
    - then each value is comma separated
    - first value is term
    - second value can be command
    - ex. line
    -     |  term
    -     v  v
    -     0: 0,SET?key=foo&val=bar
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

Use gRPC, probably easiest way

### RequestVote RPC
### AppendEntries RPC




 