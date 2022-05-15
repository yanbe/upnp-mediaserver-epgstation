package bufferpool

import (
	"bufio"
	"bytes"
	"io"
	"sync"
)

var (
	DefaultMaxMessageBytes = 2 << 10
	bufBytesPool           sync.Pool
	bytesBufferPool        sync.Pool
	bufioReaderPool        sync.Pool
	bufioWriter2kPool      sync.Pool
	bufioWriter4kPool      sync.Pool
)

func NewBytesBuf() []byte {
	if v := bufBytesPool.Get(); v != nil {
		buf := v.(*[]byte)
		return *buf
	}
	return make([]byte, DefaultMaxMessageBytes)
}

func NewBytesBufSize(size int) []byte {
	if v := bufBytesPool.Get(); v != nil {
		buf := v.(*[]byte)
		return *buf
	}
	return make([]byte, DefaultMaxMessageBytes)
}

func PutBytesBuf(buf []byte) {
	bufBytesPool.Put(&buf)
}

func NewBytesBuffer() *bytes.Buffer {
	if v := bytesBufferPool.Get(); v != nil {
		bb := v.(*bytes.Buffer)
		bb.Reset()
		return bb
	}
	// Note: if this reader size is ever changed, update
	// TestHandlerBodyClose's assumptions.
	return &bytes.Buffer{}
}

func PutBytesBuffer(bb *bytes.Buffer) {
	bytesBufferPool.Put(bb)
}

func NewBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	// Note: if this reader size is ever changed, update
	// TestHandlerBodyClose's assumptions.
	return bufio.NewReader(r)
}

func PutBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderPool.Put(br)
}

func bufioWriterPool(size int) *sync.Pool {
	switch size {
	case 2 << 10:
		return &bufioWriter2kPool
	case 4 << 10:
		return &bufioWriter4kPool
	}
	return nil
}

func NewBufioWriterSize(w io.Writer, size int) *bufio.Writer {
	pool := bufioWriterPool(size)
	if pool != nil {
		if v := pool.Get(); v != nil {
			bw := v.(*bufio.Writer)
			bw.Reset(w)
			return bw
		}
	}
	return bufio.NewWriterSize(w, size)
}

func PutBufioWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	if pool := bufioWriterPool(bw.Available()); pool != nil {
		pool.Put(bw)
	}
}
