package lru

import (
	"fmt"
	"testing"
)

type String string

func (s String) Len() int {
	return len(s)
}
func TestGet(t *testing.T) {
	lru := NewCache(int64(1000), nil)
	lru.Add("key1", String("1234"))
	v, ok := lru.Get("key1")
	fmt.Println(v)
	if !ok || String(v.(String)) != "1234" {
		t.Fatalf("cache cannot hited key")
	}
	if _, ok := lru.Get("k2"); ok {
		t.Fatalf("cache miss key2 fail")
	}
}

func TestRemoveOldest(t *testing.T) {
	k1, k2, k3 := "k1", "k2", "k3"
	v1, v2, v3 := "v1", "v2", "v3"
	cap := len(k1 + k2 + v1 + v2)
	lru := NewCache(int64(cap), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))

	if _, ok := lru.Get("k1"); ok || lru.Len() != 2 {
		t.Fatalf("err")
	}
}
