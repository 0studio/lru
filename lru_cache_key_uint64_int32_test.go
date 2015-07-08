// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lru

import (
	"encoding/json"
	key "github.com/0studio/storage_key"
	"testing"
)

func TestKeyUint64Int32InitialState(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(5)
	l, sz, c := cache.Stats()
	if l != 0 {
		t.Errorf("length = %v, want 0", l)
	}
	if sz != 0 {
		t.Errorf("size = %v, want 0", sz)
	}
	if c != 5 {
		t.Errorf("capacity = %v, want 5", c)
	}
}

func TestKeyUint64Int32SetInsertsValue(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	data := &CacheValue{0}
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	cache.Set(k, data)

	v, ok := cache.Get(k)
	if !ok || v.(*CacheValue) != data {
		t.Errorf("Cache has incorrect value: %v != %v", data, v)
	}

	keys := cache.Keys()
	if len(keys) != 1 || keys[0] != k {
		t.Errorf("Cache.Keys() returned incorrect items: %v", k)
	}
	items := cache.Items()
	if len(items) != 1 || items[0].Key != k {
		t.Errorf("Cache.Values() returned incorrect items: %v", items)
	}
	values := cache.Values()
	if len(values) != 1 {
		t.Errorf("Cache.Values() returned incorrect values: %v", values)
	}

}

func TestKeyUint64Int32SetIfAbsent(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	data := &CacheValue{0}
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	cache.SetIfAbsent(k, data)

	v, ok := cache.Get(k)
	if !ok || v.(*CacheValue) != data {
		t.Errorf("Cache has incorrect value: %v != %v", data, v)
	}

	cache.SetIfAbsent(k, &CacheValue{1})

	v, ok = cache.Get(k)
	if !ok || v.(*CacheValue) != data {
		t.Errorf("Cache has incorrect value: %v != %v", data, v)
	}
}

func TestKeyUint64Int32GetValueWithMultipleTypes(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	data := &CacheValue{0}
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	cache.Set(k, data)

	v, ok := cache.Get(k)
	if !ok || v.(*CacheValue) != data {
		t.Errorf("Cache has incorrect value for \"k\": %v != %v", data, v)
	}

	// v, ok = cache.Get(string([]byte{'k', 'e', 'y'}))
	// if !ok || v.(*CacheValue) != data {
	// 	t.Errorf("Cache has incorrect value for []byte {'k','e','y'}: %v != %v", data, v)
	// }
}

func TestKeyUint64Int32SetUpdatesSize(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	emptyValue := &CacheValue{0}
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	cache.Set(k, emptyValue)
	if _, sz, _ := cache.Stats(); sz != 0 {
		t.Errorf("cache.Size() = %v, expected 0", sz)
	}
	someValue := &CacheValue{20}
	k = key.NewKeyUint64Int32(2, 0)
	cache.Set(k, someValue)
	if _, sz, _ := cache.Stats(); sz != 20 {
		t.Errorf("cache.Size() = %v, expected 20", sz)
	}
}

func TestKeyUint64Int32SetWithOldKeyUpdatesValue(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	emptyValue := &CacheValue{0}
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	cache.Set(k, emptyValue)
	someValue := &CacheValue{20}
	cache.Set(k, someValue)

	v, ok := cache.Get(k)
	if !ok || v.(*CacheValue) != someValue {
		t.Errorf("Cache has incorrect value: %v != %v", someValue, v)
	}
}

func TestKeyUint64Int32SetWithOldKeyUpdatesSize(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	emptyValue := &CacheValue{0}
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	cache.Set(k, emptyValue)

	if _, sz, _ := cache.Stats(); sz != 0 {
		t.Errorf("cache.Size() = %v, expected %v", sz, 0)
	}

	someValue := &CacheValue{20}
	cache.Set(k, someValue)
	expected := int64(someValue.size)
	if _, sz, _ := cache.Stats(); sz != expected {
		t.Errorf("cache.Size() = %v, expected %v", sz, expected)
	}
}

func TestKeyUint64Int32GetNonExistent(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)

	if _, ok := cache.Get(k); ok {
		t.Error("Cache returned a crap value after no inserts.")
	}
}

func TestKeyUint64Int32Delete(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	value := &CacheValue{1}
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)

	if cache.Delete(k) {
		t.Error("Item unexpectedly already in cache.")
	}

	cache.Set(k, value)

	if !cache.Delete(k) {
		t.Error("Expected item to be in cache.")
	}

	if _, sz, _ := cache.Stats(); sz != 0 {
		t.Errorf("cache.Size() = %v, expected 0", sz)
	}

	if _, ok := cache.Get(k); ok {
		t.Error("Cache returned a value after deletion.")
	}
}

func TestKeyUint64Int32Clear(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	value := &CacheValue{1}
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)

	cache.Set(k, value)
	cache.Clear()

	if _, sz, _ := cache.Stats(); sz != 0 {
		t.Errorf("cache.Size() = %v, expected 0 after Clear()", sz)
	}
}

func TestKeyUint64Int32CapacityIsObeyed(t *testing.T) {
	size := int64(3)
	cache := NewLRUCacheKeyUint64Int32(100)
	cache.SetCapacity(size)
	value := &CacheValue{1}
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	var k2 key.KeyUint64Int32 = key.NewKeyUint64Int32(2, 0)
	var k3 key.KeyUint64Int32 = key.NewKeyUint64Int32(3, 0)
	var k4 key.KeyUint64Int32 = key.NewKeyUint64Int32(4, 0)

	// Insert up to the cache's capacity.
	cache.Set(k, value)
	cache.Set(k2, value)
	cache.Set(k3, value)
	if _, sz, _ := cache.Stats(); sz != size {
		t.Errorf("cache.Size() = %v, expected %v", sz, size)
	}
	// Insert one more; something should be evicted to make room.
	cache.Set(k4, value)
	if _, sz, _ := cache.Stats(); sz != size {
		t.Errorf("post-evict cache.Size() = %v, expected %v", sz, size)
	}

	// Check json stats
	data := cache.StatsJSON()
	m := make(map[string]interface{})
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Errorf("cache.StatsJSON() returned bad json data: %v %v", data, err)
	}
	if m["Size"].(float64) != float64(size) {
		t.Errorf("cache.StatsJSON() returned bad size: %v", m)
	}

	// Check various other stats
	if l := cache.Length(); l != size {
		t.Errorf("cache.StatsJSON() returned bad length: %v", l)
	}
	if s := cache.Size(); s != size {
		t.Errorf("cache.StatsJSON() returned bad size: %v", s)
	}
	if c := cache.Capacity(); c != size {
		t.Errorf("cache.StatsJSON() returned bad length: %v", c)
	}

	// checks StatsJSON on nil
	cache = nil
	if s := cache.StatsJSON(); s != "{}" {
		t.Errorf("cache.StatsJSON() on nil object returned %v", s)
	}
}

func TestKeyUint64Int32LRUIsEvicted(t *testing.T) {
	size := int64(3)
	cache := NewLRUCacheKeyUint64Int32(size)

	var k0 key.KeyUint64Int32 = key.NewKeyUint64Int32(0, 0)
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	var k2 key.KeyUint64Int32 = key.NewKeyUint64Int32(2, 0)
	var k3 key.KeyUint64Int32 = key.NewKeyUint64Int32(3, 0)
	cache.Set(k, &CacheValue{1})
	cache.Set(k2, &CacheValue{1})
	cache.Set(k3, &CacheValue{1})
	// lru: [k3, k2, k1]

	// Look up the elements. This will rearrange the LRU ordering.
	cache.Get(k3)
	// beforeKey2 := time.Now()
	cache.Get(k2)
	// afterKey2 := time.Now()
	cache.Get(k)
	// lru: [k1, k2, k3]

	cache.Set(k0, &CacheValue{1})
	// lru: [k0, k1, k2]

	// The least recently used one should have been evicted.
	if _, ok := cache.Get(k3); ok {
		t.Error("Least recently used element was not evicted.")
	}

	// // Check oldest
	// if o := cache.Oldest(); o.Before(beforeKey2) || o.After(afterKey2) {
	// 	t.Errorf("cache.Oldest returned an unexpected value: got %v, expected a value between %v and %v", o, beforeKey2, afterKey2)
	// }
}

type PurgeCacheValueKeyUint64Int32 struct {
}

func (cv *PurgeCacheValueKeyUint64Int32) OnPurge(why PurgeReason) {
	purgeReasonFlag4TestKeyUint64Int32 = why
}

var purgeReasonFlag4TestKeyUint64Int32 PurgeReason

func TestKeyUint64Int32DeleteOnPurge(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	value := &PurgeCacheValueKeyUint64Int32{}
	purgeReasonFlag4TestKeyUint64Int32 = PURGE_REASON_CACHEFULL // init
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)

	cache.Set(k, value)
	cache.Delete(k)
	if purgeReasonFlag4TestKeyUint64Int32 != PURGE_REASON_DELETE {
		t.Errorf("after cache.Delete ,purgeReason should be %d ,but get %d", PURGE_REASON_DELETE, purgeReasonFlag4TestKeyUint64Int32)
	}

}

func TestKeyUint64Int32UpdateOnPurge(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(100)
	value := &PurgeCacheValueKeyUint64Int32{}
	purgeReasonFlag4TestKeyUint64Int32 = PURGE_REASON_CACHEFULL // init
	var k key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)

	cache.Set(k, value)
	cache.Set(k, value) // set again
	if purgeReasonFlag4TestKeyUint64Int32 != PURGE_REASON_UPDATE {
		t.Errorf("after cache.Delete ,purgeReason should be %d ,but get %d", PURGE_REASON_UPDATE, purgeReasonFlag4TestKeyUint64Int32)
	}

}

func TestKeyUint64Int32CacheFullOnPurge(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(1)
	value := &PurgeCacheValueKeyUint64Int32{}
	var k1 key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	var k2 key.KeyUint64Int32 = key.NewKeyUint64Int32(2, 0)

	cache.Set(k1, value)                                     // after this cache is full
	purgeReasonFlag4TestKeyUint64Int32 = PURGE_REASON_DELETE // init
	cache.Set(k2, value)                                     // after this k1 is delete and reason set to  PURGE_REASON_CACHEFULL
	if purgeReasonFlag4TestKeyUint64Int32 != PURGE_REASON_CACHEFULL {
		t.Errorf("after cache.Delete ,purgeReason should be %d ,but get %d", PURGE_REASON_CACHEFULL, purgeReasonFlag4TestKeyUint64Int32)
	}

}

func TestKeyUint64Int32ClearOnPurge(t *testing.T) {
	cache := NewLRUCacheKeyUint64Int32(1)
	value := &PurgeCacheValueKeyUint64Int32{}
	var k1 key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	purgeReasonFlag4TestKeyUint64Int32 = PURGE_REASON_DELETE // init
	cache.Set(k1, value)                                     // after this cache is full
	cache.Clear()
	if purgeReasonFlag4TestKeyUint64Int32 != PURGE_REASON_CLEAR_ALL {
		t.Errorf("after cache.Delete ,purgeReason should be %d ,but get %d", PURGE_REASON_CLEAR_ALL, purgeReasonFlag4TestKeyUint64Int32)
	}

}

func TestKeyUint64Int32OnMiss(t *testing.T) {
	fun := func(k key.KeyUint64Int32) (Cacheable, bool) {
		return 1, true
	}
	var k1 key.KeyUint64Int32 = key.NewKeyUint64Int32(1, 0)
	cache := NewLRUCacheKeyUint64Int32(1)
	cache.OnMiss(fun)
	v, ok := cache.Get(k1) //
	if ok != true && v != 1 {
		t.Errorf("lru.onMiss is errror")
	}

}
