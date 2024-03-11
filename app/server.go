package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	conn, err := l.Accept()
	buffer := make([]byte, 100)
	n, err := conn.Read(buffer)

	if err != nil {
		log.Fatal(err)
	}

	splits := strings.Split(string(buffer[:n]), "\r\n")
	startLine := splits[0]
	startLineSplits := strings.Split(startLine, " ")
	// path :
	fullPath := startLineSplits[1]
	fullPathSplits := strings.Split(fullPath, "/")
	path := strings.Join(fullPathSplits[:2], "/")
	childPath := strings.Join(fullPathSplits[2:], "/")

	// user-agent
	userAgent := splits[2]
	userAgent = userAgent[len("User-Agent: "):]

	switch path {
	case "/":
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	case "/echo":
		reply := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(childPath), childPath)
		conn.Write([]byte(reply))
	case "/user-agent":
		reply := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)
		conn.Write([]byte(reply))

	default:
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}

	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
}
