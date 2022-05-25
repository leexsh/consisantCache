package lru

import (
	"container/list"
)

type Cache struct {
	maxBytes  int64                       // 允许使用的最大内存
	nbytes    int64                       // 已经使用的内存
	ll        *list.List                  // cache的双向链表
	cache     map[string]*list.Element    // 为了查询 O(1)
	onEvicted func(key string, val Value) // // 可选并在清除条目时执行。
}

// 链表中的节点
type entry struct {
	key string
	val Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

func NewCache(maxBytes int64, evFunc func(key string, val Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: evFunc,
	}
}

// 缓存中新增元素
func (c *Cache) Add(key string, val Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		// Value是interface的类型，转为entry
		kv := ele.Value.(*entry)
		// 如果key存在缓存中 重新计算用的内存  key不变 不用计算 只需要加上新的val， 再减去旧的val
		c.nbytes += int64(val.Len()) - int64(kv.val.Len())
		kv.val = val
	} else {
		ele := c.ll.PushFront(&entry{key: key, val: val})
		c.cache[key] = ele
		// 如果key不存在缓存中，也要重新计算  len(key) + len(val)
		c.nbytes += int64(len(key)) + int64(val.Len())
	}
	for c.nbytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// 获取key
func (c *Cache) Get(key string) (val Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.val, true
	}
	return
}

// 删除最久不使用的元素
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.val.Len())
		if c.onEvicted != nil {
			// 回调函数
			c.onEvicted(kv.key, kv.val)
		}
	}
}

// 获取链表的长度
func (c *Cache) Len() int {
	return c.ll.Len()
}
