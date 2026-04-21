package clipboard

import (
	"context"
	"crypto/sha256"
	"encoding/base64"

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

type Data struct {
	Kind   Kind
	Data   []byte
	Digest string
}

func (d *Data) CalcDigest() string {
	d.Digest = calcDigest(d.Kind, d.Data)
	return d.Digest
}

func Watch(ctx context.Context) chan *Data {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}

	c := make(chan *Data)

	go func(c chan *Data) {
		var lastDigest string

		watchTextChan := clipboard.Watch(ctx, clipboard.FmtText)
		watchImageChan := clipboard.Watch(ctx, clipboard.FmtImage)

		for {
			var kind Kind
			var data []byte

			select {
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

			c <- &Data{
				Kind:   kind,
				Data:   data,
				Digest: digest,
			}
		}
	}(c)

	return c
}

func WatchWithWrite(ctx context.Context) (in chan *Data, out chan *Data) {
	in = make(chan *Data)
	out = make(chan *Data)
	go func(in, out chan *Data) {
		var lastDigest string
		var ok bool
		var data *Data
		watchChan := Watch(ctx)
		for {
			select {
			case data, ok = <-watchChan:
				if !ok {
					return
				}
				digest := data.Digest
				if digest == lastDigest {
					continue
				}
				lastDigest = digest
				out <- data
			case data = <-in:
				lastDigest = data.Digest
				Write(data)
			}
		}
	}(in, out)
	return in, out
}

func calcDigest(kind Kind, data []byte) string {
	h := sha256.New()
	h.Write([]byte(kind))
	h.Write(data)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func WriteRaw(kind Kind, data []byte) {
	format := kind.ClipboardFormat()
	if format == -1 {
		return
	}
	clipboard.Write(format, data)
}

func Write(data *Data) {
	WriteRaw(data.Kind, data.Data)
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

func Read() *Data {
	kind, data := ReadRaw()
	if kind == KindNone {
		return nil
	}
	return &Data{
		Data:   data,
		Digest: calcDigest(kind, data),
		Kind:   kind,
	}
}
