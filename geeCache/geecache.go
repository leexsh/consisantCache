package geeCache

import (
	"fmt"
	"leexsh/gee/geeCache/PeerPicker"
	pb "leexsh/gee/geeCache/geecachepb/geecachepb"
	"leexsh/gee/geeCache/singleflight"
	"log"
	"sync"
)

//	接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
//	|  否                         是
//	|-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
//	|  否
//	|-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶

type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker.PeerPicker
	loader    *singleflight.Group // 确保每个请求只会一次
}

// getter method
type Getter interface {
	Get(key string) ([]byte, error)
}

// 函数式接口
type GetterFunc func(key string) ([]byte, error)

func (g GetterFunc) Get(key string) (bytes []byte, err error) {
	return g(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// create a new group
func NewGroup(name string, cacheBytes int64, getterFunc GetterFunc) *Group {
	if getterFunc == nil {
		panic("nil getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getterFunc,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// get group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	defer mu.RUnlock()
	return g
}

// get Value
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// cache hit
	if v, ok := g.mainCache.get(key); ok {
		log.Printf("get key, key is %s\n", key)
		return v, nil
	}
	// 缓存没命中
	fmt.Println("g.name: ", g.name)
	fmt.Printf("g.mainCache: %#v\n", g.mainCache)
	log.Printf("un hit key is : %s\n", key)
	return g.load(key)
}

func (g *Group) load(key string) (val ByteView, err error) {
	data, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if val, err = g.getFromPeer(peer, key); err == nil {
					return val, nil
				}
				log.Println("[GeeCache] Failed to get from peer, err:", err)
			}
		}
		return g.getLocal(key)
	})
	if err == nil {
		return data.(ByteView), nil
	}
	return
}

func (g *Group) getLocal(key string) (ByteView, error) {
	// 缓存没命中 调用getter方法 去获取val
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	val := ByteView{data: cloneBytes(bytes)}
	// 获取到值后，同时写入缓存
	g.populateCache(key, val)
	return val, nil

}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// register remote node
func (g *Group) RegisterPeers(peers PeerPicker.PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) getFromPeer(peer PeerPicker.PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{data: res.Value}, nil
}
