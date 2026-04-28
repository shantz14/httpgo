package main

import (
	"fmt"
	"io"
)

type Response struct {
	conn io.WriteCloser
	connType string //close / keep-alive
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

func (r Response) send(status StatusCode, contentType string, data []byte) {
	fmt.Fprintf(r.conn, "HTTP/1.1 %s %s\r\n", status, statusText[status])
	fmt.Fprintf(r.conn, "Content-Type: %s\r\n", contentType)
	fmt.Fprintf(r.conn, "Content-Length: %d\r\n", len(data))
	fmt.Fprintf(r.conn, "Connection: %s\r\n", r.connType)
	fmt.Fprintf(r.conn, "\r\n")

	r.conn.Write(data)
}

func (r Response) sendText(text string) {
	r.send(StatusOK, "text/plain", []byte(text))
}

func (r Response) sendError(status StatusCode) {
	// This silly
	r.send(status, "text/plain", []byte{})
}


