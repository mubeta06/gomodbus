package modbus

import (
	"bytes"
	"encoding/binary"
)

// A RegisterHandler implements the modbus.Handler interface, servicing
// Modbus request in accordance with http://www.modbus.org/docs/Modbus_Application_Protocol_V1_1b3.pdf
type RegisterHandler struct {
	Coils          []bool
	DiscreteInputs []bool
	Inputs         []uint16
	Holdings       []uint16
}

func (h *RegisterHandler) ServeModbus(w ResponseWriter, r *Frame) {

	// interrogate Request Frame's Function Code
	switch r.header.Fcode {
	case ReadCoils:
		h.ReadCoils(w, r)
	case ReadDiscreteInputs:
		h.ReadDiscreteInputs(w, r)
	case ReadHoldingRegisters:
		h.ReadHoldingRegisters(w, r)
	case ReadInputRegisters:
		h.ReadInputRegisters(w, r)
	case WriteSingleCoil:
		h.WriteSingleCoil(w, r)
	case WriteSingleRegister:
		h.WriteSingleRegister(w, r)
	case WriteMultipleCoils:
		h.WriteMultipleCoils(w, r)
	case WriteMultipleRegisters:
		h.WriteMultipleRegisters(w, r)
	case WriteAndReadRegisters:
		h.WriteAndReadRegisters(w, r)
	case ReadExceptionStatus: // serial only
	case ReportSlaveId: // serial only
	default:
		// Unknown Function Code
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalFunction})
	}
}

func BoolsToBytes(bools []bool) (bytes []byte) {
	value := uint8(0)

	for i, b := range bools {
		if b {
			value |= 1 << uint(i%8)
		}
		if (i%8 == 7) || i == len(bools)-1 {
			bytes = append(bytes, value)
			value = 0
		}
	}

	return
}

func BytesToBools(bytes []byte) (bools []bool) {
	for _, b := range bytes {
		for i := 0; i < 8; i++ {
			if ((b >> uint(i)) & 0x01) == 1 {
				bools = append(bools, true)
			} else {
				bools = append(bools, false)
			}
		}
	}

	return
}

func (h *RegisterHandler) ReadCoils(w ResponseWriter, r *Frame) {
	// ensure request payload is correct length
	if len(r.data) != 4 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// get offset and number of registers
	offset := binary.BigEndian.Uint16(r.data[0:2])
	num := binary.BigEndian.Uint16(r.data[2:4])

	if num < 1 || num > 0x07D0 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// check register request range
	if int(offset+num) > len(h.Coils) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataAddress})
		return
	}

	// take appropriate slice and convert to bytes
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, BoolsToBytes(h.Coils[offset:offset+num]))
	if err != nil {
		w.Header().Fcode += 0x80
		w.Write([]byte{SlaveFailure})
		return
	}

	data := buf.Bytes()

	w.Write(append([]byte{byte(len(data))}, data...))

	return
}

func (h *RegisterHandler) ReadDiscreteInputs(w ResponseWriter, r *Frame) {
	// ensure request payload is correct length
	if len(r.data) != 4 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// get offset and number of registers
	offset := binary.BigEndian.Uint16(r.data[0:2])
	num := binary.BigEndian.Uint16(r.data[2:4])

	if num < 1 || num > 0x07D0 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// check register request range
	if int(offset+num) > len(h.DiscreteInputs) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataAddress})
		return
	}

	// take appropriate slice and convert to bytes
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, BoolsToBytes(h.DiscreteInputs[offset:offset+num]))
	if err != nil {
		w.Header().Fcode += 0x80
		w.Write([]byte{SlaveFailure})
		return
	}

	data := buf.Bytes()

	w.Write(append([]byte{byte(len(data))}, data...))

	return
}

func (h *RegisterHandler) ReadInputRegisters(w ResponseWriter, r *Frame) {
	// ensure request payload is correct length
	if len(r.data) != 4 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// get offset and number of registers
	offset := binary.BigEndian.Uint16(r.data[0:2])
	num := binary.BigEndian.Uint16(r.data[2:4])

	if num < 1 || num > 0x007D {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// check register request range
	if int(offset+num) > len(h.Inputs) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataAddress})
		return
	}

	// take appropriate slice and convert to bytes
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, h.Inputs[offset:offset+num])
	if err != nil {
		w.Header().Fcode += 0x80
		w.Write([]byte{SlaveFailure})
		return
	}

	data := buf.Bytes()

	w.Write(append([]byte{byte(len(data))}, data...))

	return
}

func (h *RegisterHandler) ReadHoldingRegisters(w ResponseWriter, r *Frame) {
	// ensure request payload is correct length
	if len(r.data) != 4 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// get offset and number of registers
	offset := binary.BigEndian.Uint16(r.data[0:2])
	num := binary.BigEndian.Uint16(r.data[2:4])

	if num < 1 || num > 0x007D {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// check register request range
	if int(offset+num) > len(h.Holdings) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataAddress})
		return
	}

	// take appropriate slice and convert to bytes
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, h.Holdings[offset:offset+num])
	if err != nil {
		w.Header().Fcode += 0x80
		w.Write([]byte{SlaveFailure})
		return
	}

	data := buf.Bytes()

	w.Write(append([]byte{byte(len(data))}, data...))

	return
}

func (h *RegisterHandler) WriteSingleCoil(w ResponseWriter, r *Frame) {
	// ensure request payload is correct length
	if len(r.data) != 4 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// get register address
	address := binary.BigEndian.Uint16(r.data[0:2])

	// check register request range
	if int(address) >= len(h.Coils) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataAddress})
		return
	}

	// parse value
	value := binary.BigEndian.Uint16(r.data[2:4])

	if value == 0xFF00 {
		h.Coils[address] = true
	} else if value == 0x0 {
		h.Coils[address] = false
	} else {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	w.Write(r.data)

	return
}

func (h *RegisterHandler) WriteSingleRegister(w ResponseWriter, r *Frame) {
	// ensure request payload is correct length
	if len(r.data) != 4 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// get register address
	address := binary.BigEndian.Uint16(r.data[0:2])

	// check register request range
	if int(address) >= len(h.Holdings) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataAddress})
		return
	}

	// parse and write value
	h.Holdings[address] = binary.BigEndian.Uint16(r.data[2:4])

	w.Write(r.data)

	return
}

func (h *RegisterHandler) WriteMultipleCoils(w ResponseWriter, r *Frame) {
	// ensure request payload is at least correct length
	if len(r.data) < 6 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// get offset and number of registers
	offset := binary.BigEndian.Uint16(r.data[0:2])
	num := binary.BigEndian.Uint16(r.data[2:4])

	if num < 1 || num > 0x07B0 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// check register request range
	if int(offset+num) > len(h.Coils) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataAddress})
		return
	}

	// parse values
	nb := int(r.data[4])
	if len(r.data) != 5+nb {
		w.Header().Fcode += 0x80
		w.Write([]byte{SlaveFailure})
		return
	}

	if copy(h.Coils[offset:offset+num], BytesToBools(r.data[5:5+nb])) != int(num) {
		w.Header().Fcode += 0x80
		w.Write([]byte{SlaveFailure})
		return
	}

	w.Write(r.data[0:4])

	return
}

func (h *RegisterHandler) WriteMultipleRegisters(w ResponseWriter, r *Frame) {
	// ensure request payload is at least correct length
	if len(r.data) < 7 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// get offset and number of registers
	offset := binary.BigEndian.Uint16(r.data[0:2])
	num := binary.BigEndian.Uint16(r.data[2:4])

	if num < 1 || num > 0x007B {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// check register request range
	if int(offset+num) > len(h.Holdings) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataAddress})
		return
	}

	// parse values
	nb := int(r.data[4])
	if len(r.data) != 5+nb {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	buf := bytes.NewReader(r.data[5 : 5+nb])
	err := binary.Read(buf, binary.BigEndian, h.Holdings[offset:offset+num])
	if err != nil {
		w.Header().Fcode += 0x80
		w.Write([]byte{SlaveFailure})
		return
	}

	w.Write(r.data[0:4])

	return
}

func (h *RegisterHandler) WriteAndReadRegisters(w ResponseWriter, r *Frame) {
	// ensure request payload is at least correct length
	if len(r.data) < 11 {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// get offsets and number of registers
	roffset := binary.BigEndian.Uint16(r.data[0:2])
	rnum := binary.BigEndian.Uint16(r.data[2:4])
	woffset := binary.BigEndian.Uint16(r.data[4:6])
	wnum := binary.BigEndian.Uint16(r.data[6:8])
	nb := int(r.data[8])

	if rnum < 1 || rnum > 0x007D || wnum < 1 || wnum > 0x0079 || nb != int(wnum*2) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	// check register request ranges
	if int(roffset+rnum) > len(h.Holdings) || int(woffset+wnum) > len(h.Holdings) {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataAddress})
		return
	}

	if len(r.data) != 9+nb {
		w.Header().Fcode += 0x80
		w.Write([]byte{IllegalDataValue})
		return
	}

	err := binary.Read(bytes.NewReader(r.data[9:9+nb]), binary.BigEndian, h.Holdings[woffset:woffset+wnum])
	if err != nil {
		w.Header().Fcode += 0x80
		w.Write([]byte{SlaveFailure})
		return
	}

	// take appropriate read slice and convert to bytes
	buf := &bytes.Buffer{}
	err = binary.Write(buf, binary.BigEndian, h.Holdings[roffset:roffset+rnum])
	if err != nil {
		w.Header().Fcode += 0x80
		w.Write([]byte{SlaveFailure})
		return
	}

	data := buf.Bytes()

	w.Write(append([]byte{byte(len(data))}, data...))

	return
}
