package lru

// Reasons for a cached element to be deleted from the cache
type PurgeReason int

const (
	// Cache is growing too large and this is the least used item
	PURGE_REASON_CACHEFULL PurgeReason = iota
	// This item was explicitly deleted using Cache.Delete(id)
	PURGE_REASON_DELETE
	// A new element with the same k is stored (usually indicates an update)
	PURGE_REASON_UPDATE
	// when Cache.Clear() is called
	PURGE_REASON_CLEAR_ALL
)

// Optional interface for cached objects
type OnPurger interface {
	// Called once when the element is purged from cache. The argument
	// indicates why.
	//
	// Example use-case: a session cache where sessions are not stored in a
	// database until they are purged from the memory cache. As long as the
	// memory cache is large enough to hold all of them, they expire before the
	// cache grows too large and no database connection is ever needed. This
	// OnPurge implementation would store items to a database iff reason ==
	// PurgeReason_CACHE_FULL.
	//
	// Called from within a private goroutine, but never called concurrently
	// with other elements' OnPurge(). The entire cache is blocked until this
	// function returns. By all means, feel free to launch a fresh goroutine
	// and return immediately.
	OnPurge(why PurgeReason)
	//
	// To use this library, first create a cache:
	//
	//      c := lru.NewLRUCacheString(1234)
	//
	// Then, optionally, define a type that implements some of the interfaces:
	//
	//      type cacheableInt int
	//
	//      func (i cacheableInt) OnPurge(why lru.PurgeReason) {
	//          fmt.Printf("Purging %d\n", i)
	//      }
	//
	// Finally:
	//
	//     for i := 0; i < 2000; i++ {
	//         c.Set(strconv.Itoa(i), cacheableInt(i))
	//     }
	//
	// This will generate the following output:
	//
	//     Purging 0
	//     Purging 1
	//     ...
	//     Purging 764
	//     Purging 765
	//
}

// Anything can be cached!
type Cacheable interface{}

// Optional interface for cached objects. If this interface is not implemented,
// an element is assumed to have size 1.
type SizeAware interface {
	// See Cache.MaxSize() for an explanation of the semantics. Please report a
	// constant size; the cache does not expect objects to change size while
	// they are cached. Items are trusted to report their own size accurately.
	Size() int
}

func getSize(x Cacheable) int64 {
	if s, ok := x.(SizeAware); ok {
		return int64(s.Size())
	}
	return 1
}

// Only call c.OnPurge() if c implements OnPurger.
func safeOnPurge(c Cacheable, why PurgeReason) {
	if t, ok := c.(OnPurger); ok {
		t.OnPurge(why)
	}
	return
}
