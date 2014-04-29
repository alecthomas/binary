package binary

import (
	"bufio"
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

var (
	DefaultEndian = binary.LittleEndian
)

func Marshal(v interface{}) ([]byte, error) {
	b := &bytes.Buffer{}
	if err := NewEncoder(b).Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func Unmarshal(b []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(b)).Decode(v)
}

type Encoder struct {
	w   io.Writer
	buf []byte
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w, make([]byte, 8)}
}

func (e *Encoder) writeVarint(v int) error {
	l := binary.PutUvarint(e.buf, uint64(v))
	_, err := e.w.Write(e.buf[:l])
	return err
}

func (b *Encoder) Encode(v interface{}) (err error) {
	switch cv := v.(type) {
	case encoding.BinaryMarshaler:
		buf, err := cv.MarshalBinary()
		if err != nil {
			return err
		}
		if err = b.writeVarint(len(buf)); err != nil {
			return err
		}
		_, err = b.w.Write(buf)

	case []byte: // fast-path byte arrays
		if err = b.writeVarint(len(cv)); err != nil {
			return
		}
		_, err = b.w.Write(cv)

	case string:
		if err = b.writeVarint(len(cv)); err != nil {
			return
		}
		_, err = b.w.Write([]byte(cv))

	case bool:
		var out byte
		if cv {
			out = 1
		}
		err = binary.Write(b.w, DefaultEndian, out)

	case int:
		err = binary.Write(b.w, DefaultEndian, int64(cv))

	case uint:
		err = binary.Write(b.w, DefaultEndian, int64(cv))

	case int8, uint8, int16, uint16, int32, uint32, int64, uint64, float32,
		float64, complex64, complex128:
		err = binary.Write(b.w, DefaultEndian, v)

	default:
		rv := reflect.Indirect(reflect.ValueOf(v))
		t := rv.Type()
		switch t.Kind() {
		case reflect.Array, reflect.Slice:
			l := rv.Len()
			if err = b.writeVarint(l); err != nil {
				return
			}
			for i := 0; i < l; i++ {
				if err = b.Encode(rv.Index(i).Interface()); err != nil {
					return
				}
			}

		case reflect.Struct:
			l := rv.NumField()
			for i := 0; i < l; i++ {
				if v := rv.Field(i); v.CanSet() && t.Field(i).Name != "_" {
					if err = b.Encode(v.Interface()); err != nil {
						return
					}
				}
			}

		case reflect.Map:
			l := rv.Len()
			if err = b.writeVarint(l); err != nil {
				return
			}
			for _, key := range rv.MapKeys() {
				value := rv.MapIndex(key)
				if err = b.Encode(key.Interface()); err != nil {
					return err
				}
				if err = b.Encode(value.Interface()); err != nil {
					return err
				}
			}

		default:
			return errors.New("unsupported type " + t.String())
		}
	}
	return
}

type Decoder struct {
	r *bufio.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{bufio.NewReader(r)}
}

func (d *Decoder) Decode(v interface{}) (err error) {
	switch cv := v.(type) {
	case *string:
		var l uint64
		if l, err = binary.ReadUvarint(d.r); err != nil {
			return
		}
		buf := make([]byte, l)
		_, err = d.r.Read(buf)
		*cv = string(buf)

	case *bool:
		var out byte
		err = binary.Read(d.r, DefaultEndian, &out)
		*cv = out != 0

	case *int:
		var out int64
		err = binary.Read(d.r, DefaultEndian, &out)
		*cv = int(out)

	case *uint:
		var out uint64
		err = binary.Read(d.r, DefaultEndian, &out)
		*cv = uint(out)

	case *int8, *uint8, *int16, *uint16, *int32, *uint32, *int64, *uint64, *float32,
		*float64, *complex64, *complex128:
		err = binary.Read(d.r, DefaultEndian, v)

	default:
		// Check if the type implements the encoding.BinaryUnmarshaler interface, and use it if so.
		if i, ok := v.(encoding.BinaryUnmarshaler); ok {
			var l uint64
			if l, err = binary.ReadUvarint(d.r); err != nil {
				return
			}
			buf := make([]byte, l)
			_, err = d.r.Read(buf)
			return i.UnmarshalBinary(buf)
		}

		// Otherwise, use reflection.
		rv := reflect.Indirect(reflect.ValueOf(v))
		if !rv.CanAddr() {
			return errors.New("can only Decode to pointer type")
		}
		t := rv.Type()

		switch t.Kind() {
		case reflect.Array, reflect.Slice:
			var l uint64
			if l, err = binary.ReadUvarint(d.r); err != nil {
				return
			}
			if t.Kind() == reflect.Slice {
				rv.Set(reflect.MakeSlice(t, int(l), int(l)))
			} else if int(l) != t.Len() {
				return fmt.Errorf("encoded size %d != real size %d", l, t.Len())
			}
			for i := 0; i < int(l); i++ {
				if err = d.Decode(rv.Index(i).Addr().Interface()); err != nil {
					return
				}
			}

		case reflect.Struct:
			l := rv.NumField()
			for i := 0; i < l; i++ {
				if v := rv.Field(i); v.CanSet() && t.Field(i).Name != "_" {
					if err = d.Decode(v.Addr().Interface()); err != nil {
						return
					}
				}
			}

		case reflect.Map:
			var l uint64
			if l, err = binary.ReadUvarint(d.r); err != nil {
				return
			}
			kt := t.Key()
			vt := t.Elem()
			rv.Set(reflect.MakeMap(t))
			for i := 0; i < int(l); i++ {
				kv := reflect.Indirect(reflect.New(kt))
				if err = d.Decode(kv.Addr().Interface()); err != nil {
					return
				}
				vv := reflect.Indirect(reflect.New(vt))
				if err = d.Decode(vv.Addr().Interface()); err != nil {
					return
				}
				rv.SetMapIndex(kv, vv)
			}

		default:
			return errors.New("unsupported type " + t.String())
		}
	}
	return
}
