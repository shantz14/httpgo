package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

type Log struct {
	Method string
	Resource string
	Status StatusCode
	Latency string
	BytesSent int
	Timestamp string
	Err error
}

func initLogger(ctx context.Context) chan Log {
	// TODO test this buffered channel under load.  All handleClient goroutines
	// will be hitting this it will break
	// or i guess they will block on channel send
	logCh := make(chan Log, 10)

	go runLogger(logCh, ctx)

	return logCh
}

func runLogger(ch chan Log, ctx context.Context) {
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.OpenFile("logs/log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case l := <-ch:
			s := l.toString()
			if _, err := f.WriteString(s); err != nil {
				fmt.Println("Failed to write to log file: ", err)
			}
		}
	}
}

func sendLog(r Response, bytesSent int, err error) {
	log := Log {
		Method: r.Request.Method,
		Resource: r.Request.Resource,
		Status: r.status,
		Latency: time.Since(r.Request.StartTime).String(),
		BytesSent: bytesSent,
		Timestamp: getTimestamp(),
		Err: err,
	}
	r.logCh <- log
}

func getTimestamp() string {
	now := time.Now()
    return now.Format("2006-01-02 15:04:05")
}

func (l Log) toString() string {
	var s string
	if l.Err == nil {
		s = fmt.Sprintf("%s %s %s %s %s %dB\n", l.Timestamp, l.Method, l.Resource, l.Status, l.Latency, l.BytesSent)
	} else {
		s = fmt.Sprintf("Error processing [%s %s %s %s %s %dB] err: %s\n", l.Timestamp, l.Method, l.Resource, l.Status, l.Latency, l.BytesSent, l.Err.Error())
	}

	return s
}
