// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cache implements a LRU cache.
//
// The implementation borrows heavily from SmallLRUCacheKeyUint64
// (originally by Nathan Schrenk). The object maintains a doubly-linked list of
// elements. When an element is accessed, it is promoted to the head of the
// list. When space is needed, the element at the tail of the list
// (the least recently used element) is evicted.
package lru

import (
	"fmt"
	key "github.com/0studio/storage_key"
)

// ShardLRUCacheKeyUint64 is a typical LRU cache implementation.  If the cache
// reaches the capacity, the least recently used item is deleted from
// the cache. Note the capacity is not the number of items, but the
// total sum of the Size() of each item.
type ShardLRUCacheKeyUint64 struct {
	shardCount int
	cachelist  []*LRUCacheKeyUint64
}

// NewLRUCacheKeyUint64 creates a new empty cache with the given capacity.
func NewShardLRUCacheKeyUint64(shardCount int, capacity int64) *ShardLRUCacheKeyUint64 {
	if shardCount < 1 {
		shardCount = 1
	}
	var shardCap int64 = capacity / int64(shardCount)
	var leftCap int64 = capacity - shardCap*int64(shardCount)

	c := &ShardLRUCacheKeyUint64{shardCount: shardCount, cachelist: make([]*LRUCacheKeyUint64, shardCount)}
	for i := 0; i < shardCount; i++ {
		if i == shardCount-1 {
			c.cachelist[i] = NewLRUCacheKeyUint64(shardCap + leftCap)
		} else {
			c.cachelist[i] = NewLRUCacheKeyUint64(shardCap)
		}

	}

	return c
}

func (lru *ShardLRUCacheKeyUint64) GetShard(k key.KeyUint64) *LRUCacheKeyUint64 {
	idx := k.ToSum() % lru.shardCount
	return lru.cachelist[idx]
}

// Get returns a value from the cache, and marks the keyuint64Entry as most
// recently used.
func (lru *ShardLRUCacheKeyUint64) Get(k key.KeyUint64) (v Cacheable, ok bool) {
	return lru.GetShard(k).Get(k)
}

// Set sets a value in the cache.
func (lru *ShardLRUCacheKeyUint64) Set(k key.KeyUint64, value Cacheable) {
	lru.GetShard(k).Set(k, value)
}

// SetIfAbsent will set the value in the cache if not present. If the
// value exists in the cache, we don't set it.
func (lru *ShardLRUCacheKeyUint64) SetIfAbsent(k key.KeyUint64, value Cacheable) {
	lru.GetShard(k).SetIfAbsent(k, value)
}

// Delete removes an keyuint64Entry from the cache, and returns if the keyuint64Entry existed.
func (lru *ShardLRUCacheKeyUint64) Delete(k key.KeyUint64) bool {
	return lru.GetShard(k).Delete(k)
}

// Clear will clear the entire cache.
func (lru *ShardLRUCacheKeyUint64) Clear() {
	for idx, _ := range lru.cachelist {
		lru.cachelist[idx].Clear()
	}
}

// SetCapacity will set the capacity of the cache. If the capacity is
// smaller, and the current cache size exceed that capacity, the cache
// will be shrank.
func (lru *ShardLRUCacheKeyUint64) SetCapacity(capacity int64) {
	var shardCap int64 = capacity / int64(lru.shardCount)
	var leftCap int64 = capacity - shardCap*int64(lru.shardCount)

	for i := 0; i < lru.shardCount; i++ {
		if i == lru.shardCount-1 {
			lru.cachelist[i].SetCapacity(shardCap + leftCap)
		} else {
			lru.cachelist[i].SetCapacity(shardCap)
		}

	}

}
func (lru *ShardLRUCacheKeyUint64) OnMiss(onMiss OnMissHandlerKeyUint64) {
	for idx, _ := range lru.cachelist {
		lru.cachelist[idx].OnMiss(onMiss)
	}
}

// Stats
func (lru *ShardLRUCacheKeyUint64) Stats() (length, size, capacity int64) {
	for idx, _ := range lru.cachelist {
		l, s, c := lru.cachelist[idx].Stats()
		length += l
		size += s
		capacity += c
	}
	return
}

// StatsJSON returns stats as a JSON object in a key.KeyUint64.
func (lru *ShardLRUCacheKeyUint64) StatsJSON() string {
	if lru == nil {
		return "{}"
	}
	l, s, c := lru.Stats()
	return fmt.Sprintf("{\"Length\": %v, \"Size\": %v, \"Capacity\": %v }", l, s, c)
}

// Length returns how many elements are in the cache
func (lru *ShardLRUCacheKeyUint64) Length() (length int64) {
	for idx, _ := range lru.cachelist {
		l := lru.cachelist[idx].Length()
		length += l
	}
	return
}

// Size returns the sum of the objects' Size() method.
func (lru *ShardLRUCacheKeyUint64) Size() (size int64) {
	for idx, _ := range lru.cachelist {
		s := lru.cachelist[idx].Size()
		size += s
	}
	return
}

// Capacity returns the cache maximum capacity.
func (lru *ShardLRUCacheKeyUint64) Capacity() (capacity int64) {
	for idx, _ := range lru.cachelist {
		c := lru.cachelist[idx].Size()
		capacity += c
	}
	return
}

// Keys returns all the ks for the cache, ordered from most recently
// used to last recently used.
func (lru *ShardLRUCacheKeyUint64) Keys() (ks key.KeyUint64List) {
	ks = make([]key.KeyUint64, 0, lru.Length()+int64(lru.shardCount))
	for idx, _ := range lru.cachelist {
		tmp := lru.cachelist[idx].Keys()
		ks = append(ks, tmp...)

	}
	return ks
}

// Items returns all the values for the cache, ordered from most recently
// used to last recently used.
func (lru *ShardLRUCacheKeyUint64) Items() (items []KeyUint64Item) {
	items = make([]KeyUint64Item, 0, lru.Length()+int64(lru.shardCount))
	for idx, _ := range lru.cachelist {
		tmp := lru.cachelist[idx].Items()
		items = append(items, tmp...)

	}
	return items
}

func (lru *ShardLRUCacheKeyUint64) Values() []Cacheable {
	values := make([]Cacheable, 0, lru.Length()+int64(lru.shardCount))
	for idx, _ := range lru.cachelist {
		tmp := lru.cachelist[idx].Values()
		values = append(values, tmp...)

	}
	return values
}
