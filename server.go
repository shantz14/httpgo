package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
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


	// Mutex around connStatuses
	var mx sync.Mutex
	var wg sync.WaitGroup

	connStatuses := map[net.Conn]*atomic.Int32{}

	logCh := initLogger(ctx)

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

		fmt.Println("New connection from ", conn.RemoteAddr().String())

		connStatuses[conn] = new(atomic.Int32)
		connStatuses[conn].Store(ConnNew)

		wg.Add(1)
		go s.handleClient(conn, ctx, &mx, &wg, connStatuses, logCh)
	}
}

func (s *Server) handleClient(conn net.Conn, ctx context.Context, mx *sync.Mutex, wg *sync.WaitGroup,  connStatuses map[net.Conn]*atomic.Int32, logCh chan Log) {
	defer func() {
		mx.Lock()
		delete(connStatuses, conn)
		mx.Unlock()
		conn.Close()
		wg.Done()
	}()

	reader := bufio.NewReader(conn)

	statusPtr := connStatuses[conn]
	
	for {

		select {
		case <-ctx.Done():
			return
		default:
			// TODO refactor SO THAT this blocks in some waitForConn function
			// SO THAT the logic that has to run right after unblocking but
			// has nothing to do with parsing the request isn't shoved into
			// parseRequest

			// 15 second timeout
			conn.SetDeadline(time.Now().Add(15 * time.Second))

			var res Response
			res.conn = conn
			res.logCh = logCh
			res.Header = make(map[string][]string)

			var req *Request 
			var start time.Time
			// parseRequest has ended up with a bunch of stuff because its
			// the first place that blocks when the connection is idle
			req, err := parseRequest(conn, reader, statusPtr, mx, &start)
			if err != nil {
				if ctx.Err() != nil {
					return
				}

				if errors.Is(err, os.ErrClosed) {
					res.sendError(StatusServiceUnavailable)
				} else if errors.Is(err, os.ErrDeadlineExceeded) {
					if statusPtr.Load() == ConnIdle { return }
					res.sendError(StatusRequestTimeout)
				} else {
					fmt.Println("Failed to parse request:", err)
					res.sendError(StatusBadRequest)
				}
				return
			}
			req.StartTime = start
			res.Request = req

			s.Handler(res, req)

			// Current behavior:
			// No Connection header - assume keep-alive, like HTTP/1.1
			if value, ok := req.Header["Connection"]; ok && value[0] == "close" {
				return
			}

			statusPtr.Store(ConnIdle)
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

func shutdownIdleConns(statuses map[net.Conn]*atomic.Int32) {
	for conn, status := range statuses {
		if status.Load() == ConnIdle {
			status.Store(ConnClosed)
			conn.Close()
		}
	}
}
