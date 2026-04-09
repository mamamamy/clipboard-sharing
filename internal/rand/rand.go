package rand

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

const (
	AES = "aes"
)

var DefaultAlgorithm = AES

var ErrUnknownInstanceName = errors.New("rand: unknown instance name")

type Instance interface {
	NewReader() io.Reader
}

var globalInstance Instance
var readerPool sync.Pool

func init() {
	var err error
	globalInstance, err = NewInstance(DefaultAlgorithm)
	if err != nil {
		panic(err)
	}
	readerPool.New = func() any {
		return NewReader()
	}
}

func NewInstance(name string) (Instance, error) {
	switch name {
	case AES:
		return newAESInstance()
	default:
		return nil, ErrUnknownInstanceName
	}
}

func NewReader() io.Reader {
	return globalInstance.NewReader()
}

func Read(b []byte) (int, error) {
	r, ok := readerPool.Get().(io.Reader)
	if !ok {
		panic("rand: not a io.Reader")
	}
	defer readerPool.Put(r)
	n, err := io.ReadFull(r, b)
	if err != nil {
		panic(err)
	}
	return n, nil
}

func Uint64() uint64 {
	var b [8]byte
	_, _ = Read(b[:])
	return binary.BigEndian.Uint64(b[:])
}

func Int63() int64 {
	x := Uint64()
	return int64(x >> 1)
}

func Int63n(n int64) int64 {
	if n <= 0 {
		panic("rand: invalid argument to Int63n")
	}
	if n&(n-1) == 0 {
		return Int63() & (n - 1)
	}
	m := int64((1 << 63) - 1 - (1<<63)%uint64(n))
	v := Int63()
	for v > m {
		v = Int63()
	}
	return v % n
}
