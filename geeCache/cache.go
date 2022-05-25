package geeCache

import (
	"leexsh/gee/geeCache/lru"
	"sync"
)

/*
	对lru的一层封装  支持并发 新增锁
*/
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64 // 缓存的大小
}

// 新增元素
func (c *cache) add(key string, val ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.NewCache(c.cacheBytes, nil)
	}
	c.lru.Add(key, val)
}

// 获取缓存值
func (c *cache) get(key string) (val ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if val, ok := c.lru.Get(key); ok {
		return val.(ByteView), ok
	}
	return
}
