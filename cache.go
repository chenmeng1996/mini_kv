package main

import (
	"encoding/json"
	"io"
	"sync"
)

// local kv cache
type KVCache struct {
	data map[string]string
	sync.RWMutex
}

func NewKVCache() *KVCache {
	kvCache := &KVCache{}
	kvCache.data = make(map[string]string)
	return kvCache
}

func (c *KVCache) Get(key string) string {
	c.RLock()
	ret := c.data[key]
	c.RUnlock()
	return ret
}

func (c *KVCache) Set(key string, value string) error {
	c.Lock()
	defer c.Unlock()
	c.data[key] = value
	return nil
}

// Marshal serializes cache data
func (c *KVCache) Marshal() ([]byte, error) {
	c.RLock()
	defer c.RUnlock()
	dataBytes, err := json.Marshal(c.data)
	return dataBytes, err
}

// UnMarshal deserializes from input(it is disk there), restore local kv cache
func (c *KVCache) UnMarshal(serialized io.ReadCloser) error {
	var newData map[string]string
	if err := json.NewDecoder(serialized).Decode(&newData); err != nil {
		return err
	}

	// similar to copy on write, to improve read performance
	c.Lock()
	defer c.Unlock()
	c.data = newData

	return nil
}
