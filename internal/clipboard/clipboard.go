package clipboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"golang.design/x/clipboard"
)

type Digest []byte

func (d Digest) String() string {
	return base64.RawURLEncoding.EncodeToString(d)
}

type Kind string

func (k Kind) ClipboardFormat() clipboard.Format {
	switch k {
	case KindImage:
		return clipboard.FmtImage
	case KindText:
		return clipboard.FmtText
	}
	return -1
}

const (
	KindNone  Kind = ""
	KindImage Kind = "IMAGE"
	KindText  Kind = "TEXT"
)

type Info struct {
	Kind   Kind
	Data   []byte
	Digest Digest
}

func Watch(ctx context.Context) chan *Info {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}

	watchTextChan := clipboard.Watch(ctx, clipboard.FmtText)
	watchImageChan := clipboard.Watch(ctx, clipboard.FmtImage)

	c := make(chan *Info)

	go func(chan *Info) {
		var latestDigest Digest
		h := sha256.New()

		for {
			var kind Kind
			var data []byte

			select {
			case <-time.After(1 * time.Second):
				kind, data = Read()
			case data = <-watchTextChan:
				kind = KindText
			case data = <-watchImageChan:
				kind = KindImage
			case <-ctx.Done():
				close(c)
				return
			}

			h.Reset()
			h.Write(data)
			digest := h.Sum(nil)

			if bytes.Equal(digest, latestDigest) {
				continue
			}
			latestDigest = digest

			c <- &Info{
				Kind:   kind,
				Data:   data,
				Digest: digest,
			}
		}
	}(c)
	return c
}

func Write(info *Info) {
	clipboard.Write(info.Kind.ClipboardFormat(), info.Data)
}

func Read() (Kind, []byte) {
	var data []byte

	data = clipboard.Read(clipboard.FmtText)
	if len(data) > 0 {
		return KindText, data
	}

	data = clipboard.Read(clipboard.FmtImage)
	if len(data) > 0 {
		return KindImage, data
	}

	return KindNone, nil
}
