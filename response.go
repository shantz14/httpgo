package main

import (
	"fmt"
	"io"
	"strconv"
)

type Response struct {
	conn io.WriteCloser
}

func (r Response) send(data []byte) {
	res := []byte(`
HTTP/1.1 200 OK
Date: Mon, 27 Jul 2026 12:28:53 GMT
Server: Apache/2.2.14 (Win32)
Last-Modified: Wed, 22 Jul 2026 19:15:56 GMT
Content-Length: ` + strconv.Itoa(len(data)) + `
Content-Type: text/plain
Connection: Closed

`)

	res = append(res, data...)

	_, err := r.conn.Write(res)
	if err != nil {
		fmt.Println("Error writing response:", err)
	}
}

func (r Response) sendText(text string) {
	r.send([]byte(text))
}


