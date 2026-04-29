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
			timestamp := getTimestamp()

			s := fmt.Sprintf("%s %s %s %s %s %dB\n", timestamp, l.Method, l.Resource, l.Status, l.Latency, l.BytesSent)
			if _, err := f.WriteString(s); err != nil {
				fmt.Println("Failed to write to log file: ", err)
			}
		}
	}
}

func sendLog(r Response, bytesSent int) {
	log := Log {
		Method: r.Request.Method,
		Resource: r.Request.Resource,
		Status: r.status,
		Latency: time.Since(r.Request.StartTime).String(),
		BytesSent: bytesSent,
	}
	r.logCh <- log
}

// wow i thought this would be more work
func getTimestamp() string {
	now := time.Now()
    return now.Format("2006-01-02 15:04:05")
}


