package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

type Server struct {
	Addr string
	Handler Handler
	Routes map[string]Handler
}

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

	// 30 second timeout.  
	// TODO if keepalive is added, have to
	// reset the deadline when a new request comes in and not kill the
	// whole conn
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	var res Response
	res.conn = conn

	var req *Request 
	req, err := parseRequest(conn)
	if err != nil {
		fmt.Println("Failed to parse request:", err)
		res.sendError(StatusBadRequest)
		return
	}

	s.Handler(res, req)
}

func (s *Server) HandleFunc(route string, handler Handler) {
	s.Routes[route] = handler
}

func (s *Server) DefaultMux() Handler {
	return func(res Response, req *Request) {
		handler, ok := s.Routes[req.Resource]
		if !ok {
			//Send resource not found status
			res.sendError(StatusNotFound)
		} else {
			handler(res, req)
		}
	}
}

