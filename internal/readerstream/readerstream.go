package readerstream

import (
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
)

type ReaderStream struct {
	Reader io.Reader
}

func New(r io.Reader) ReaderStream {
	return ReaderStream{Reader: r}
}

func (r ReaderStream) ToBytes() ([]byte, error) {
	return ioutil.ReadAll(r.Reader)
}

func (r ReaderStream) ToString() (string, error) {
	b, err := r.ToBytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Stringer Interface
func (r ReaderStream) String() string {
	str, _ := r.ToString()
	return str
}

func (r ReaderStream) Transform(trans transform.Transformer) ReaderStream {
	return New(transform.NewReader(r.Reader, trans))
}

func (r ReaderStream) ToShiftJIS() ReaderStream {
	return r.Transform(japanese.ShiftJIS.NewEncoder())
}

func (r ReaderStream) FromShiftJIS() ReaderStream {
	return r.Transform(japanese.ShiftJIS.NewDecoder())
}
