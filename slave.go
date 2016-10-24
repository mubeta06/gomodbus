package modbus

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

type Handler interface {
	ServeModbus(ResponseWriter, *Frame)
}

// A ResponseWriter interface is used by an Modbus handler to
// construct a Modbus response.
type ResponseWriter interface {
	Header() *Header

	Write([]byte) (int, error)

	WriteHeader()
}

// loggingConn is used for debugging.
type loggingConn struct {
	name string
	net.Conn
}

var (
	uniqNameMu   sync.Mutex
	uniqNameNext = make(map[string]int)
)

func newLoggingConn(baseName string, c net.Conn) net.Conn {
	uniqNameMu.Lock()
	defer uniqNameMu.Unlock()
	uniqNameNext[baseName]++
	return &loggingConn{
		name: fmt.Sprintf("%s-%d", baseName, uniqNameNext[baseName]),
		Conn: c,
	}
}

func (c *loggingConn) Write(p []byte) (n int, err error) {
	log.Printf("%s.Write(%d) = ....", c.name, len(p))
	n, err = c.Conn.Write(p)
	log.Printf("%s.Write(%d) = %d, %v", c.name, len(p), n, err)
	return
}

func (c *loggingConn) Read(p []byte) (n int, err error) {
	log.Printf("%s.Read(%d) = ....", c.name, len(p))
	n, err = c.Conn.Read(p)
	log.Printf("%s.Read(%d) = %d, %v", c.name, len(p), n, err)
	return
}

func (c *loggingConn) Close() (err error) {
	log.Printf("%s.Close() = ...", c.name)
	err = c.Conn.Close()
	log.Printf("%s.Close() = %v", c.name, err)
	return
}

// checkConnErrorWriter writes to c.rwc and records any write errors to c.werr.
// It only contains one field (and a pointer field at that), so it
// fits in an interface value without an extra allocation.
type checkConnErrorWriter struct {
	c *conn
}

func (w checkConnErrorWriter) Write(p []byte) (n int, err error) {
	n, err = w.c.w.Write(p) // c.w == c.rwc, except after a hijack, when rwc is nil.
	if err != nil && w.c.werr == nil {
		w.c.werr = err
	}
	return
}

// A conn represents the server side of an HTTP connection.
type conn struct {
	remoteAddr string            // network address of remote side
	server     *Server           // the Server on which the connection arrived
	rwc        net.Conn          // i/o connection
	w          io.Writer         // checkConnErrorWriter's copy of wrc, not zeroed on Hijack
	werr       error             // any errors writing to w
	sr         liveSwitchReader  // where the LimitReader reads from; usually the rwc
	lr         *io.LimitedReader // io.LimitReader(sr)
	buf        *bufio.ReadWriter // buffered(lr,rwc), reading from bufio->limitReader->sr->rwc

	//    mu           sync.Mutex // guards the following
	//    clientGone   bool       // if client has disconnected mid-request
	//    closeNotifyc chan bool  // made lazily
	//    hijackedv    bool       // connection has been hijacked by handler
}

// A liveSwitchReader can have its Reader changed at runtime. It's
// safe for concurrent reads and switches, if its mutex is held.
type liveSwitchReader struct {
	sync.Mutex
	r io.Reader
}

func (sr *liveSwitchReader) Read(p []byte) (n int, err error) {
	sr.Lock()
	r := sr.r
	sr.Unlock()
	return r.Read(p)
}

// A response represents the server side of a Modbus response.
type response struct {
	conn        *conn
	req         *Frame // request for this response
	wroteHeader bool   // reply header has been (logically) written

	w *bufio.Writer // buffers output in chunks to conn.buf

	header       Header
	calledHeader bool // handler accessed handlerHeader via Header

	written       int64 // number of bytes written in body
	contentLength int64 // explicitly-declared Content-Length; or -1
	status        uint8 // exception status

	// close connection after this reply.  set on request and
	// updated after response from handler if there's a
	// "Connection: keep-alive" response header and a
	// Content-Length.
	closeAfterReply bool

	handlerDone bool // set true when the handler exits
}

// noLimit is an effective infinite upper bound for io.LimitedReader
const noLimit int64 = (1 << 63) - 1

// debugServerConnections controls whether all server connections are wrapped
// with a verbose logging wrapper.
const debugServerConnections = false

// Create new connection from rwc.
func (srv *Server) newConn(rwc net.Conn) (c *conn, err error) {
	c = new(conn)
	c.remoteAddr = rwc.RemoteAddr().String()
	c.server = srv
	c.rwc = rwc
	c.w = rwc
	if debugServerConnections {
		c.rwc = newLoggingConn("server", c.rwc)
	}
	c.sr.r = c.rwc
	c.lr = io.LimitReader(&c.sr, noLimit).(*io.LimitedReader)
	br := newBufioReader(c.lr)
	bw := newBufioWriterSize(checkConnErrorWriter{c}, 4<<10)
	c.buf = bufio.NewReadWriter(br, bw)
	return c, nil
}

var (
	bufioReaderPool   sync.Pool
	bufioWriter2kPool sync.Pool
	bufioWriter4kPool sync.Pool
)

func bufioWriterPool(size int) *sync.Pool {
	switch size {
	case 2 << 10:
		return &bufioWriter2kPool
	case 4 << 10:
		return &bufioWriter4kPool
	}
	return nil
}

func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	// Note: if this reader size is every changed, update
	// TestHandlerBodyClose's assumptions.
	return bufio.NewReader(r)
}

func putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderPool.Put(br)
}

func newBufioWriterSize(w io.Writer, size int) *bufio.Writer {
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

func putBufioWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	if pool := bufioWriterPool(bw.Available()); pool != nil {
		pool.Put(bw)
	}
}

var errTooLarge = errors.New("modbus: request too large")

// Read next request from connection.
func (c *conn) readRequest() (w *response, err error) {
	if d := c.server.ReadTimeout; d != 0 {
		c.rwc.SetReadDeadline(time.Now().Add(d))
	}
	if d := c.server.WriteTimeout; d != 0 {
		defer func() {
			c.rwc.SetWriteDeadline(time.Now().Add(d))
		}()
	}

	var req *Frame
	if req, err = ReadFrame(c.buf.Reader); err != nil {
		if c.lr.N == 0 {
			return nil, errTooLarge
		}
		return nil, err
	}
	c.lr.N = noLimit

	w = &response{
		conn: c,
		req:  req,
	}

	w.w = newBufioWriterSize(w.conn.buf, 2048)

	return w, nil
}

func (c *conn) setState(nc net.Conn, state ConnState) {
	if hook := c.server.ConnState; hook != nil {
		hook(nc, state)
	}
}

// Serve a new connection.
func (c *conn) serve() {
	origConn := c.rwc // copy it before it's set nil on Close or Hijack
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			c.server.logf("http: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
		c.close()
		c.setState(origConn, StateClosed)
	}()

	for {
		w, err := c.readRequest()
		if c.lr.N != 0 { //c.server.initialLimitedReaderSize() {
			// If we read any bytes off the wire, we're active.
			c.setState(c.rwc, StateActive)
		}
		if err != nil {
			if err == errTooLarge {
				break // Don't reply
			} else if err == io.EOF {
				break // Don't reply
			} else if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				break // Don't reply
			}
			//io.WriteString(c.rwc, "HTTP/1.1 400 Bad Request\r\n\r\n")
			break
		}

		c.server.Handler.ServeModbus(w, w.req)
		w.finishRequest() // write the payload
		if !w.shouldReuseConnection() {
			break
		}
		c.setState(c.rwc, StateIdle)
	}
}

func (w *response) Header() *Header {
	w.calledHeader = true
	return &w.req.header
}

func (w *response) Write(data []byte) (n int, err error) {
	if !w.wroteHeader {
		// need to calculate new length
		w.header = *w.Header()
		w.header.Length = uint16(len(data) + 2)
		w.WriteHeader()
	}
	if len(data) == 0 {
		return 0, nil
	}

	w.written += int64(len(data)) // ignoring errors, for errorKludge
	return w.w.Write(data)
}

func (w *response) WriteHeader() {
	binary.Write(w.w, binary.BigEndian, w.header)
	w.wroteHeader = true
}

func (w *response) finishRequest() {
	w.handlerDone = true
	w.w.Flush()
	putBufioWriter(w.w)
	w.conn.buf.Flush()
}

// shouldReuseConnection reports whether the underlying TCP connection can be reused.
// It must only be called after the handler is done executing.
func (w *response) shouldReuseConnection() bool {
	if w.closeAfterReply {
		// The request or something set while executing the
		// handler indicated we shouldn't reuse this
		// connection.
		return false
	}

	//if w.req.Method != "HEAD" && w.contentLength != -1 && w.bodyAllowed() && w.contentLength != w.written {
	// Did not write enough. Avoid getting out of sync.
	//    return false
	//}

	// There was some error writing to the underlying connection
	// during the request, so don't re-use this conn.
	if w.conn.werr != nil {
		return false
	}

	return true
}

func (c *conn) finalFlush() {
	if c.buf != nil {
		c.buf.Flush()

		// Steal the bufio.Reader (~4KB worth of memory) and its associated
		// reader for a future connection.
		putBufioReader(c.buf.Reader)

		// Steal the bufio.Writer (~4KB worth of memory) and its associated
		// writer for a future connection.
		putBufioWriter(c.buf.Writer)

		c.buf = nil
	}
}

// Close the connection.
func (c *conn) close() {
	c.finalFlush()
	if c.rwc != nil {
		c.rwc.Close()
		c.rwc = nil
	}
}

// A Server defines parameters for running an HTTP server.
// The zero value for Server is a valid configuration.
type Server struct { // this to become Slave
	Addr           string        // TCP address to listen on, ":http" if empty
	Handler        Handler       // handler to invoke, http.DefaultServeMux if nil
	ReadTimeout    time.Duration // maximum duration before timing out read of the request
	WriteTimeout   time.Duration // maximum duration before timing out write of the response
	MaxHeaderBytes int           // maximum size of request headers, DefaultMaxHeaderBytes if 0

	// ConnState specifies an optional callback function that is
	// called when a client connection changes state. See the
	// ConnState type and associated constants for details.
	ConnState func(net.Conn, ConnState)

	// ErrorLog specifies an optional logger for errors accepting
	// connections and unexpected behavior from handlers.
	// If nil, logging goes to os.Stderr via the log package's
	// standard logger.
	ErrorLog *log.Logger

	// keep Alive functionality not implemented for the moment - matb.
	disableKeepAlives int32 // accessed atomically.
}

// A ConnState represents the state of a client connection to a server.
// It's used by the optional Server.ConnState hook.
type ConnState int

const (
	// StateNew represents a new connection that is expected to
	// send a request immediately. Connections begin at this
	// state and then transition to either StateActive or
	// StateClosed.
	StateNew ConnState = iota

	// StateActive represents a connection that has read 1 or more
	// bytes of a request. The Server.ConnState hook for
	// StateActive fires before the request has entered a handler
	// and doesn't fire again until the request has been
	// handled. After the request is handled, the state
	// transitions to StateClosed, StateHijacked, or StateIdle.
	StateActive

	// StateIdle represents a connection that has finished
	// handling a request and is in the keep-alive state, waiting
	// for a new request. Connections transition from StateIdle
	// to either StateActive or StateClosed.
	StateIdle

	// StateHijacked represents a hijacked connection.
	// This is a terminal state. It does not transition to StateClosed.
	StateHijacked

	// StateClosed represents a closed connection.
	// This is a terminal state. Hijacked connections do not
	// transition to StateClosed.
	StateClosed
)

var stateName = map[ConnState]string{
	StateNew:      "new",
	StateActive:   "active",
	StateIdle:     "idle",
	StateHijacked: "hijacked",
	StateClosed:   "closed",
}

func (c ConnState) String() string {
	return stateName[c]
}

// ListenAndServe listens on the TCP network address srv.Addr and then
// calls Serve to handle requests on incoming connections.  If
// srv.Addr is blank, ":1502" is used.
func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":1502"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each.  The service goroutines read requests and
// then call srv.Handler to reply to them.
func (srv *Server) Serve(l net.Listener) error {
	defer l.Close()
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		rw, e := l.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.logf("http: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		c, err := srv.newConn(rw)
		if err != nil {
			continue
		}
		c.setState(c.rwc, StateNew) // before Serve can return
		go c.serve()
	}
}

func (s *Server) logf(format string, args ...interface{}) {
	if s.ErrorLog != nil {
		s.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func ListenAndServe(addr string, handler Handler) error {
	srv := &Server{Addr: addr, Handler: handler}
	return srv.ListenAndServe()
}
