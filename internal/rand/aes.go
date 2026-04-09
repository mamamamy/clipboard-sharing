package rand

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
	"sync/atomic"
)

type aesInstance struct {
	block cipher.Block
	nonce atomic.Uint64
}

func (i *aesInstance) NewReader() io.Reader {
	nonce := i.nonce.Add(1)
	iv := make([]byte, 16)
	binary.BigEndian.PutUint64(iv, nonce)
	return &aesReader{
		s: cipher.NewCTR(i.block, iv),
	}
}

type aesReader struct {
	s cipher.Stream
}

func (r *aesReader) Read(p []byte) (n int, err error) {
	r.s.XORKeyStream(p, p)
	return len(p), nil
}

func newAESInstance() (Instance, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &aesInstance{
		block: block,
	}, nil
}
