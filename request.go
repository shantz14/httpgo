package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Request struct {
	Method string
	Resource string
	Protocol string
	Version string
	Header map[string][]string
	Body io.ReadCloser
	StartTime time.Time
}

type LimitReadCloser struct {
	io.Reader
	io.Closer
}

func (c LimitReadCloser) Close() error {
	return c.Closer.Close()
}


var InvalidRequestLine = errors.New("invalid request line")


func parseRequest(conn net.Conn, reader *bufio.Reader, status *ConnStatus, mx *sync.Mutex, start *time.Time) (*Request, error) {
	var req Request
	// Parse the request
	// Parse request line
	err := getReqLine(reader, &req)
	if err != nil {
		return &req, err
	}

	*start = time.Now()

	mx.Lock()
	*status = ConnProcessing
	mx.Unlock()

	req.Header = make(map[string][]string)

	// Parse headers
	getHeaders(reader, &req.Header)
	if _, ok := req.Header["Connection"]; !ok {
		req.Header["Connection"] = []string{"keep-alive"} // keep-alive is the default behavior
	}

	// Parse body
	body, err := getBody(req.Header, conn, reader)
	if err != nil {
		fmt.Println("Error getting body")
		return &req, err
	}
	req.Body = body

	return &req, nil
}

func getReqLine(r *bufio.Reader, req *Request) (error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}

	// Parse the request line
	resArr := strings.Split(line, " ")
	if len(resArr) != 3 {
		return InvalidRequestLine
	}

	req.Method = strings.TrimSpace(resArr[0])
	req.Resource = strings.TrimSpace(resArr[1])

	/* TODO IF a client is silly/malicious and sends a HTTP/0.9 request
		this will panic because it looks like this

		GET /index.html

	*/
	var protocol = strings.TrimSpace(resArr[2])
	req.Protocol = strings.Split(protocol, "/")[0]
	req.Version = strings.Split(protocol, "/")[1]

	return nil
}

// getBody combines the bufio.Reader Read and the net.Conn Close into an
// io.LimitReader with a limit equal to the Content-Length and returns
// it as the request body.
func getBody(header map[string][]string, conn io.Closer, reader io.Reader) (io.ReadCloser, error) {
	var body io.ReadCloser

	// Check if there is a body... via Content-Length
	if clStr, ok := header["Content-Length"]; ok {
		cl, err := strconv.Atoi(clStr[0])
		if err != nil {
			fmt.Println("Failed to convert Content-Length to an integer")
			return nil, err
		}
		body = LimitReadCloser {
			Reader: io.LimitReader(reader, int64(cl)),
			Closer: conn,
		}
	} else {
		body = nil
	}

	return body, nil
}

func getHeaders(r *bufio.Reader, headers *map[string][]string) error {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return err
		}

		// If its not a header line
		if !strings.Contains(line, ":") {
			return nil
		}

		// Parse header line
		// Its literally impossible for this not to be "ok" right?
		field, value, _ := strings.Cut(line, ":")
		field = strings.TrimSpace(field)
		values := strings.Split(value, ",")
		for i, s := range values {
			values[i] = strings.TrimSpace(s)
		}
		
		(*headers)[field] = values
	}
}

