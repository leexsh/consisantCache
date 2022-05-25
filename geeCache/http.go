package geeCache

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"leexsh/gee/geeCache/PeerPicker"
	"leexsh/gee/geeCache/consistanthash"
	pb "leexsh/gee/geeCache/geecachepb/geecachepb"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

type HTTPPool struct {
	self       string
	basePath   string
	mu         sync.Mutex
	peers      *consistanthash.Map    // 一致性hash的map
	httpGetter map[string]*HTTPGetter // 远程节点的映射，一个远程节点 对应一个映射函数 // keyed by e.g. "http://10.0.0.2:8008"  远程节点与baseURL来进行区分
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (h *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", h.self, fmt.Sprintf(format, v...))
}

// http handler
func (h *HTTPPool) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !strings.HasPrefix(request.URL.Path, h.basePath) {
		panic("HTTPPool serving unexpected path: " + request.URL.Path)
	}

	h.Log("%s, %s", request.Method, request.URL.Path)
	parts := strings.SplitN(request.URL.Path[len(h.basePath):], "/", 2)
	fmt.Println("parts: ", len(parts))
	fmt.Println("URL: ", request.URL.Path)

	if len(parts) != 2 {
		http.Error(writer, "bad request ", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(writer, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	data, err := group.Get(key)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	body, err := proto.Marshal(&pb.Response{Value: data.CopyData()})
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Write(body)
}

// 传入所有的http节点 并为每个节点创建了一个http客户端 httpGetter
func (h *HTTPPool) Set(peers ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peers = consistanthash.New(defaultReplicas, nil)
	h.peers.Add(peers...)
	h.httpGetter = make(map[string]*HTTPGetter, len(peers))
	for _, peer := range peers {
		h.httpGetter[peer] = &HTTPGetter{baseURL: peer + h.basePath}
	}
}

//  用于远程节点的选择
func (h *HTTPPool) PickPeer(key string) (peer PeerPicker.PeerGetter, ok bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if peer := h.peers.Get(key); peer != "" && peer != h.self {
		h.Log("Pick peer %s", peer)
		return h.httpGetter[peer], true
	}
	return nil, false
}

// 用于判定HTTPPool是否实现了interface PeerPicker
var _ PeerPicker.PeerPicker = (*HTTPPool)(nil)

type HTTPGetter struct {
	baseURL string //baseURL 表示将要访问的远程节点的地址，例如 http://example.com/_geecache/。
}

// 类似于openAPI 使用 http.Get() 方式获取返回值，并转换为 []bytes 类型。
func (h *HTTPGetter) Get(in *pb.Request, out *pb.Response) error {
	// url.QueryEscape  将string转为url可使用的字符串
	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(in.GetGroup()), url.QueryEscape(in.GetKey()))
	fmt.Println("HTTPGetter, url is: ", u)
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response bodyd: %v", err)
	}
	if err = proto.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	return nil
}

var _ PeerPicker.PeerGetter = (*HTTPGetter)(nil)
