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
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)

	if err != nil {
		log.Fatal(err)
	}

	request := string(buffer[:n])

	splits := strings.Split(request, "\r\n")
	fmt.Println(splits)
	if len(splits) < 1 {
		log.Println("Invalid request: no headers")
		return
	}

	// request line
	startLineSplits := strings.Split(splits[0], " ")
	fmt.Println(startLineSplits)
	if len(startLineSplits) < 3 {
		log.Println("Invalid request: malformed start line")
		return
	}

	// Extract method, path, and HTTP version
	method := startLineSplits[0]
	path := startLineSplits[1]
	pathSplits := strings.Split(path, "/")
	root := "/" + pathSplits[1]
	var subpath string
	if len(pathSplits) > 1 {
		subpath = strings.Join(pathSplits[2:], "/")
	}
	httpVersion := startLineSplits[2]
	fmt.Println(method, httpVersion)
	// Extract User-Agent header
	var userAgent string
	for _, header := range splits {
		if strings.HasPrefix(header, "User-Agent: ") {
			userAgent = strings.TrimPrefix(header, "User-Agent: ")
			break
		}
	}

	switch root {
	case "/":
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	case "/echo":
		reply := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(subpath), subpath)
		conn.Write([]byte(reply))
	case "/user-agent":
		reply := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)
		conn.Write([]byte(reply))
	default:
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}
