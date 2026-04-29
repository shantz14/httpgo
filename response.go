package main

import (
	"fmt"
	"io"
)

type Response struct {
	conn io.WriteCloser
	status StatusCode
	Request *Request
	logCh chan Log
}

type StatusCode string

const (
	StatusOK StatusCode = "200"
	StatusNotFound StatusCode = "404"
	StatusBadRequest StatusCode = "400"
	StatusServiceUnavailable StatusCode = "503"
	StatusRequestTimeout StatusCode = "408"
)

var statusText = map[StatusCode]string {
	StatusOK:       "OK",
	StatusNotFound: "Not Found",
	StatusBadRequest: "Bad Request",
	StatusServiceUnavailable: "Service Unavailable",
	StatusRequestTimeout: "Request Timeout",
}

func (r *Response) send(contentType string, data []byte) {
	fmt.Fprintf(r.conn, "HTTP/1.1 %s %s\r\n", r.status, statusText[r.status])
	fmt.Fprintf(r.conn, "Content-Type: %s\r\n", contentType)
	fmt.Fprintf(r.conn, "Content-Length: %d\r\n", len(data))
	fmt.Fprintf(r.conn, "Connection: %s\r\n", r.Request.Header["Connection"][0])
	fmt.Fprintf(r.conn, "\r\n")

	sendLog(*r, len(data))

	r.conn.Write(data)
}

func (r *Response) sendText(text string) {
	r.status = StatusOK
	r.send("text/plain", []byte(text))
}

func (r *Response) sendError(status StatusCode) {
	r.status = status
	// This silly
	r.send("text/plain", []byte{})
}


