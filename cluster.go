package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// RaftNode raft node
type RaftNode struct {
	raft           *raft.Raft // raft client
	fsm            *FSM       // fsm of raft log
	leaderNotifyCh chan bool  // notify if it is a master node
}

// configure network communication between raft node
func newRaftTransport(opts *Options) (*raft.NetworkTransport, error) {
	address, err := net.ResolveTCPAddr("tcp", opts.raftTCPAddress)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(address.String(), address, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}
	return transport, nil
}

// create a new raft node
func newRaftNode(opts *Options, ctx *KVStoreContext) (*RaftNode, error) {
	// raft node configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(opts.raftTCPAddress) // raft node id（unique identifier）
	//raftConfig.Logger = log.New(os.Stderr, "raftNode: ", log.Ldate|log.Ltime) // format and localtion of log
	raftConfig.SnapshotInterval = 20 * time.Second // snapshort interval
	raftConfig.SnapshotThreshold = 2               // when more than 2 new log, do snapshot
	leaderNotifyCh := make(chan bool, 1)
	raftConfig.NotifyCh = leaderNotifyCh

	// network communication
	transport, err := newRaftTransport(opts)
	if err != nil {
		return nil, err
	}

	// create disk direction
	if err := os.MkdirAll(opts.dataDir, 0700); err != nil {
		return nil, err
	}

	// fsm
	fsm := &FSM{
		ctx: ctx,
		log: log.New(os.Stderr, "FSM: ", log.Ldate|log.Ltime),
	}
	// snapshot store, here's file store
	snapshotStore, err := raft.NewFileSnapshotStore(opts.dataDir, 1, os.Stderr)
	if err != nil {
		return nil, err
	}

	// log store, here's embedded db store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(opts.dataDir, "raftNode-log.bolt"))
	if err != nil {
		return nil, err
	}

	// kv value store, here's embedded db store
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(opts.dataDir, "raftNode-stable.bolt"))
	if err != nil {
		return nil, err
	}

	// raft node
	raftNode, err := raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, err
	}

	// if raft node is master, initialize the raft cluster
	if opts.bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raftConfig.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		raftNode.BootstrapCluster(configuration)
	}

	return &RaftNode{raft: raftNode, fsm: fsm, leaderNotifyCh: leaderNotifyCh}, nil
}

// joinRaftCluster add a raft node to cluster
func joinRaftCluster(opts *Options) error {
	url := fmt.Sprintf("http://%s/join?peerAddress=%s", opts.joinAddress, opts.raftTCPAddress)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(body) != "ok" {
		return errors.New(fmt.Sprintf("Error joining cluster: %s", body))
	}

	return nil
}
