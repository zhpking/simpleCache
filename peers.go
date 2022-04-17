package simpleCache

import "simpleCache/proto"

type PeerPicker interface {
	// 根据缓存key值，返回对应的服务节点
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// 根据数据库(group)，查询返回缓存key对应的value值
	Get(group string, key string) ([]byte, error)
	PbGet(in *proto.SearchRequest, out *proto.SearchResponse) error
}


