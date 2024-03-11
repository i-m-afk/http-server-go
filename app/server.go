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

	fmt.Println(string(buffer[:n]))
	splits := strings.Split(string(buffer[:n]), "\r\n")
	startLine := splits[0]
	startLineSplits := strings.Split(startLine, " ")
	// path :
	path := startLineSplits[1]
	fmt.Println(path)

	// reply
	if path == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}

	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
}
