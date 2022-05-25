package geeCache

// 抽象出来的只读数据结构 byteview，用来表示缓存值
// 实现了Value interface
type ByteView struct {
	data []byte
}

// 返回数据的长度
func (b ByteView) Len() int {
	return len(b.data)
}

// 返回只读的数据副本
func (b ByteView) CopyData() []byte {
	return cloneBytes(b.data)
}

func cloneBytes(data []byte) []byte {
	cpData := make([]byte, len(data))
	copy(cpData, data)
	return cpData
}

// 将data用string的形式返回回去
func (b ByteView) String() string {

	return string(b.data)
}
