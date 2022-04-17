package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash Hash // 自定义hash函数，对缓存数据的key值进行hash运算
	replicas int // 每个节点对应的虚拟节点数
	keys []int // 哈希环
	hashMap map[int]string // 虚拟节点和真实节点的映射(key是虚拟节点名，value是真实节点名)
}

func New (replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash: fn,
		hashMap: make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

// 添加服务器节点（key一般是服务器的ip，也可以自定义服务器名称）
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 生成虚拟节点
		for i := 0; i < m.replicas; i ++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}

	sort.Ints(m.keys)
}

func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {
		// 返回最小索引i满足m.keys[i] >= hash
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx % len(m.keys)]]
}
