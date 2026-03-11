package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type ReqLine struct {
	Method string
	Resource string
	Protocol string
	Version string
}

type Field struct {
	Field string
	Value string
}

var InvalidRequestLine = errors.New("invalid request line")

func getReader(c io.ReadCloser) <-chan Field {
	out := make(chan Field)

	// First line: GET /index.html HTTP/1.0

	go func() {
		defer close(out)

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

			} else {
				line += string(data)
			}

		}

	}()

	return out
}

func main() {
	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatal("Error starting tcp server:", err)
	}
	defer listener.Close()

	fmt.Printf("Listening on port: 8080\n\n")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:\n", err)
			continue
		}

		fmt.Println("Request accepted from ", conn.RemoteAddr().String())

		go handleClient(conn)
	}

}

func handleClient(conn net.Conn) {
	defer conn.Close()

	reqLine, err := getReqLine(conn)
	fmt.Println("Method: \n", reqLine.Method) 
	fmt.Println("Resource: \n", reqLine.Resource) 
	fmt.Println("Protocol: \n", reqLine.Protocol) 
	fmt.Println("Version: \n", reqLine.Version) 
	if err != nil {
		fmt.Println("Error parsing request line")
		return
	}

	reader := getReader(conn)

	for field := range reader {
		fmt.Println("Field: \n", field.Field)
		fmt.Println("Value: \n", field.Value)
	}

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

