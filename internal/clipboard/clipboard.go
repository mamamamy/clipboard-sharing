package clipboard

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"golang.design/x/clipboard"
)

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
	Digest string
}

func (i *Info) CalcDigest() string {
	i.Digest = calcDigest(i.Kind, i.Data)
	return i.Digest
}

func Watch(ctx context.Context) chan *Info {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}

	watchTextChan := clipboard.Watch(ctx, clipboard.FmtText)
	watchImageChan := clipboard.Watch(ctx, clipboard.FmtImage)

	c := make(chan *Info)

	go func() {
		var lastDigest string

		for {
			var kind Kind
			var data []byte

			select {
			case <-time.After(200 * time.Millisecond):
				kind, data = ReadRaw()
			case data = <-watchTextChan:
				kind = KindText
			case data = <-watchImageChan:
				kind = KindImage
			case <-ctx.Done():
				close(c)
				return
			}

			if kind == KindNone {
				continue
			}

			digest := calcDigest(kind, data)

			if digest == lastDigest {
				continue
			}
			lastDigest = digest

			c <- &Info{
				Kind:   kind,
				Data:   data,
				Digest: digest,
			}
		}
	}()

	return c
}

func WatchWithWrite(ctx context.Context) (in chan *Info, out chan *Info) {
	in = make(chan *Info)
	out = make(chan *Info)
	watchChan := Watch(ctx)
	go func() {
		var lastDigest string
		var ok bool
		var info *Info
		for {
			select {
			case info, ok = <-watchChan:
				if !ok {
					return
				}
				digest := info.Digest
				if digest == lastDigest {
					continue
				}
				lastDigest = digest
				out <- info
			case info = <-in:
				lastDigest = info.Digest
				Write(info)
			}
		}
	}()
	return in, out
}

func calcDigest(kind Kind, data []byte) string {
	h := sha256.New()
	h.Write([]byte(kind))
	h.Write(data)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func Write(info *Info) {
	if info.Kind.ClipboardFormat() == -1 {
		return
	}
	clipboard.Write(info.Kind.ClipboardFormat(), info.Data)
}

func ReadRaw() (Kind, []byte) {
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

func Read() *Info {
	kind, data := ReadRaw()
	if kind == KindNone {
		return nil
	}
	return &Info{
		Data:   data,
		Digest: calcDigest(kind, data),
		Kind:   kind,
	}
}
