package geeCache

import (
	"fmt"
	"reflect"
	"testing"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "590",
	"Sam":  "320",
}

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Fatalf("callback fail")
	}
}

// todo
func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	gee := NewGroup("scores", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		if v, ok := db[key]; ok {
			if _, ok := loadCounts[key]; !ok {
				loadCounts[key] = 0
			}
			loadCounts[key]++
			return []byte(v), nil
		}
		return nil, fmt.Errorf("not exist")
	}))

	for k, v := range db {
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		}
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	if data, err := gee.Get("unknow"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", data)
	}
}

func TestGetGroup(t *testing.T) {
	gname := "scores"
	NewGroup(gname, 2<<10, GetterFunc(func(key string) (bytes []byte, err error) {
		return
	}))

	if group := GetGroup(gname); group == nil || group.name != gname {
		t.Fatalf("group %s not exist", gname)
	}

	if group := GetGroup(gname + "11"); group != nil {
		t.Fatalf("expect nil but %s got", gname)
	}
}
