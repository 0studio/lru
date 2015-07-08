// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cache implements a LRU cache.
//
// The implementation borrows heavily from SmallLRUCacheInt
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

// IntItem is what is stored in the cache
type IntItem struct {
	Key   int
	Value Cacheable
}

type OnMissHandlerInt func(k int) (Cacheable, bool)

// LRUCacheInt is a typical LRU cache implementation.  If the cache
// reaches the capacity, the least recently used item is deleted from
// the cache. Note the capacity is not the number of items, but the
// total sum of the Size() of each item.
type LRUCacheInt struct {
	mu sync.Mutex

	// list & table of *intEntry objects
	list  *list.List
	table map[int]*list.Element

	// Our current size. Obviously a gross simplification and
	// low-grade approximation.
	size int64

	// How much we are limiting the cache to.
	capacity int64
	onMiss   OnMissHandlerInt
}
type intEntry struct {
	key   int
	value Cacheable
	size  int64
}

// NewLRUCacheInt creates a new empty cache with the given capacity.
func NewLRUCacheInt(capacity int64) *LRUCacheInt {
	return &LRUCacheInt{
		list:     list.New(),
		table:    make(map[int]*list.Element),
		capacity: capacity,
	}
}

// Get returns a value from the cache, and marks the intEntry as most
// recently used.
func (lru *LRUCacheInt) Get(k int) (v Cacheable, ok bool) {
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
	return element.Value.(*intEntry).value, true
}

// Set sets a value in the cache.
func (lru *LRUCacheInt) Set(k int, value Cacheable) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	lru.set(k, value)
}
func (lru *LRUCacheInt) set(k int, value Cacheable) {
	if element := lru.table[k]; element != nil {
		lru.updateInplace(element, value)
	} else {
		lru.addNew(k, value)
	}
}

// SetIfAbsent will set the value in the cache if not present. If the
// value exists in the cache, we don't set it.
func (lru *LRUCacheInt) SetIfAbsent(k int, value Cacheable) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if element := lru.table[k]; element != nil {
		lru.moveToFront(element)
	} else {
		lru.addNew(k, value)
	}
}

// Delete removes an intEntry from the cache, and returns if the intEntry existed.
func (lru *LRUCacheInt) Delete(k int) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[k]
	if element == nil {
		return false
	}

	lru.list.Remove(element)
	delete(lru.table, k)
	lru.size -= element.Value.(*intEntry).size
	safeOnPurge(element.Value.(*intEntry).value, PURGE_REASON_DELETE)
	return true
}

// Clear will clear the entire cache.
func (lru *LRUCacheInt) Clear() {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	for e := lru.list.Front(); e != nil; e = e.Next() {
		safeOnPurge(e.Value.(*intEntry).value, PURGE_REASON_CLEAR_ALL)
	}

	lru.list.Init()
	lru.table = make(map[int]*list.Element)
	lru.size = 0
}

// SetCapacity will set the capacity of the cache. If the capacity is
// smaller, and the current cache size exceed that capacity, the cache
// will be shrank.
func (lru *LRUCacheInt) SetCapacity(capacity int64) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.capacity = capacity
	lru.checkCapacity()
}
func (lru *LRUCacheInt) OnMiss(onMiss OnMissHandlerInt) {
	lru.onMiss = onMiss
}

// Stats
func (lru *LRUCacheInt) Stats() (length, size, capacity int64) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	// if lastElem := lru.list.Back(); lastElem != nil {
	// 	oldest = lastElem.Value.(*intEntry).time_accessed
	// }
	return int64(lru.list.Len()), lru.size, lru.capacity
}

// StatsJSON returns stats as a JSON object in a int.
func (lru *LRUCacheInt) StatsJSON() string {
	if lru == nil {
		return "{}"
	}
	l, s, c := lru.Stats()
	return fmt.Sprintf("{\"Length\": %v, \"Size\": %v, \"Capacity\": %v }", l, s, c)
}

// Length returns how many elements are in the cache
func (lru *LRUCacheInt) Length() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return int64(lru.list.Len())
}

// Size returns the sum of the objects' Size() method.
func (lru *LRUCacheInt) Size() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.size
}

// Capacity returns the cache maximum capacity.
func (lru *LRUCacheInt) Capacity() int64 {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.capacity
}

// Keys returns all the ks for the cache, ordered from most recently
// used to last recently used.
func (lru *LRUCacheInt) Keys() []int {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	ks := make([]int, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		ks = append(ks, e.Value.(*intEntry).key)
	}
	return ks
}

// Items returns all the values for the cache, ordered from most recently
// used to last recently used.
func (lru *LRUCacheInt) Items() []IntItem {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	items := make([]IntItem, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		v := e.Value.(*intEntry)
		items = append(items, IntItem{Key: v.key, Value: v.value})
	}
	return items
}

func (lru *LRUCacheInt) Values() []Cacheable {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	values := make([]Cacheable, 0, lru.list.Len())
	for e := lru.list.Front(); e != nil; e = e.Next() {
		v := e.Value.(*intEntry)
		values = append(values, v.value)
	}
	return values
}
func (lru *LRUCacheInt) updateInplace(element *list.Element, value Cacheable) {
	valueSize := getSize(value)
	sizeDiff := valueSize - element.Value.(*intEntry).size
	safeOnPurge(element.Value.(*intEntry).value, PURGE_REASON_UPDATE)
	element.Value.(*intEntry).value = value
	element.Value.(*intEntry).size = valueSize
	lru.size += sizeDiff
	lru.moveToFront(element)
	lru.checkCapacity()
}

func (lru *LRUCacheInt) moveToFront(element *list.Element) {
	lru.list.MoveToFront(element)
}

func (lru *LRUCacheInt) addNew(k int, value Cacheable) {
	newEntry := &intEntry{k, value, getSize(value)}
	element := lru.list.PushFront(newEntry)
	lru.table[k] = element
	lru.size += newEntry.size
	lru.checkCapacity()
}

func (lru *LRUCacheInt) checkCapacity() {
	// Partially duplicated from Delete
	for lru.size > lru.capacity {
		delElem := lru.list.Back()
		delValue := delElem.Value.(*intEntry)
		lru.list.Remove(delElem)
		delete(lru.table, delValue.key)
		lru.size -= delValue.size
		safeOnPurge(delValue.value, PURGE_REASON_CACHEFULL)
	}
}
