// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cache implements a LRU cache.
//
// The implementation borrows heavily from SmallLRUCacheInt64
// (originally by Nathan Schrenk). The object maintains a doubly-linked list of
// elements. When an element is accessed, it is promoted to the head of the
// list. When space is needed, the element at the tail of the list
// (the least recently used element) is evicted.
package lru

import (
	"container/list"
	"fmt"
	"sync"
)

// Int64Item is what is stored in the cache
type Int64Item struct {
	Key   int64
	Value Cacheable
}

type OnMissHandlerInt64 func(k int64) (Cacheable, bool)

// LRUCacheInt64 is a typical LRU cache implementation.  If the cache
// reaches the capacity, the least recently used item is deleted from
// the cache. Note the capacity is not the number of items, but the
// total sum of the Size() of each item.
type LRUCacheInt64 struct {
	mu sync.Mutex

	// list & table of *int64Entry objects
	list  *list.List
	table map[int64]*list.Element

	// Our current size. Obviously a gross simplification and
	// low-grade approximation.
	size int64

	// How much we are limiting the cache to.
	capacity int64
	onMiss   OnMissHandlerInt64
}
type int64Entry struct {
	key   int64
	value Cacheable
	size  int64
}

// NewLRUCacheInt64 creates a new empty cache with the given capacity.
func NewLRUCacheInt64(capacity int64) *LRUCacheInt64 {
	return &LRUCacheInt64{
		list:     list.New(),
		table:    make(map[int64]*list.Element),
		capacity: capacity,
	}
}

// Get returns a value from the cache, and marks the int64Entry as most
// recently used.
func (lru *LRUCacheInt64) Get(k int64) (v Cacheable, ok bool) {
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
	return element.Value.(*int64Entry).value, true
}

// Set sets a value in the cache.
func (lru *LRUCacheInt64) Set(k int64, value Cacheable) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	lru.set(k, value)
}
func (lru *LRUCacheInt64) set(k int64, value Cacheable) {
	if element := lru.table[k]; element != nil {
		lru.updateInplace(element, value)
	} else {
		lru.addNew(k, value)
	}
}

// SetIfAbsent will set the value in the cache if not present. If the
// value exists in the cache, we don't set it.
func (lru *LRUCacheInt64) SetIfAbsent(k int64, value Cacheable) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if element := lru.table[k]; element != nil {
		lru.moveToFront(element)
	} else {
		lru.addNew(k, value)
	}
}

// Delete removes an int64Entry from the cache, and returns if the int64Entry existed.
func (lru *LRUCacheInt64) Delete(k int64) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[k]
	if element == nil {
		return false
	}

	lru.list.Remove(element)
	delete(lru.table, k)
	lru.size -= element.Value.(*int64Entry).size
	safeOnPurge(element.Value.(*int64Entry).value, PURGE_REASON_DELETE)
	return true
}

// Clear will clear the entire cache.
func (lru *LRUCacheInt64) Clear() {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	for e := lru.list.Front(); e != nil; e = e.Next() {
		safeOnPurge(e.Value.(*int64Entry).value, PURGE_REASON_CLEAR_ALL)
	}

	lru.list.Init()
	lru.table = make(map[int64]*list.Element)
	lru.size = 0
}

// SetCapacity will set the capacity of the cache. If the capacity is
// smaller, and the current cache size exceed that capacity, the cache
// will be shrank.
func (lru *LRUCacheInt64) SetCapacity(capacity int64) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.capacity = capacity
	lru.checkCapacity()
}
func (lru *LRUCacheInt64) OnMiss(onMiss OnMissHandlerInt64) {
	lru.onMiss = onMiss
}

// Stats
func (lru *LRUCacheInt64) Stats() (length, size, capacity int64) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	// if lastElem := lru.list.Back(); lastElem != nil {
	// 	oldest = lastElem.Value.(*int64Entry).time_accessed
	// }
	return int64(lru.list.Len()), lru.size, lru.capacity
}

// StatsJSON returns stats as a JSON object in a int64.
func (lru *LRUCacheInt64) StatsJSON() string {
	if lru == nil {
		return "{}"
	}
	l, s, c := lru.Stats()
	return fmt.Sprintf("{\"Length\": %v, \"Size\": %v, \"Capacity\": %v }", l, s, c)
}

// Length returns how many elements are in the cache
func (lru *LRUCacheInt64) Length() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return int64(lru.list.Len())
}

// Size returns the sum of the objects' Size() method.
func (lru *LRUCacheInt64) Size() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.size
}

// Capacity returns the cache maximum capacity.
func (lru *LRUCacheInt64) Capacity() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.capacity
}

// Keys returns all the ks for the cache, ordered from most recently
// used to last recently used.
func (lru *LRUCacheInt64) Keys() []int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	ks := make([]int64, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		ks = append(ks, e.Value.(*int64Entry).key)
	}
	return ks
}

// Items returns all the values for the cache, ordered from most recently
// used to last recently used.
func (lru *LRUCacheInt64) Items() []Int64Item {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	items := make([]Int64Item, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		v := e.Value.(*int64Entry)
		items = append(items, Int64Item{Key: v.key, Value: v.value})
	}
	return items
}

func (lru *LRUCacheInt64) Values() []Cacheable {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	values := make([]Cacheable, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		v := e.Value.(*int64Entry)
		values = append(values, v.value)
	}
	return values
}
func (lru *LRUCacheInt64) updateInplace(element *list.Element, value Cacheable) {
	valueSize := getSize(value)
	sizeDiff := valueSize - element.Value.(*int64Entry).size
	safeOnPurge(element.Value.(*int64Entry).value, PURGE_REASON_UPDATE)
	element.Value.(*int64Entry).value = value
	element.Value.(*int64Entry).size = valueSize
	lru.size += sizeDiff
	lru.moveToFront(element)
	lru.checkCapacity()
}

func (lru *LRUCacheInt64) moveToFront(element *list.Element) {
	lru.list.MoveToFront(element)
}

func (lru *LRUCacheInt64) addNew(k int64, value Cacheable) {
	newEntry := &int64Entry{k, value, getSize(value)}
	element := lru.list.PushFront(newEntry)
	lru.table[k] = element
	lru.size += newEntry.size
	lru.checkCapacity()
}

func (lru *LRUCacheInt64) checkCapacity() {
	// Partially duplicated from Delete
	for lru.size > lru.capacity {
		delElem := lru.list.Back()
		delValue := delElem.Value.(*int64Entry)
		lru.list.Remove(delElem)
		delete(lru.table, delValue.key)
		lru.size -= delValue.size
		safeOnPurge(delValue.value, PURGE_REASON_CACHEFULL)
	}
}
