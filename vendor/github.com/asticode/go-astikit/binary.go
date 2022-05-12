package astikit

import (
	"encoding/binary"
	"errors"
	"io"
)

// BitsWriter represents an object that can write individual bits into a writer
// in a developer-friendly way. Check out the Write method for more information.
// This is particularly helpful when you want to build a slice of bytes based
// on individual bits for testing purposes.
type BitsWriter struct {
	bo       binary.ByteOrder
	cache    byte
	cacheLen byte
	bsCache  []byte
	w        io.Writer
	writeCb  BitsWriterWriteCallback
}

type BitsWriterWriteCallback func([]byte)

// BitsWriterOptions represents BitsWriter options
type BitsWriterOptions struct {
	ByteOrder binary.ByteOrder
	// WriteCallback is called every time when full byte is written
	WriteCallback BitsWriterWriteCallback
	Writer        io.Writer
}

// NewBitsWriter creates a new BitsWriter
func NewBitsWriter(o BitsWriterOptions) (w *BitsWriter) {
	w = &BitsWriter{
		bo:      o.ByteOrder,
		bsCache: make([]byte, 1),
		w:       o.Writer,
		writeCb: o.WriteCallback,
	}
	if w.bo == nil {
		w.bo = binary.BigEndian
	}
	return
}

func (w *BitsWriter) SetWriteCallback(cb BitsWriterWriteCallback) {
	w.writeCb = cb
}

// Write writes bits into the writer. Bits are only written when there are
// enough to create a byte. When using a string or a bool, bits are added
// from left to right as if
// Available types are:
//   - string("10010"): processed as n bits, n being the length of the input
//   - []byte: processed as n bytes, n being the length of the input
//   - bool: processed as one bit
//   - uint8/uint16/uint32/uint64: processed as n bits, if type is uintn
func (w *BitsWriter) Write(i interface{}) error {
	// Transform input into "10010" format

	switch a := i.(type) {
	case string:
		for _, r := range a {
			var err error
			if r == '1' {
				err = w.writeBit(1)
			} else {
				err = w.writeBit(0)
			}
			if err != nil {
				return err
			}
		}
	case []byte:
		for _, b := range a {
			if err := w.writeFullByte(b); err != nil {
				return err
			}
		}
	case bool:
		if a {
			return w.writeBit(1)
		} else {
			return w.writeBit(0)
		}
	case uint8:
		return w.writeFullByte(a)
	case uint16:
		return w.writeFullInt(uint64(a), 2)
	case uint32:
		return w.writeFullInt(uint64(a), 4)
	case uint64:
		return w.writeFullInt(a, 8)
	default:
		return errors.New("astikit: invalid type")
	}

	return nil
}

// Writes exactly n bytes from bs
// Writes first n bytes of bs if len(bs) > n
// Pads with padByte at the end if len(bs) < n
func (w *BitsWriter) WriteBytesN(bs []byte, n int, padByte uint8) error {
	if len(bs) >= n {
		return w.Write(bs[:n])
	}

	if err := w.Write(bs); err != nil {
		return err
	}

	// no bytes.Repeat here to avoid allocation
	for i := 0; i < n-len(bs); i++ {
		if err := w.Write(padByte); err != nil {
			return err
		}
	}

	return nil
}

func (w *BitsWriter) writeFullInt(in uint64, len int) error {
	if w.bo == binary.BigEndian {
		for i := len - 1; i >= 0; i-- {
			err := w.writeFullByte(byte((in >> (i * 8)) & 0xff))
			if err != nil {
				return err
			}
		}
	} else {
		for i := 0; i < len; i++ {
			err := w.writeFullByte(byte((in >> (i * 8)) & 0xff))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *BitsWriter) flushBsCache() error {
	if _, err := w.w.Write(w.bsCache); err != nil {
		return err
	}

	if w.writeCb != nil {
		w.writeCb(w.bsCache)
	}

	return nil
}

func (w *BitsWriter) writeFullByte(b byte) error {
	if w.cacheLen == 0 {
		w.bsCache[0] = b
	} else {
		w.bsCache[0] = w.cache | (b >> w.cacheLen)
		w.cache = b << (8 - w.cacheLen)
	}
	return w.flushBsCache()
}

func (w *BitsWriter) writeBit(bit byte) error {
	w.cache = w.cache | (bit)<<(7-w.cacheLen)
	w.cacheLen++
	if w.cacheLen == 8 {
		w.bsCache[0] = w.cache
		if err := w.flushBsCache(); err != nil {
			return err
		}

		w.cacheLen = 0
		w.cache = 0
	}
	return nil
}

// WriteN writes the input into n bits
func (w *BitsWriter) WriteN(i interface{}, n int) error {
	var toWrite uint64
	switch a := i.(type) {
	case uint8:
		toWrite = uint64(a)
	case uint16:
		toWrite = uint64(a)
	case uint32:
		toWrite = uint64(a)
	case uint64:
		toWrite = a
	default:
		return errors.New("astikit: invalid type")
	}

	for i := n - 1; i >= 0; i-- {
		err := w.writeBit(byte(toWrite>>i) & 0x1)
		if err != nil {
			return err
		}
	}
	return nil
}

// BitsWriterBatch allows to chain multiple Write* calls and check for error only once
// For more info see https://github.com/asticode/go-astikit/pull/6
type BitsWriterBatch struct {
	err error
	w   *BitsWriter
}

func NewBitsWriterBatch(w *BitsWriter) BitsWriterBatch {
	return BitsWriterBatch{
		w: w,
	}
}

// Calls BitsWriter.Write if there was no write error before
func (b *BitsWriterBatch) Write(i interface{}) {
	if b.err == nil {
		b.err = b.w.Write(i)
	}
}

// Calls BitsWriter.WriteN if there was no write error before
func (b *BitsWriterBatch) WriteN(i interface{}, n int) {
	if b.err == nil {
		b.err = b.w.WriteN(i, n)
	}
}

// Calls BitsWriter.WriteBytesN if there was no write error before
func (b *BitsWriterBatch) WriteBytesN(bs []byte, n int, padByte uint8) {
	if b.err == nil {
		b.err = b.w.WriteBytesN(bs, n, padByte)
	}
}

// Returns first write error
func (b *BitsWriterBatch) Err() error {
	return b.err
}

var byteHamming84Tab = [256]uint8{
	0x01, 0xff, 0xff, 0x08, 0xff, 0x0c, 0x04, 0xff, 0xff, 0x08, 0x08, 0x08, 0x06, 0xff, 0xff, 0x08,
	0xff, 0x0a, 0x02, 0xff, 0x06, 0xff, 0xff, 0x0f, 0x06, 0xff, 0xff, 0x08, 0x06, 0x06, 0x06, 0xff,
	0xff, 0x0a, 0x04, 0xff, 0x04, 0xff, 0x04, 0x04, 0x00, 0xff, 0xff, 0x08, 0xff, 0x0d, 0x04, 0xff,
	0x0a, 0x0a, 0xff, 0x0a, 0xff, 0x0a, 0x04, 0xff, 0xff, 0x0a, 0x03, 0xff, 0x06, 0xff, 0xff, 0x0e,
	0x01, 0x01, 0x01, 0xff, 0x01, 0xff, 0xff, 0x0f, 0x01, 0xff, 0xff, 0x08, 0xff, 0x0d, 0x05, 0xff,
	0x01, 0xff, 0xff, 0x0f, 0xff, 0x0f, 0x0f, 0x0f, 0xff, 0x0b, 0x03, 0xff, 0x06, 0xff, 0xff, 0x0f,
	0x01, 0xff, 0xff, 0x09, 0xff, 0x0d, 0x04, 0xff, 0xff, 0x0d, 0x03, 0xff, 0x0d, 0x0d, 0xff, 0x0d,
	0xff, 0x0a, 0x03, 0xff, 0x07, 0xff, 0xff, 0x0f, 0x03, 0xff, 0x03, 0x03, 0xff, 0x0d, 0x03, 0xff,
	0xff, 0x0c, 0x02, 0xff, 0x0c, 0x0c, 0xff, 0x0c, 0x00, 0xff, 0xff, 0x08, 0xff, 0x0c, 0x05, 0xff,
	0x02, 0xff, 0x02, 0x02, 0xff, 0x0c, 0x02, 0xff, 0xff, 0x0b, 0x02, 0xff, 0x06, 0xff, 0xff, 0x0e,
	0x00, 0xff, 0xff, 0x09, 0xff, 0x0c, 0x04, 0xff, 0x00, 0x00, 0x00, 0xff, 0x00, 0xff, 0xff, 0x0e,
	0xff, 0x0a, 0x02, 0xff, 0x07, 0xff, 0xff, 0x0e, 0x00, 0xff, 0xff, 0x0e, 0xff, 0x0e, 0x0e, 0x0e,
	0x01, 0xff, 0xff, 0x09, 0xff, 0x0c, 0x05, 0xff, 0xff, 0x0b, 0x05, 0xff, 0x05, 0xff, 0x05, 0x05,
	0xff, 0x0b, 0x02, 0xff, 0x07, 0xff, 0xff, 0x0f, 0x0b, 0x0b, 0xff, 0x0b, 0xff, 0x0b, 0x05, 0xff,
	0xff, 0x09, 0x09, 0x09, 0x07, 0xff, 0xff, 0x09, 0x00, 0xff, 0xff, 0x09, 0xff, 0x0d, 0x05, 0xff,
	0x07, 0xff, 0xff, 0x09, 0x07, 0x07, 0x07, 0xff, 0xff, 0x0b, 0x03, 0xff, 0x07, 0xff, 0xff, 0x0e,
}

// ByteHamming84Decode hamming 8/4 decodes
func ByteHamming84Decode(i uint8) (o uint8, ok bool) {
	o = byteHamming84Tab[i]
	if o == 0xff {
		return
	}
	ok = true
	return
}

var byteParityTab = [256]uint8{
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
}

// ByteParity returns the byte parity
func ByteParity(i uint8) (o uint8, ok bool) {
	ok = byteParityTab[i] == 1
	o = i & 0x7f
	return
}