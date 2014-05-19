package binary

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"
)

type s0 struct {
	A string
	B string
	C int16
}

var (
	s0v = &s0{"A", "B", 1}
	s0b = []byte{0x1, 0x41, 0x1, 0x42, 0x1, 0x0}
)

func TestBinaryEncodeStruct(t *testing.T) {
	b, err := Marshal(s0v)
	assert.NoError(t, err)
	assert.Equal(t, s0b, b)
}

func TestBinaryDecodeStruct(t *testing.T) {
	s := &s0{}
	err := Unmarshal(s0b, s)
	assert.NoError(t, err)
	assert.Equal(t, s0v, s)
}

func TestBinaryDecodeToValueErrors(t *testing.T) {
	b := []byte{1, 0, 0, 0}
	var v uint32
	err := Unmarshal(b, v)
	assert.Error(t, err)
	err = Unmarshal(b, &v)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), v)
}

type s1 struct {
	Name     string
	BirthDay time.Time
	Phone    string
	Siblings int
	Spouse   bool
	Money    float64
	Tags     map[string]string
	Aliases  []string
}

var (
	s1v = &s1{
		Name:     "Bob Smith",
		BirthDay: time.Date(2013, 1, 2, 3, 4, 5, 6, time.UTC),
		Phone:    "5551234567",
		Siblings: 2,
		Spouse:   false,
		Money:    100.0,
		Tags:     map[string]string{"key": "value"},
		Aliases:  []string{"Bobby", "Robert"},
	}

	svb = []byte{0x9, 0x42, 0x6f, 0x62, 0x20, 0x53, 0x6d, 0x69, 0x74, 0x68, 0xf,
		0x1, 0x0, 0x0, 0x0, 0xe, 0xc8, 0x75, 0x9a, 0xa5, 0x0, 0x0, 0x0, 0x6, 0xff,
		0xff, 0xa, 0x35, 0x35, 0x35, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x2,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x59,
		0x40, 0x1, 0x3, 0x6b, 0x65, 0x79, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x2, 0x5,
		0x42, 0x6f, 0x62, 0x62, 0x79, 0x6, 0x52, 0x6f, 0x62, 0x65, 0x72, 0x74}
)

func TestBinaryEncodeComplex(t *testing.T) {
	b, err := Marshal(s1v)
	assert.NoError(t, err)
	assert.Equal(t, svb, b)
	s := &s1{}
	err = Unmarshal(svb, s)
	assert.NoError(t, err)
	assert.Equal(t, s1v, s)
}

type s2 struct {
	b []byte
}

func (s *s2) UnmarshalBinary(data []byte) error {
	if len(data) != 1 {
		return errors.New("expected data to be length 1")
	}
	s.b = data
	return nil
}

func (s *s2) MarshalBinary() (data []byte, err error) {
	return s.b, nil
}

func TestBinaryMarshalUnMarshaler(t *testing.T) {
	s2v := &s2{[]byte{0x13}}
	b, err := Marshal(s2v)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x1, 0x13}, b)
}

func TestMarshalUnMarshalTypeAliases(t *testing.T) {
	type Foo int64
	f := Foo(32)
	b, err := Marshal(f)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, b)
}
