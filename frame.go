package modbus

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
)

const (
	TcpPid uint16 = 0x0000

	// Function codes
	ReadCoils              uint8 = 0x01
	ReadDiscreteInputs     uint8 = 0x02
	ReadHoldingRegisters   uint8 = 0x03
	ReadInputRegisters     uint8 = 0x04
	WriteSingleCoil        uint8 = 0x05
	WriteSingleRegister    uint8 = 0x06
	ReadExceptionStatus    uint8 = 0x07
	WriteMultipleCoils     uint8 = 0x0F
	WriteMultipleRegisters uint8 = 0x10
	ReportSlaveId          uint8 = 0x11
	WriteAndReadRegisters  uint8 = 0x17

	// Exception Codes
	IllegalFunction        uint8 = 0x01
	IllegalDataAddress     uint8 = 0x02
	IllegalDataValue       uint8 = 0x03
	SlaveFailure           uint8 = 0x04
	Acknowledge            uint8 = 0x05
	SlaveBusy              uint8 = 0x06
	NegativeAcknowledge    uint8 = 0x07
	MemoryParityError      uint8 = 0x08
	NotDefined             uint8 = 0x09
	GatewayPathUnavailable uint8 = 0x0A
	GatewayTargetFailed    uint8 = 0x0B
)

// A Frame represents an Modbus request received by a server / slave
// or to be sent by a client / master.
type Frame struct {

	// Modbus TCP Header
	header Header

	// Data bytes - Data as reponse or commands
	data []byte
}

type Header struct {
	// Transaction identifier - For synchronization between messages of server & client
	Tid uint16
	// Protocol identifier - Zero for Modbus/TCP
	Pid uint16
	// Length field - Number of remaining bytes in this frame
	Length uint16
	// Unit identifier - Slave address (255 if not used)
	Uid byte
	// Function code - Indicates the function codes like read coils / inputs
	Fcode byte
}

// A wrapper for Modbus Frame representing a Register Request
type Request struct {
	*Frame
}

// return the number of bytes in Frame f.
func (f *Frame) Size() int {
	return 6 + int(f.header.Length)
}

func NewRequest(f *Frame) (r *Request, err error) {
	if len(f.data) < 4 {
		return nil, errors.New("modbus: Frame too small to be valid Modbus Request.")
	}
	return &Request{f}, nil
}

func (r *Request) Offset() uint16 {
	return binary.BigEndian.Uint16(r.data[0:2])
}

func (r *Request) Number() uint16 {
	if r.header.Fcode == WriteSingleCoil || r.header.Fcode == WriteSingleRegister {
		return 1
	}
	return binary.BigEndian.Uint16(r.data[2:4])
}

// ReadRequest reads and parses an incoming request from b.
func ReadFrame(b *bufio.Reader) (req *Frame, err error) {
	req = new(Frame)

	// read the header
	err = binary.Read(b, binary.BigEndian, &req.header)
	if err != nil {
		return
	}

	// now read the data
	req.data = make([]byte, req.header.Length-2)

	lr := io.LimitReader(b, int64(req.header.Length-2)).(*io.LimitedReader)
	_, err = lr.Read(req.data)

	if err != nil {
		return
	} else if lr.N != 0 {
		err = errors.New("modbus: request too small")
		return
	}

	return req, nil
}

// The calling code should prepare the buffer size accordingly
// // NewWriterSize(w io.Writer, size int) *Writer

// WriteFrame writes an outgoing Modbus Frame to b
func WriteFrame(f *Frame, b *bufio.Writer) (err error) {
	// check that the buffer is big enough
	if b.Available() < f.Size() {
		err = errors.New("modbus: write buffer too small")
		return
	}

	// write Frame
	err = binary.Write(b, binary.BigEndian, f.header)
	if err != nil {
		return
	}
	err = binary.Write(b, binary.BigEndian, f.data)
	if err != nil {
		return
	}

	return nil
}
