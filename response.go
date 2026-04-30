package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"slices"
)

type Response struct {
	conn io.WriteCloser
	status StatusCode
	Request *Request
	logCh chan Log
	Header map[string][]string
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
	var body []byte

	// TODO shouldn't compress all content-types only stuff like plaintext, html
	if slices.Contains(r.Request.Header["Accept-Encoding"], "gzip") {
		compressed, err := compressThisJawn(data)
		if err != nil {
			sendLog(*r, len(data), err)
			return
		}
		sendLog(*r, len(compressed), nil)
		body = compressed
		
		r.Header["Content-Encoding"] = []string{"gzip"}
	} else {
		sendLog(*r, len(data), nil)
		body = data
	}

	// Default headers
	fmt.Fprintf(r.conn, "HTTP/1.1 %s %s\r\n", r.status, statusText[r.status])
	fmt.Fprintf(r.conn, "Content-Type: %s\r\n", contentType)
	fmt.Fprintf(r.conn, "Content-Length: %d\r\n", len(body))
	fmt.Fprintf(r.conn, "Connection: %s\r\n", r.Request.Header["Connection"][0])

	// Extra headers
	r.sendHeaders()

	fmt.Fprintf(r.conn, "\r\n")
	
	r.conn.Write(body)
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

func compressThisJawn(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)

	if _, err := zw.Write(data); err != nil {
		return []byte{}, err
	}
	if err := zw.Close(); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

func (r Response) sendHeaders() {
	for k, v := range r.Header {
		fmt.Fprintf(r.conn, "%s: ", k)

		for i, item := range v {
			if i > 0 {
				fmt.Fprintf(r.conn, ", ")
			}
			fmt.Fprintf(r.conn, "%s", item)
		}

		fmt.Fprintf(r.conn, "\r\n")
	}
}
