package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/hashicorp/raft"
)

const (
	ENABLE_WRITE_TRUE  = int32(1)
	ENABLE_WRITE_FALSE = int32(0)
)

// http服务器
type HttpServer struct {
	ctx         *KVStoreContext // 本地缓存上下文
	log         *log.Logger
	mux         *http.ServeMux // 多路复用器
	enableWrite int32          // 是否接受写请求，只有leader可写
}

func NewHttpServer(ctx *KVStoreContext, log *log.Logger) *HttpServer {
	mux := http.NewServeMux()
	s := &HttpServer{
		ctx:         ctx,
		log:         log,
		mux:         mux,
		enableWrite: ENABLE_WRITE_FALSE,
	}

	mux.HandleFunc("/set", s.doSet)
	mux.HandleFunc("/get", s.doGet)
	mux.HandleFunc("/join", s.doJoin)
	return s
}

func (h *HttpServer) checkWritePermission() bool {
	return atomic.LoadInt32(&h.enableWrite) == ENABLE_WRITE_TRUE
}

func (h *HttpServer) setWriteFlag(flag bool) {
	if flag {
		atomic.StoreInt32(&h.enableWrite, ENABLE_WRITE_TRUE)
	} else {
		atomic.StoreInt32(&h.enableWrite, ENABLE_WRITE_FALSE)
	}
}

func (h *HttpServer) doGet(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()

	key := vars.Get("key")
	if key == "" {
		h.log.Println("doGet() error, get nil key")
		fmt.Fprint(w, "")
		return
	}

	// 本地缓存 根据key读取数据
	ret := h.ctx.store.cache.Get(key)
	fmt.Fprintf(w, "%s\n", ret)
}

// doSet saves data to cache, only raftNode master node provides this api
func (h *HttpServer) doSet(w http.ResponseWriter, r *http.Request) {
	if !h.checkWritePermission() {
		fmt.Fprint(w, "write method not allowed\n")
		return
	}
	vars := r.URL.Query()

	key := vars.Get("key")
	value := vars.Get("value")
	if key == "" || value == "" {
		h.log.Println("doSet() error, get nil key or nil value")
		fmt.Fprint(w, "param error\n")
		return
	}

	// KV包装成log entry，并序列化
	event := LogEntry{Key: key, Value: value}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		h.log.Printf("json.Marshal failed, err:%v", err)
		fmt.Fprint(w, "internal error\n")
		return
	}

	// 提交日志（包含KV数据）
	applyFuture := h.ctx.store.raftNode.raft.Apply(eventBytes, 5*time.Second)
	// 提交日志失败的操作
	if err := applyFuture.Error(); err != nil {
		h.log.Printf("raftNode.Apply failed:%v", err)
		fmt.Fprint(w, "internal error\n")
		return
	}
	// 提交日志成功的操作
	fmt.Fprintf(w, "ok\n")
}

// doJoin handles joining cluster request
func (h *HttpServer) doJoin(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()

	peerAddress := vars.Get("peerAddress")
	if peerAddress == "" {
		h.log.Println("invalid PeerAddress")
		fmt.Fprint(w, "invalid peerAddress\n")
		return
	}
	// leader节点添加voter节点
	addPeerFuture := h.ctx.store.raftNode.raft.AddVoter(raft.ServerID(peerAddress), raft.ServerAddress(peerAddress), 0, 0)
	if err := addPeerFuture.Error(); err != nil {
		h.log.Printf("Error joining peer to raftNode, peeraddress:%s, err:%v, code:%d", peerAddress, err, http.StatusInternalServerError)
		fmt.Fprint(w, "internal error\n")
		return
	}
	fmt.Fprint(w, "ok")
}
