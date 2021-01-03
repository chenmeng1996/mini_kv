package main

import (
	"flag"
)

// configuration
type Options struct {
	dataDir        string // director location of local storage
	httpAddress    string // http server address
	raftTCPAddress string // address for communication between raftNode nodes
	bootstrap      bool   // start as master or not
	joinAddress    string // leader raft node address to join
}

func NewOptions() *Options {
	opts := &Options{}

	var httpAddress = flag.String("http", "127.0.0.1:6000", "Http address")
	var raftTCPAddress = flag.String("raft", "127.0.0.1:7000", "raftNode tcp address")
	var node = flag.String("node", "node1", "raftNode node name")
	var bootstrap = flag.Bool("bootstrap", false, "start as raftNode cluster")
	var joinAddress = flag.String("join", "", "join address for raftNode cluster")
	flag.Parse()

	opts.dataDir = "./" + *node
	opts.httpAddress = *httpAddress
	opts.bootstrap = *bootstrap
	opts.raftTCPAddress = *raftTCPAddress
	opts.joinAddress = *joinAddress
	return opts
}
