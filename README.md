# simpleCache

一个基于key-value的简单的分布式缓存服务系统

simpleCache 是参考groupcache以及兔大的7天go项目的基础上进行修改，此系统用于学习使用，仅仅通过一些简单的测试用例进行测试，**并未在生产环境中使用过**

## 功能特性

- 单机缓存和基于HTTP的分布式缓存

- 支持lru缓存淘汰策略

- 使用锁机制防止缓存击穿

- 通过一致性哈希算法选择节点

## 快速使用

	// 模拟db数据源
	var db = map[string]string {
		"Tom": "630",
		"Jack": "589",
		"Sam": "567",
	}
	
	// 点节点测试
	func httpTest() {
		simpleCache.NewGroup("scores", 2<<10, simpleCache.GetterFunc(
			func(key string) ([]byte, error) {
				log.Println("[SlowDB] search key", key)
				if v, ok := db[key]; ok {
					return []byte(v), nil
				}
	
				return nil, fmt.Errorf("%s not exist", key)
			}))
	
		addr := "localhost:9999"
		peers := simpleCache.NewHTTPPool(addr)
		log.Println("simplecache is running at", addr)
		log.Fatal(http.ListenAndServe(addr, peers))
	}
	
	func main() {
		httpTest()
	}


