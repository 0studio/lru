// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cache implements a LRU cache.
//
// The implementation borrows heavily from SmallLRUCacheKeyInt32
// (originally by Nathan Schrenk). The object maintains a doubly-linked list of
// elements. When an element is accessed, it is promoted to the head of the
// list. When space is needed, the element at the tail of the list
// (the least recently used element) is evicted.
package lru

import (
	"container/list"
	"fmt"
	key "github.com/0studio/storage_key"
	"sync"
)

// KeyInt32Item is what is stored in the cache
type KeyInt32Item struct {
	Key   key.KeyInt32
	Value Cacheable
}

type OnMissHandlerKeyInt32 func(k key.KeyInt32) (Cacheable, bool)

// LRUCacheKeyInt32 is a typical LRU cache implementation.  If the cache
// reaches the capacity, the least recently used item is deleted from
// the cache. Note the capacity is not the number of items, but the
// total sum of the Size() of each item.
type LRUCacheKeyInt32 struct {
	mu sync.Mutex

	// list & table of *keyint32Entry objects
	list  *list.List
	table map[key.KeyInt32]*list.Element

	// Our current size. Obviously a gross simplification and
	// low-grade approximation.
	size int64

	// How much we are limiting the cache to.
	capacity int64
	onMiss   OnMissHandlerKeyInt32
}
type keyint32Entry struct {
	key   key.KeyInt32
	value Cacheable
	size  int64
}

// NewLRUCacheKeyInt32 creates a new empty cache with the given capacity.
func NewLRUCacheKeyInt32(capacity int64) *LRUCacheKeyInt32 {
	return &LRUCacheKeyInt32{
		list:     list.New(),
		table:    make(map[key.KeyInt32]*list.Element),
		capacity: capacity,
	}
}

// Get returns a value from the cache, and marks the keyint32Entry as most
// recently used.
func (lru *LRUCacheKeyInt32) Get(k key.KeyInt32) (v Cacheable, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[k]
	if element == nil {
		if lru.onMiss == nil {
			return nil, false
		}
		v, ok = lru.onMiss(k)
		if ok { // should check v==nil ???
			lru.set(k, v)
		}
		return
	}
	lru.moveToFront(element)
	return element.Value.(*keyint32Entry).value, true
}

// Set sets a value in the cache.
func (lru *LRUCacheKeyInt32) Set(k key.KeyInt32, value Cacheable) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	lru.set(k, value)
}
func (lru *LRUCacheKeyInt32) set(k key.KeyInt32, value Cacheable) {
	if element := lru.table[k]; element != nil {
		lru.updateInplace(element, value)
	} else {
		lru.addNew(k, value)
	}
}

// SetIfAbsent will set the value in the cache if not present. If the
// value exists in the cache, we don't set it.
func (lru *LRUCacheKeyInt32) SetIfAbsent(k key.KeyInt32, value Cacheable) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if element := lru.table[k]; element != nil {
		lru.moveToFront(element)
	} else {
		lru.addNew(k, value)
	}
}

// Delete removes an keyint32Entry from the cache, and returns if the keyint32Entry existed.
func (lru *LRUCacheKeyInt32) Delete(k key.KeyInt32) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[k]
	if element == nil {
		return false
	}

	lru.list.Remove(element)
	delete(lru.table, k)
	lru.size -= element.Value.(*keyint32Entry).size
	safeOnPurge(element.Value.(*keyint32Entry).value, PURGE_REASON_DELETE)
	return true
}

// Clear will clear the entire cache.
func (lru *LRUCacheKeyInt32) Clear() {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	for e := lru.list.Front(); e != nil; e = e.Next() {
		safeOnPurge(e.Value.(*keyint32Entry).value, PURGE_REASON_CLEAR_ALL)
	}

	lru.list.Init()
	lru.table = make(map[key.KeyInt32]*list.Element)
	lru.size = 0
}

// SetCapacity will set the capacity of the cache. If the capacity is
// smaller, and the current cache size exceed that capacity, the cache
// will be shrank.
func (lru *LRUCacheKeyInt32) SetCapacity(capacity int64) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.capacity = capacity
	lru.checkCapacity()
}
func (lru *LRUCacheKeyInt32) OnMiss(onMiss OnMissHandlerKeyInt32) {
	lru.onMiss = onMiss
}

// Stats
func (lru *LRUCacheKeyInt32) Stats() (length, size, capacity int64) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	// if lastElem := lru.list.Back(); lastElem != nil {
	// 	oldest = lastElem.Value.(*keyint32Entry).time_accessed
	// }
	return int64(lru.list.Len()), lru.size, lru.capacity
}

// StatsJSON returns stats as a JSON object in a key.KeyInt32.
func (lru *LRUCacheKeyInt32) StatsJSON() string {
	if lru == nil {
		return "{}"
	}
	l, s, c := lru.Stats()
	return fmt.Sprintf("{\"Length\": %v, \"Size\": %v, \"Capacity\": %v }", l, s, c)
}

// Length returns how many elements are in the cache
func (lru *LRUCacheKeyInt32) Length() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return int64(lru.list.Len())
}

// Size returns the sum of the objects' Size() method.
func (lru *LRUCacheKeyInt32) Size() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.size
}

// Capacity returns the cache maximum capacity.
func (lru *LRUCacheKeyInt32) Capacity() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.capacity
}

// Keys returns all the ks for the cache, ordered from most recently
// used to last recently used.
func (lru *LRUCacheKeyInt32) Keys() []key.KeyInt32 {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	ks := make([]key.KeyInt32, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		ks = append(ks, e.Value.(*keyint32Entry).key)
	}
	return ks
}

// Items returns all the values for the cache, ordered from most recently
// used to last recently used.
func (lru *LRUCacheKeyInt32) Items() []KeyInt32Item {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	items := make([]KeyInt32Item, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		v := e.Value.(*keyint32Entry)
		items = append(items, KeyInt32Item{Key: v.key, Value: v.value})
	}
	return items
}

func (lru *LRUCacheKeyInt32) Values() []Cacheable {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	values := make([]Cacheable, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		v := e.Value.(*keyint32Entry)
		values = append(values, v.value)
	}
	return values
}
func (lru *LRUCacheKeyInt32) updateInplace(element *list.Element, value Cacheable) {
	valueSize := getSize(value)
	sizeDiff := valueSize - element.Value.(*keyint32Entry).size
	safeOnPurge(element.Value.(*keyint32Entry).value, PURGE_REASON_UPDATE)
	element.Value.(*keyint32Entry).value = value
	element.Value.(*keyint32Entry).size = valueSize
	lru.size += sizeDiff
	lru.moveToFront(element)
	lru.checkCapacity()
}

func (lru *LRUCacheKeyInt32) moveToFront(element *list.Element) {
	lru.list.MoveToFront(element)
}

func (lru *LRUCacheKeyInt32) addNew(k key.KeyInt32, value Cacheable) {
	newEntry := &keyint32Entry{k, value, getSize(value)}
	element := lru.list.PushFront(newEntry)
	lru.table[k] = element
	lru.size += newEntry.size
	lru.checkCapacity()
}

func (lru *LRUCacheKeyInt32) checkCapacity() {
	// Partially duplicated from Delete
	for lru.size > lru.capacity {
		delElem := lru.list.Back()
		delValue := delElem.Value.(*keyint32Entry)
		lru.list.Remove(delElem)
		delete(lru.table, delValue.key)
		lru.size -= delValue.size
		safeOnPurge(delValue.value, PURGE_REASON_CACHEFULL)
	}
}
