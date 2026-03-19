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

type Request struct {
	Method string
	Resource string
	Protocol string
	Version string
	Header map[string][]string
	Body io.ReadCloser
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

type LimitReadCloser struct {
	io.Reader
	io.Closer
}

func (b LimitReadCloser) Close() error {
	return b.Closer.Close()
}


var InvalidRequestLine = errors.New("invalid request line")


func parseRequest(conn net.Conn) (*Request, error) {
	var req Request
	// Parse the request
	// Parse request line
	reqLine, err := getReqLine(conn)
	if err != nil {
		fmt.Println("Error parsing request line")
		return &req, err
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

	return &req, nil
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

