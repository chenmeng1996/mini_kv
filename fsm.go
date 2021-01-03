package main

import (
	"encoding/json"
	"io"
	"log"

	"github.com/hashicorp/raft"
)

// FSM impletation of FSM interface provided by hashicorp/raftNode
type FSM struct {
	ctx *KVStoreContext
	log *log.Logger
}

// LogEntry data structure of KV
type LogEntry struct {
	Key   string
	Value string
}

// Apply applies a Raft log entry to the key-value store.
func (fsm *FSM) Apply(logEntry *raft.Log) interface{} {
	e := LogEntry{}
	if err := json.Unmarshal(logEntry.Data, &e); err != nil {
		panic("Failed unmarshaling Raft log entry. This is a bug.")
	}
	// KV数据存储到本地缓存
	ret := fsm.ctx.store.cache.Set(e.Key, e.Value)
	fsm.log.Printf("fms.Apply(), logEntry:%s, ret:%v\n", logEntry.Data, ret)
	// 返回value
	return ret
}

// Snapshot returns a latest Snapshot
func (fsm *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &Snapshot{cm: fsm.ctx.store.cache}, nil
}

// Restore restore the key-value store to a previous state.
func (fsm *FSM) Restore(serialized io.ReadCloser) error {
	return fsm.ctx.store.cache.UnMarshal(serialized)
}
