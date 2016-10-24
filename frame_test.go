package modbus

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
)

func TestReadFrame(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x05, 0xFF, 0x04, 0x02, 0x00, 0x0A}
	b := bufio.NewReader(bytes.NewReader(req))
	f, err := ReadFrame(b)

	if err != nil {
		t.Errorf("err not nil")
	}
	if f.header.Tid != 1 {
		t.Errorf("Transaction identifier should be %v not %v", 1, f.header.Tid)
	}
	if f.header.Pid != TcpPid {
		t.Errorf("Protocol identifier should be %v not %v", TcpPid, f.header.Pid)
	}
	if f.header.Uid != 0xFF {
		t.Errorf("Unit identifier should be %v not %v", 0xFF, f.header.Uid)
	}
	if f.header.Fcode != ReadInputRegisters {
		t.Errorf("Function code should be %v not %v", ReadInputRegisters, f.header.Uid)
	}
}

func TestReadFrameBadLength(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x04, 0x02, 0x00, 0x0A}
	b := bufio.NewReader(bytes.NewReader(req))
	_, err := ReadFrame(b)

	if err == nil {
		t.Errorf("err should not be nil")
	}
}

func TestReadFrameBadHeader(t *testing.T) {
	req := []byte{0x00, 0x01}
	b := bufio.NewReader(bytes.NewReader(req))
	_, err := ReadFrame(b)

	if err == nil {
		t.Errorf("err should not be nil")
	}
}

func TestRequestTooShort(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x01, 0x00}
	b := bufio.NewReader(bytes.NewReader(req))
	f, _ := ReadFrame(b)

	_, err := NewRequest(f)

	if err == nil {
		t.Errorf("Error should not be nil.")
	}
}

func TestRequestMulti(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x01, 0x00, 0x13, 0x00, 0x25}
	b := bufio.NewReader(bytes.NewReader(req))
	f, _ := ReadFrame(b)

	r, err := NewRequest(f)

	if err != nil {
		t.Errorf("Error should be nil.")
	}

	if r.Offset() != 0x0013 {
		t.Errorf("Offset should be %v not %v", 0x0013, r.Offset())
	}

	if r.Number() != 0x0025 {
		t.Errorf("Number of registers should be %v not %v", 0x0025, r.Number())
	}
}

func TestRequestSingle(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x05, 0x00, 0x0A, 0xFF, 0x00}
	b := bufio.NewReader(bytes.NewReader(req))
	f, _ := ReadFrame(b)

	r, err := NewRequest(f)

	if err != nil {
		t.Errorf("Error should be nil.")
	}

	if r.Offset() != 0x000A {
		t.Errorf("Offset should be %v not %v", 0x000A, r.Offset())
	}

	if r.Number() != 1 {
		t.Errorf("Number of registers should be %v not %v", 1, r.Number())
	}
}

func TestFrameSize(t *testing.T) {
	h := Header{Tid: 0x0000, Pid: 0x0000, Length: 0x0005, Uid: 0xFF, Fcode: 0x04}
	f := Frame{header: h, data: []byte{0x02, 0x00, 0x0A}}
	if f.Size() != 11 {
		t.Errorf("Incorrect Size of Frame %v bytes not %v bytes", f.Size(), 11)
	}
}

func TestWriteFrame(t *testing.T) {
	h := Header{Tid: 0x0000, Pid: 0x0000, Length: 0x0005, Uid: 0xFF, Fcode: 0x04}
	f := &Frame{header: h, data: []byte{0x02, 0x00, 0x0A}}

	buf := &bytes.Buffer{}
	bw := bufio.NewWriterSize(buf, 0)

	err := WriteFrame(f, bw)
	err = bw.Flush()
	if err != nil {
		t.Errorf("this is err %v", err)
		//t.Error("Expected buffer too small error.")
	}

	fmt.Printf("this is len buf %v\n", len(buf.Bytes()))
	for _, bb := range buf.Bytes() {
		t.Errorf("%02X", bb)
	}

	//b := bufio.NewWriter(w)(bytes.NewReader(req))
	//f, err := ReadFrame(b)
}
