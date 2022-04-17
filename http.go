package simpleCache

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"simpleCache/consistenthash"
	pb "simpleCache/proto"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_simplecache/"
	defaultReplicas = 50
)

type HTTPPool struct {
	self string // 自身节点地址
	basePath string // 节点间的通讯地址前缀（用于校验）

	mu sync.Mutex
	peers *consistenthash.Map // 一致性哈希算法的 Map，用来根据具体的 key 选择节点
	httpGetters map[string]*httpGetter // 映射远程节点与对应的 httpGetter,每一个远程节点对应一个 httpGetter，格式其实就是 ip地址:ip地址 + basepath，如127.0.0.1:8001 => 127.0.0.1:2001/_simplecache/
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self: self, // 自身节点地址
		basePath: defaultBasePath, // 节点间的通讯地址前缀（用于校验）
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 处理http请求
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 检查uri前缀
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	p.Log("%s %s", r.Method, r.URL.Path)

	// url格式：/basepath/groupname/key
	// uri去掉p.basePath前缀，只切割分开groupname/key
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: " + groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	/*
	// http原本传输方式
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
	*/

	// 使用protobuf传输
	body, err := proto.Marshal(&pb.SearchResponse{Value:view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// 实例化一致性哈希算法，为传入的节点创建了一个 HTTP 客户端 httpGetter
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil)
	// 添加虚拟节点
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		// 节点ip地址 : 节点ip地址 + basePath（/_simplecache/）
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// 根据具体的key选择节点，返回节点对应的 HTTP 客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("pick peer %s", peer)
		return p.httpGetters[peer], true
	}

	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

// 客户端
type httpGetter struct {
	baseURL string // 将要访问的远程节点的地址，例如 http://example.com/_geecache/
}

func (h *httpGetter) Get (group string, key string) ([]byte, error) {
	// QueryEscape函数对s进行转码使之可以安全的用在URL查询里
	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key))
	// 发送get请求
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

func (h *httpGetter) PbGet (in *pb.SearchRequest, out *pb.SearchResponse) error {
	// QueryEscape函数对s进行转码使之可以安全的用在URL查询里
	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(in.GetGroup()), url.QueryEscape(in.GetKey()))
	// 发送get请求
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

var _ PeerGetter = (*httpGetter) (nil)