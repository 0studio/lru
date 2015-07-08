# LRU CACHE for golang

## Demo
```
	      c := lru.NewLRUCacheString(1234)

	      onMiss := func(k string) (Cacheable, bool) {
           //load from db
		     return cacheableInt(1), true
	      }
          c.OnMiss(onMiss) // if set
	
	 Then, optionally, define a type that implements some of the interfaces:
	
	      type cacheableInt int
	
	      func (i cacheableInt) OnPurge(why lru.PurgeReason) {
	          fmt.Printf("Purging %d\n", i)
	      }
          
	
	 Finally:
	
	     for i := 0; i < 2000; i++ {
	         c.Set(strconv.Itoa(i), cacheableInt(i))
	     }
	
	 This will generate the following output:
	
	     Purging 0
	     Purging 1
	     ...
	     Purging 764
	     Purging 765
	
```
