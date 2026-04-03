package enc

import (
	"clipboard/internal/rand"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

var (
	ErrPayloadTooShort = errors.New("payload too short")
)

type Conn struct {
	net.Conn
	key    []byte
	decBuf []byte
}

func NewEncConn(conn net.Conn, key []byte) net.Conn {
	return &Conn{
		Conn: conn,
		key:  key,
	}
}

func (c *Conn) Read(b []byte) (int, error) {
	var n int
	var err error
	if len(c.decBuf) > 0 {
		n = copy(b, c.decBuf)
		c.decBuf = c.decBuf[n:]
		return n, nil
	}
	buf := make([]byte, 4096)
	_, err = io.ReadFull(c.Conn, buf[:2])
	if err != nil {
		return 0, err
	}
	l := binary.BigEndian.Uint16(buf[:2])
	if int(l) > len(buf) {
		buf = make([]byte, l)
	} else if l < 32+16 {
		return 0, ErrPayloadTooShort
	}
	_, err = io.ReadFull(c.Conn, buf[:l])
	if err != nil {
		return 0, err
	}
	buf, err = decrypt(c.key, buf[:l])
	if err != nil {
		return 0, err
	}
	n = copy(b, buf)
	c.decBuf = buf[n:]
	return n, nil
}

func (c *Conn) Write(b []byte) (int, error) {
	var n int
	for len(b) > 0 {
		wn := min(len(b), 4096-2-32-16)
		ciphertext, err := encrypt(c.key, b[:wn])
		if err != nil {
			return n, err
		}
		lb := make([]byte, 2)
		binary.BigEndian.PutUint16(lb, uint16(len(ciphertext)))
		_, err = c.Conn.Write(lb)
		if err != nil {
			return n, err
		}
		_, err = c.Conn.Write(ciphertext)
		if err != nil {
			return n, err
		}
		n += wn
		b = b[wn:]
	}
	return n, nil
}

func encrypt(key, b []byte) ([]byte, error) {
	var err error
	random := make([]byte, 32, 32+len(b)+16)
	_, err = rand.Read(random)
	if err != nil {
		return nil, err
	}
	aead, err := newAEAD(key, random)
	if err != nil {
		return nil, err
	}
	nonce := random[:12]
	return aead.Seal(random, nonce, b, nil), nil
}

func decrypt(key, b []byte) ([]byte, error) {
	var err error
	random := b[:32]
	aead, err := newAEAD(key, random)
	if err != nil {
		return nil, err
	}
	nonce := random[:12]
	ciphertext := b[32:]
	return aead.Open(ciphertext[:0], nonce, ciphertext, nil)
}

func newAEAD(key, random []byte) (cipher.AEAD, error) {
	var err error
	h := hmac.New(sha256.New, key)
	h.Write(random)
	aeadKey := h.Sum(nil)
	block, err := aes.NewCipher(aeadKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
