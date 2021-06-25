package handlers

import (
	"fmt"
	"go.uber.org/zap"
	"io"
	"net"
	"strings"
)

func ConfigConnection(log *zap.Logger, conn net.Conn, kill chan interface{}, configsJoined string) {
	var buf []byte
	tmp := make([]byte, 4096)
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				select {
				case <-kill:
					// ignore errors from force shutdown
				default:
					log.Error("Error handling message", zap.Error(err))
				}
			}
			return
		}
		buf = append(buf, tmp[:n]...)

		for {
			i := strings.Index(string(buf), "\r\n")
			if i < 0 {
				break
			}
			command := buf[:i]
			buf = buf[i+2:]
			if string(command) == "stats" {
				_, err = conn.Write([]byte("STAT version 1.6.0\nEND\n"))
				if err != nil {
					panic(err)
				}
			} else if string(command) == "config get cluster" {
				out := fmt.Sprintf("CONFIG cluster 0 %d\n1\n%s\n\nEND\r\n", len(configsJoined), configsJoined)
				_, err = conn.Write([]byte(out))
				if err != nil {
					panic(err)
				}
			}
		}
	}
}
