package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"strings"
)

var directory *string

func main() {
	// get flags
	directory = flag.String("directory", "", "directory where files are searched")
	flag.Parse()
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
	if len(splits) < 1 {
		log.Println("Invalid request: no headers")
		return
	}

	// request line
	startLineSplits := strings.Split(splits[0], " ")
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
	fmt.Println("http version: ", httpVersion)
	// Extract User-Agent header
	var userAgent string
	for _, header := range splits {
		if strings.HasPrefix(header, "User-Agent: ") {
			userAgent = strings.TrimPrefix(header, "User-Agent: ")
			break
		}
	}
	// Read Body
	bodyStart := strings.Index(string(request), "\r\n\r\n")
	if bodyStart == -1 || bodyStart+4 == len(request) {
		log.Println("No body found")
	}

	switch method {
	case "GET":
		switch root {
		case "/":
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		case "/echo":
			reply := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(subpath), subpath)
			conn.Write([]byte(reply))
		case "/user-agent":
			reply := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)
			conn.Write([]byte(reply))
		case "/files":
			filepath := *directory + subpath
			ok, content := getFile(filepath)
			if !ok {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			} else {
				conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(content), content)))
			}
		default:
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
	case "POST":
		switch root {
		case "/files":
			filename := subpath
			filepath := *directory + "/" + filename
			writeFile(filepath, string(request[bodyStart+4:]))
			conn.Write([]byte("HTTP/1.1 201\r\n\r\n"))
		}
	}
}

func getFile(filepath string) (bool, string) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return false, ""
	}
	return true, string(file)
}

func writeFile(filepath string, content string) {
	os.WriteFile(filepath, []byte(content), fs.FileMode(os.O_CREATE))
}
