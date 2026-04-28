package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	Addr string
	Handler Handler
	Routes map[string]Handler
}

type Handler func(res Response, req *Request)

type ConnStatus int

const (
	ConnNew = 1
	ConnIdle
	ConnProcessing
	ConnClosed
)

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

	// ctx to cancel when SIGINIT/SIGTERM happens
	// so we can gracefully shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	// waitgroup to wait for all client goroutines to
	// gracefully shutdown before main loop can exit
	// waitgroup is just a counting semaphore
	var wg sync.WaitGroup

	connStatuses := map[net.Conn]*ConnStatus{}

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				fmt.Println()
				fmt.Println()
				fmt.Println(context.Cause(ctx))
				fmt.Println("Shutting down idle connections...")
				shutdownIdleConns(connStatuses);
				fmt.Println("Done")

				fmt.Println("Waiting for all connections to shutdown...")
				wg.Wait()

				fmt.Println("Done")
				return ctx.Err()
			}

			fmt.Println("Error accepting connection:\n", err)
			continue
		}

		fmt.Println("Request accepted from ", conn.RemoteAddr().String())

		connStatuses[conn] = new(ConnStatus)
		*connStatuses[conn] = ConnNew

		wg.Add(1)
		go s.handleClient(conn, ctx, &wg, connStatuses[conn])
	}
}

func (s *Server) handleClient(conn net.Conn, ctx context.Context, wg *sync.WaitGroup, status *ConnStatus) {
	defer func() {
		*status = ConnClosed
	}()
	defer conn.Close()
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 15 second timeout
			conn.SetDeadline(time.Now().Add(15 * time.Second))

			var res Response
			res.conn = conn

			var req *Request 
			req, err := parseRequest(conn, status)
			if err != nil {
				if errors.Is(err, os.ErrClosed) {
					res.sendError(StatusServiceUnavailable)
				} else if errors.Is(err, os.ErrDeadlineExceeded) {
					res.sendError(StatusRequestTimeout)
				} else {
					fmt.Println("Failed to parse request:", err)
					res.sendError(StatusBadRequest)
				}
				return
			}
			res.connType = req.Header["Connection"][0]

			s.Handler(res, req)

			// Current behaivor:
			// No Connection header - assume keep-alive, like HTTP/1.1
			if value, ok := req.Header["Connection"]; ok && value[0] == "close" {
				return
			}

			*status = ConnIdle
		}
	}
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

func shutdownIdleConns(statuses map[net.Conn]*ConnStatus) {
	for conn, status := range statuses {
		if *status == ConnIdle {
			conn.Close()
		}
	}
}
