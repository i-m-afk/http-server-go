package main

/*
* This is a toy implementation of an HTTP server
* from scratch using a TCP server
 */

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
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
		log.Fatalf("Failed to bind to port 4221: %v", err)
	}
	defer l.Close()
	log.Println("Listening on port 4221...")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading from connection: %v", err)
		return
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
	// headers start from splits[1:]
	for _, header := range splits[1:] {
		idx := strings.Index(header, ":")
		if idx != -1 {
			httpReq.Headers[header[:idx]] = strings.TrimSpace(header[idx+1:])
		}
	}

	// Read Body
	bodyStart := strings.Index(request, "\r\n\r\n")
	if bodyStart != -1 && bodyStart+4 < len(request) && httpReq.Method != "GET" {
		httpReq.Body = request[bodyStart+4:]
	}

	serveHTTP(conn, httpReq)
}

func serveHTTP(conn net.Conn, req Request) {
	root, subpath := getRootAndSubpath(req.URI)
	switch req.Method {
	case "GET":
		handleGET(conn, req, root, subpath)
	case "POST":
		handlePOST(conn, req, root, subpath)
	default:
		writeResponse(conn, 405, "Method Not Allowed", "")
	}
}

func handleGET(conn net.Conn, req Request, root, subpath string) {
	switch root {
	case "/":
		writeResponse(conn, 200, "OK", "")
	case "/echo":
		handleEcho(conn, req, subpath)
	case "/user-agent":
		handleUserAgent(conn, req)
	case "/files":
		handleFileRequest(conn, subpath)
	default:
		writeResponse(conn, 404, "Not Found", "")
	}
}

func handlePOST(conn net.Conn, req Request, root, subpath string) {
	if root == "/files" {
		handleFileUpload(conn, subpath, req.Body)
	} else {
		writeResponse(conn, 404, "Not Found", "")
	}
}

func handleEcho(conn net.Conn, req Request, subpath string) {
	body := subpath
	if acceptsGzip(req.Headers["Accept-Encoding"]) {
		gzippedBody := compressBody(body)
		headers := map[string]string{
			"Content-Encoding": "gzip",
			"Content-Type":     "text/plain",
			"Content-Length":   fmt.Sprintf("%d", gzippedBody.Len()),
		}
		writeResponseWithHeaders(conn, 200, "OK", gzippedBody.String(), headers)
	} else {
		writeResponse(conn, 200, "OK", body)
	}
}

func handleUserAgent(conn net.Conn, req Request) {
	userAgent := req.Headers["User-Agent"]
	writeResponse(conn, 200, "OK", userAgent)
}

func handleFileRequest(conn net.Conn, subpath string) {
	filepath := filepath.Clean(*directory + "/" + subpath)
	if !strings.HasPrefix(filepath, *directory) {
		writeResponse(conn, 403, "Forbidden", "")
		return
	}
	ok, content := getFile(filepath)
	if !ok {
		writeResponse(conn, 404, "Not Found", "")
	} else {
		headers := map[string]string{
			"Content-Type":   "application/octet-stream",
			"Content-Length": fmt.Sprintf("%d", len(content)),
		}
		writeResponseWithHeaders(conn, 200, "OK", content, headers)
	}
}

func handleFileUpload(conn net.Conn, filename, content string) {
	filepath := filepath.Clean(*directory + "/" + filename)
	if !strings.HasPrefix(filepath, *directory) {
		writeResponse(conn, 403, "Forbidden", "")
		return
	}
	err := os.WriteFile(filepath, []byte(content), fs.FileMode(0644))
	if err != nil {
		log.Printf("Error writing file: %v", err)
		writeResponse(conn, 500, "Internal Server Error", "")
		return
	}
	writeResponse(conn, 201, "Created", "")
}

func getFile(filepath string) (bool, string) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return false, ""
	}
	return true, string(file)
}

func acceptsGzip(encodings string) bool {
	for _, encoding := range strings.Split(encodings, ",") {
		if strings.TrimSpace(encoding) == "gzip" {
			return true
		}
	}
	return false
}

func compressBody(body string) *bytes.Buffer {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write([]byte(body))
	if err != nil {
		log.Printf("Error compressing body: %v", err)
	}
	w.Close()
	return &b
}

func writeResponse(conn net.Conn, statusCode int, statusText, body string) {
	headers := map[string]string{
		"Content-Type":   "text/plain",
		"Content-Length": fmt.Sprintf("%d", len(body)),
	}
	writeResponseWithHeaders(conn, statusCode, statusText, body, headers)
}

func writeResponseWithHeaders(conn net.Conn, statusCode int, statusText, body string, headers map[string]string) {
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText)
	for key, value := range headers {
		response += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	response += "\r\n" + body
	conn.Write([]byte(response))
}

func getRootAndSubpath(uri string) (string, string) {
	pathSplits := strings.Split(uri, "/")
	root := "/" + pathSplits[1]
	subpath := ""
	if len(pathSplits) > 2 {
		subpath = strings.Join(pathSplits[2:], "/")
	}
	return root, subpath
}
