package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"strings"
)

var directory *string

type Request struct {
	Method  string
	URI     string
	Version string
	Headers map[string]string
	Body    string
}

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

	httpReq := Request{
		Method:  startLineSplits[0],
		URI:     startLineSplits[1],
		Version: startLineSplits[2],
		Headers: make(map[string]string),
	}
	// headers starts from splits[1:]
	for _, header := range splits[1:] {
		idx := strings.Index(header, ":") // get first index of ":"
		if idx != -1 {
			httpReq.Headers[header[:idx]] = header[idx+2:] // +2 for removing space
		}
	}

	// Extract method, path, and HTTP version
	method := httpReq.Method
	path := httpReq.URI
	pathSplits := strings.Split(path, "/")
	root := "/" + pathSplits[1]
	var subpath string
	if len(pathSplits) > 1 {
		subpath = strings.Join(pathSplits[2:], "/")
	}

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
	if (bodyStart == -1 || bodyStart+4 == len(request)) && method != "GET" {
		log.Println("No body found")
	}
	httpReq.Body = string(request[bodyStart+4:])

	switch method {
	case "GET":
		switch root {
		case "/":
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		case "/echo":
			reply := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(subpath), subpath)

			// Only accepts gzip abstract later
			if isAcceptedPresent(httpReq.Headers["Accept-Encoding"]) {
				reply = fmt.Sprintf("%s 200 OK\r\nContent-Encoding: %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
					httpReq.Version, "gzip", len(subpath), subpath)
				fmt.Println(subpath)
				gzippedBody := compressBody(subpath)
				fmt.Println(gzippedBody)

			}
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
			writeFile(filepath, httpReq.Body)
			conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
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

func isAcceptedPresent(encodings string) bool {
	e := strings.Split(encodings, ", ")
	for _, v := range e {
		if v == "gzip" {
			return true
		}
	}
	return false
}

func compressBody(body string) bytes.Buffer {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte(body))
	w.Close()
	return b
}
