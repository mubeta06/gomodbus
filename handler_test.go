package modbus

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"testing"
)

type testResponseWriter struct {
	req    *Frame // request for this response
	w      *bufio.Writer
	header Header
}

func (w *testResponseWriter) Header() *Header {
	return &w.req.header
}

func (w *testResponseWriter) Write(data []byte) (n int, err error) {
	// need to calculate new length
	w.header = *w.Header()
	w.header.Length = uint16(len(data) + 2)
	w.WriteHeader()

	if len(data) == 0 {
		return 0, nil
	}

	return w.w.Write(data)
}

func (w *testResponseWriter) WriteHeader() {
	binary.Write(w.w, binary.BigEndian, w.header)
}

func TestBoolsToBytes(t *testing.T) {
	bools := []bool{true, false, true, false, false, true, true, true,
		false, true, true}
	expected := []byte{0xE5, 0x06}

	if !bytes.Equal(BoolsToBytes(bools), expected) {
		t.Errorf("incorrect bools to bytes conversion")
	}
}

func TestBytesToBools(t *testing.T) {
	bytes := []byte{0xE5, 0x06}
	expected := []bool{true, false, true, false, false, true, true, true,
		false, true, true}

	bools := BytesToBools(bytes)
	for i, b := range expected {
		if bools[i] != b {
			t.Errorf("incorrect bytes to bools conversion")
		}
	}
}

func TestBytesToBoolsAndBack(t *testing.T) {
	b := []byte{0xAC, 0xDB, 0x35}
	expected := []bool{false, false, true, true, false, true, false, true,
		true, true, false, true, true, false, true, true,
		true, false, true, false, true, true, false, false}

	bools := BytesToBools(b)

	for i, v := range expected {
		if bools[i] != v {
			t.Errorf("incorrect bytes to bools conversion")
		}
	}

	if !bytes.Equal(BoolsToBytes(bools), b) {
		t.Errorf("incorrect bools to bytes conversion")
	}
}

func TestIllegalFunction(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x73, 0x00}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xFF, 0xF3, IllegalFunction}

	h := &RegisterHandler{}
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)

	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestReadCoils(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x01, 0x00, 0x13, 0x00, 0x25}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x08, 0xFF, 0x01, 0x05, 0xCD, 0x6B, 0xB2, 0x0E, 0x1B}

	h := &RegisterHandler{}
	h.Coils = append(make([]bool, 0x13), BytesToBools([]byte{0xCD, 0x6B, 0xB2, 0x0E, 0x1B})...)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestReadCoilsIllegalAddress(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x01, 0x00, 0xA3, 0x00, 0x25}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x81, IllegalDataAddress}

	h := &RegisterHandler{}
	h.Coils = append(make([]bool, 0x13), BytesToBools([]byte{0xCD, 0x6B, 0xB2, 0x0E, 0x1B})...)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestReadDiscreteInputs(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x02, 0x00, 0xC4, 0x00, 0x16}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x02, 0x03, 0xAC, 0xDB, 0x35}

	h := &RegisterHandler{}
	h.DiscreteInputs = append(make([]bool, 0xC4), BytesToBools([]byte{0xAC, 0xDB, 0x35})[:0x16]...)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestReadDiscreteInputsIllegalAddress(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x02, 0x00, 0xC4, 0x00, 0x17}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x82, IllegalDataAddress}

	h := &RegisterHandler{}
	h.DiscreteInputs = append(make([]bool, 0xC4), BytesToBools([]byte{0xAC, 0xDB, 0x35})[:0x16]...)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestReadInputs(t *testing.T) {
	req := []byte{0x00, 0x08, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x04, 0x00, 0x08, 0x00, 0x01}
	expected := []byte{0x00, 0x08, 0x00, 0x00, 0x00, 0x05, 0xFF, 0x04, 0x02, 0x00, 0x0A}

	h := &RegisterHandler{}
	h.Inputs = []uint16{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x000A, 0x0}
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestReadInputsIllegalAddress(t *testing.T) {
	req := []byte{0x00, 0x08, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x04, 0x00, 0x18, 0x00, 0x01}
	expected := []byte{0x00, 0x08, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x84, IllegalDataAddress}

	h := &RegisterHandler{}
	h.Inputs = []uint16{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x000A, 0x0}
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestHoldings(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x03, 0x00, 0x6B, 0x00, 0x03}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x09, 0xFF, 0x03, 0x06, 0x02, 0x2B, 0x00, 0x01, 0x00, 0x64}

	h := &RegisterHandler{}
	h.Holdings = append(make([]uint16, 0x6B), []uint16{0x022B, 0x0001, 0x0064}...)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestHoldingsIllegalAddress(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x03, 0x00, 0x6B, 0x00, 0x03}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x83, IllegalDataAddress}

	h := &RegisterHandler{}
	h.Holdings = make([]uint16, 0x1B)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestWriteSingleCoil(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x05, 0x00, 0x0A, 0xFF, 0x00}

	h := &RegisterHandler{}
	h.Coils = make([]bool, 0x0A+1)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), req) {
		t.Errorf("Incorrect Response")
	}

	if !h.Coils[0x000A] {
		t.Errorf("Coil value should be true.")
	}
}

func TestWriteSingleCoilIllegalAddress(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x05, 0x00, 0x0A, 0xFF, 0x00}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x85, IllegalDataAddress}

	h := &RegisterHandler{}
	h.Coils = make([]bool, 0x0A)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestWriteSingleCoilIllegalValue(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x05, 0x00, 0x0A, 0xFF, 0x01}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x85, IllegalDataValue}

	h := &RegisterHandler{}
	h.Coils = make([]bool, 0x0A+1)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestWriteSingleHolding(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x06, 0x00, 0x6B, 0x12, 0x34}

	h := &RegisterHandler{}
	h.Holdings = make([]uint16, 0x6B+1)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), req) {
		t.Errorf("Incorrect Response")
	}

	if h.Holdings[0x006B] != 0x1234 {
		t.Errorf("0x%04X not 0x%04X", h.Holdings[0x006B], 0x1234)
	}
}

func TestWriteSingleHoldingIllegalAddress(t *testing.T) {
	req := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x06, 0x00, 0x6B, 0x12, 0x34}
	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x86, IllegalDataAddress}

	h := &RegisterHandler{}
	h.Holdings = make([]uint16, 0x6B)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}

func TestWriteMultipleCoils(t *testing.T) {
	req := []byte{0x00, 0x0B, 0x00, 0x00, 0x00, 0x0C, 0xFF, 0x0F, 0x00, 0x13,
		0x00, 0x25, 0x05, 0xCD, 0x6B, 0xB2, 0x0E, 0x1B}
	expected := []byte{0x00, 0x0B, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x0F, 0x00,
		0x13, 0x00, 0x25}

	h := &RegisterHandler{}
	h.Coils = make([]bool, 0x13+0x25)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}

	//check the values made it ok
	v := BytesToBools([]byte{0xCD, 0x6B, 0xB2, 0x0E, 0x1B})
	for i, coil := range h.Coils[0x13 : 0x13+0x25] {
		if v[i] != coil {
			t.Errorf("Incorrect Coil value")
		}
	}
}

func TestWriteMultipleRegisters(t *testing.T) {
	req := []byte{0x00, 0x0F, 0x00, 0x00, 0x00, 0x0D, 0xFF, 0x10, 0x00, 0x6B,
		0x00, 0x03, 0x06, 0x02, 0x2B, 0x00, 0x01, 0x00, 0x64}
	expected := []byte{0x00, 0x0F, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x10, 0x00,
		0x6B, 0x00, 0x03}

	h := &RegisterHandler{}
	h.Holdings = make([]uint16, 0x6B+0x03)
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}

	//check the values made it ok
	v := []uint16{0x022B, 0x0001, 0x0064}
	for i, register := range h.Holdings[0x6B : 0x6B+0x03] {
		if v[i] != register {
			t.Errorf("Incorrect Holding value")
		}
	}

}

func TestWriteAndReadRegisters(t *testing.T) {
	req := []byte{0x00, 0x12, 0x00, 0x00, 0x00, 0x0F, 0xFF, 0x17, 0x00, 0x6B,
		0x00, 0x03, 0x00, 0x6C, 0x00, 0x02, 0x04, 0x00, 0x00, 0x00, 0x00}
	expected := []byte{0x00, 0x12, 0x00, 0x00, 0x00, 0x09, 0xFF, 0x17, 0x06,
		0x02, 0x2B, 0x00, 0x00, 0x00, 0x00}

	h := &RegisterHandler{}
	h.Holdings = make([]uint16, 0x6B+0x03)
	h.Holdings[0x6B] = 0x022B
	br := bufio.NewReader(bytes.NewReader(req))
	bw := bytes.Buffer{}
	r, _ := ReadFrame(br)
	w := &testResponseWriter{req: r, w: bufio.NewWriter(&bw)}

	h.ServeModbus(w, r)
	w.w.Flush()

	if !bytes.Equal(bw.Bytes(), expected) {
		t.Errorf("Incorrect Response")
	}
}
