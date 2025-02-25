package cache

import (
	"sync"
	"time"
)

type item struct {
	value      interface{}
	expiration int64
	staleTime  int64
}

type Cache struct {
	items      map[string]item
	mu         sync.RWMutex
	refreshing sync.Map
}

func New() *Cache {
	cache := &Cache{
		items: make(map[string]item),
	}
	go cache.startCleanup()
	return cache
}

func (c *Cache) Set(key string, value interface{}, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	c.items[key] = item{
		value:      value,
		expiration: now + duration.Nanoseconds(),
		staleTime:  now + (duration * 2).Nanoseconds(), // Stale time is double the normal duration
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	now := time.Now().UnixNano()
	if now > item.staleTime {
		return nil, false
	}

	return item.value, true
}

func (c *Cache) GetOrRefresh(key string, refreshFn func() (interface{}, error), duration time.Duration) (interface{}, bool) {
	// Try to get from cache first
	if value, exists := c.Get(key); exists {
		now := time.Now().UnixNano()
		item := c.items[key]

		// If the value is expired but not stale, trigger refresh in background
		if now > item.expiration && now <= item.staleTime {
			if _, loading := c.refreshing.LoadOrStore(key, true); !loading {
				go func() {
					defer c.refreshing.Delete(key)
					if newValue, err := refreshFn(); err == nil {
						c.Set(key, newValue, duration)
					}
				}()
			}
		}
		return value, true
	}

	// If no value exists, do a blocking refresh
	value, err := refreshFn()
	if err != nil {
		return nil, false
	}

	c.Set(key, value, duration)
	return value, true
}

func (c *Cache) startCleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now().UnixNano()
		for k, v := range c.items {
			if now > v.staleTime {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
