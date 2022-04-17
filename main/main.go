package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"simpleCache"
)

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

// 分布式节点测试
// 创建客户端交互服务
func createGroup() *simpleCache.Group {
	return simpleCache.NewGroup("scores", 2<<10, simpleCache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}

			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 启动缓存节点
func startCacheServer(addr string, addrs []string, simple *simpleCache.Group) {
	peers := simpleCache.NewHTTPPool(addr)
	peers.Set(addrs...)
	simple.RegisterPeers(peers)
	log.Println("simplecache is running at", addr)
	err := http.ListenAndServe(addr[7:], peers)
	log.Fatal(err)
}

// 启动http服务，处理客户端的请求
func startAPIServer(apiAddr string, simple *simpleCache.Group) {
	// 处理http请求方法
	f := func (w http.ResponseWriter, r *http.Request) {
		// http接收请求后响应逻辑
		key := r.URL.Query().Get("key")
		view, err := simple.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view.ByteSlice())
	}
	http.Handle("/api", http.HandlerFunc(f))

	log.Println("fontend server is running at", apiAddr)
	// 监听请求
	err := http.ListenAndServe(apiAddr[7:], nil)
	log.Fatal(err)
}

func distributeTest() {
	var port int
	var api bool
	// ip都是本机地址，用端口控制不同的节点服务
	flag.IntVar(&port, "port", 8001, "SimpleCache server port")
	// 开启api接口，提供服务给客户端
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://127.0.0.1:9999"
	addrMap := map[int]string {
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	simple := createGroup()
	if api {
		go startAPIServer(apiAddr, simple)
	}

	startCacheServer(addrMap[port], addrs, simple)
}

func distributeCacheServiceStart(port int) {
	if port != 0 {
		// 写死几个测试
		addrMap := map[int]string{
			8001: "http://localhost:8001",
			// 8002: "http://localhost:8002",
			// 8003: "http://localhost:8003",
		}

		var addrs []string
		for _, v := range addrMap {
			addrs = append(addrs, v)
		}

		simple := createGroup()
		startCacheServer(addrMap[port], addrs, simple)
	}
}

func distributeHttpServiceStart(api bool) {
	if api {
	apiAddr := "http://127.0.0.1:9999"
	simple := createGroup()
		// 开启api接口，提供服务给客户端
		startAPIServer(apiAddr, simple)
	}
}

func distributeTestV2() {
	apiAddr := "http://127.0.0.1:9999"
	addrMap := map[int]string {
		8001: "http://localhost:8001",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	simple := createGroup()
	go startAPIServer(apiAddr, simple)


	startCacheServer(addrMap[8001], addrs, simple)
}

func main() {
	// httpTest()
	// distributeTest()
	/*
	var api bool
	var port int
	// ip都是本机地址，用端口控制不同的节点服务
	flag.IntVar(&port, "port", 0, "SimpleCache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	distributeHttpServiceStart(api)
	distributeCacheServiceStart(port)
	*/

	// distributeTestV2()

	distributeCacheServiceStart(8001)
}
