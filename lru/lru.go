package lru

import (
	"container/list"
)

/*
lru算法：最近最少使用算法，如果数据近期被访问过，那么就认为该数据以后会被访问到的概率也越大
数据结构
双向队列：保存数据
hash表：根据key来查找双向列表中的数据value
*/

type Cache struct {
	maxBytes int64 // 最大内存
	nbytes int64 // 已使用内存
	ll *list.List // 双向链表（定义：双向链表头为队尾，尾为队头）
	cache map[string]*list.Element // hash表，Element类型代表是双向链表的一个元素
	// 记录（entry）被移除时的回调函数
	OnEvicted func(key string, value Value)
}

// 双向链表的数据类型
type entry struct {
	key string // hash表的key，删除entry的时候，需要把hash表对应的映射删除
	value Value
}

type Value interface {
	Len() int
}

// 实例化
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:maxBytes,
		ll:list.New(),
		cache:make(map[string]*list.Element),
		OnEvicted:onEvicted,
	}
}

// 查找
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 移动到队尾（将元素e移动到链表的第一个位置）
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}

	return
}

// 删除
func (c *Cache) RemoveOldest() {
	// 返回链表最后一个元素
	ele := c.ll.Back()
	if ele != nil {
		/*
		// 删除链表元素
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		// 维护内存已使用长度
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
		*/
		kv := ele.Value.(*entry)
		c.Remove(kv.key)
	}
}

// 新增/修改
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		// 修改
		// 移动到队尾
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 维护内存已使用长度
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		// 更新值
		kv.value = value
	} else {
		// 新增
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		// 维护内存已使用长度（key+value）
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		// 如果内存超了，那么就淘汰旧数据
		c.RemoveOldest()
	}
}

// 删除缓存
func (c *Cache) Remove(key string) {
	if ele, ok := c.cache[key]; ok {
		// 删除链表元素
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		// 维护内存已使用长度
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 获取当前链表长度
func (c *Cache) Len() int {
	return c.ll.Len()
}
