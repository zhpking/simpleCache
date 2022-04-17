package simpleCache

import (
	"fmt"
	"log"
	"simpleCache/proto"
	"simpleCache/singleflight"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 负责与用户的交互，并且控制缓存值存储和获取的流程
// Group相当于一个数据库，name代码数据库名
type Group struct {
	name string
	getter Getter // 如果缓存不存在，获取数据的途径（如数据库）
	mainCache cache // 缓存

	peers PeerPicker // 哈希环上的节点

	loader *singleflight.Group // 防止cache的缓存击穿，在并发的情况下保证每个key只从数据源（如db）获取一次数据
}

var (
	mu sync.RWMutex
	groups = make(map[string]*Group)
)

// 创建Group实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name: name,
		getter: getter,
		mainCache:cache{cacheBytes: cacheBytes},

		loader:&singleflight.Group{},
	}

	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		// 在缓存，直接返回
		log.Println("[SimpleCache] hit")
		return v, nil
	}

	// 不在缓存，则从数据源重新加载数据或者其他远程节点中获取数据
	return g.load(key)
}

// 根据key从数据源或远程节点获取数据
func (g *Group) load(key string) (value ByteView, err error) {
	/*
	if g.peers != nil {
		// 远程获取数据
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[SimpleCache] Failed to get from peer", err)
		}
	}

	// 从数据源获取数据
	return g.getLocally(key)
	*/

	f := func() (interface{}, error) {
		if g.peers != nil {
			// 远程获取数据
			if peer, ok := g.peers.PickPeer(key); ok {
				// http原本传输方式
				// if value, err = g.getFromPeer(peer, key); err == nil {
				// 使用protobuf传输
				if value, err = g.getFromProtobufPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[SimpleCache] Failed to get from peer", err)
			}
		}

		// 从数据源获取数据
		return g.getLocally(key)
	}

	viewi, err := g.loader.Do(key, f)

	if err == nil {
		return viewi.(ByteView), nil
	}

	return
}

// 调用用户传入的自定义getter方法，从源数据（如db）中获取数据
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)}
	// 把获取到的数据添加到缓存中
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	// 校验注册节点
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 访问远程节点获取数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}

	return ByteView{b: bytes}, nil
}

func (g *Group) getFromProtobufPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &proto.SearchRequest{
		Group: g.name,
		Key: key,
	}

	res := &proto.SearchResponse{}
	err := peer.PbGet(req, res)
	if err != nil {
		return ByteView{}, err
	}

	return ByteView{b: res.Value}, nil
}
