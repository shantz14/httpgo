package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type Server struct {
	Addr string
	Handler Handler
	Routes map[string]Handler
}

type Request struct {
	Method string
	Resource string
	Protocol string
	Version string
	Header map[string][]string
	Body io.ReadCloser
}

type Response struct {

}

type LimitReadCloser struct {
	io.Reader
	io.Closer
}

func (b LimitReadCloser) Close() error {
	return b.Closer.Close()
}

type ReqLine struct {
	Method string
	Resource string
	Protocol string
	Version string
}

type Field struct {
	Field string
	Value []string
}

var InvalidRequestLine = errors.New("invalid request line")

type Handler func(res Response, req *Request)

func newServer(addr string, handler Handler) *Server {
	if handler == nil {
		s := &Server{Addr: addr, Handler: nil, Routes: make(map[string]Handler)}
		s.Handler = s.DefaultMux()
		return s
	}
	return &Server{Addr: addr, Handler: handler, Routes:make(map[string]Handler)}
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	fmt.Println("Listening on port:", strings.Split(s.Addr, ":")[1])

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:\n", err)
			continue
		}

		fmt.Println("Request accepted from ", conn.RemoteAddr().String())

		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()

	var req Request
	// Parse the request
	// Parse request line
	reqLine, err := getReqLine(conn)
	if err != nil {
		fmt.Println("Error parsing request line")
		return
	}
	req.Method = reqLine.Method
	req.Resource = reqLine.Resource
	req.Protocol = reqLine.Protocol
	req.Version = reqLine.Version

	req.Header = make(map[string][]string)
	// Parse headers
	reader := getHeaderReader(conn)
	for field := range reader {
		req.Header[field.Field] = field.Value
	}

	// Parse body
	body, err := getBody(req.Header, conn)
	req.Body = body

	// Set up the Response
	var res Response

	s.Handler(res, &req)
}

func getHeaderReader(c io.ReadCloser) <-chan Field {
	out := make(chan Field)

	go func() {
		defer close(out)

		// Parse headers
		line := ""
		for {
			data := make([]byte, 8)
			n, err := c.Read(data)
			if err != nil {
				break
			}
			data = data[:n]

			if i := bytes.IndexByte(data, '\n'); i != -1 {
				line += string(data[:i])

				// Check if next line is not a header
				if !strings.Contains(line, ":") {
					break
				}

				var f Field
				f.Field = strings.TrimSpace(strings.Split(line, ":")[0])

				values := strings.Split(strings.Split(line, ":")[1], ",")

				for i, s := range values {
					values[i] = strings.TrimSpace(s)
				} 

				f.Value = values

				out <- f

				line = string(data[i:])
			} else {
				line += string(data)
			}
		}
	}()

	return out
}

func getBody(header map[string][]string, c io.ReadCloser) (io.ReadCloser, error) {
	var body io.ReadCloser

	// Check if there is a body... via Content-Length
	if clStr, ok := header["Content-Length"]; ok {
		cl, err := strconv.Atoi(clStr[0])
		if err != nil {
			fmt.Println("Failed to convert Content-Length to an integer")
			return nil, err
		}
		body = LimitReadCloser {
			Reader: io.LimitReader(c, int64(cl)),
			Closer: c,
		}
	} else {
		body = nil
	}

	return body, nil
}

func getReqLine(c net.Conn) (ReqLine, error) {
	var result ReqLine
	
	// Read the request line
	var sb strings.Builder
	for {
		b := make([]byte, 1)
		_, err := c.Read(b)
		if err != nil {
			return result, err
		}

		if rune(b[0]) != '\n' {
			sb.WriteByte(b[0])
		} else {
			break
		}

	}

	var line = sb.String()
	// Parse the request line
	resArr := strings.Split(line, " ")
	if len(resArr) != 3 {
		return result, InvalidRequestLine
	}

	result.Method = strings.TrimSpace(resArr[0])
	result.Resource = strings.TrimSpace(resArr[1])

	var protocol = strings.TrimSpace(resArr[2])
	result.Protocol = strings.Split(protocol, "/")[0]
	result.Version = strings.Split(protocol, "/")[1]

	return result, nil
}

func (s *Server) HandleFunc(route string, handler Handler) {
	s.Routes[route] = handler
}

func (s *Server) DefaultMux() Handler {
	return func(res Response, req *Request) {
		s.Routes[req.Resource](res, req)
	}
}

