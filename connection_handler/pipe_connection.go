package connection_handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"track_proxy/frames_parser"
	"track_proxy/request_parser"
	"track_proxy/requests_storage"

	"golang.org/x/net/http2"
)

func forwardData(src, dst net.Conn, wg *sync.WaitGroup, bufferChan chan bytes.Buffer) {
	defer wg.Done()

	buffer := make([]byte, 1024)
	var bytesCounter int
	var cpBuffer bytes.Buffer

	teeReader := io.TeeReader(src, &cpBuffer)
	for {
		n, err := teeReader.Read(buffer)
		bytesCounter += n
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from connection:", err)
				break
			}
			log.Printf("%s received EOF\n", src.RemoteAddr().String())
			break
		}

		if n == 0 {
			log.Println("No data to process")
			break
		}

		log.Printf("Processed data (%s -> %s, %d bytes)\n", src.RemoteAddr().String(), dst.RemoteAddr().String(), n)
		_, err = dst.Write(buffer[:n])
		if err != nil {
			fmt.Println("Error writing to connection:", err)
			break
		}
	}

	log.Println("Total bytes transferred:", bytesCounter, src.RemoteAddr().String(), dst.RemoteAddr().String())
	bufferChan <- cpBuffer
	log.Println("Closing connection to", dst.RemoteAddr().String())
	dst.Close()
}

func PipeHttp(srcConn net.Conn, destConn net.Conn, externalWg *sync.WaitGroup, requestChan chan requests_storage.Request) {
	defer externalWg.Done()
	var wg sync.WaitGroup

	srcBufferChan := make(chan bytes.Buffer)
	dstBufferChan := make(chan bytes.Buffer)

	var requestProtocol string

	wg.Add(2)

	go forwardData(srcConn, destConn, &wg, srcBufferChan)
	go forwardData(destConn, srcConn, &wg, dstBufferChan)

	var request requests_storage.Request
	for i := 0; i < 2; i++ {
		select {
		case srcBuffer := <-srcBufferChan:
			log.Println("Processing src buffer")
			if strings.HasPrefix(srcBuffer.String(), http2.ClientPreface) {
				request.Request = *frames_parser.ParseRequestFrames(&srcBuffer)
			} else {
				request.Request = *request_parser.ParseHttpRequest(&srcBuffer)
			}
			requestProtocol = request.Request.HttpVersion

		case dstBuffer := <-dstBufferChan:
			log.Println("Processing dst buffer")
			if requestProtocol == "HTTP/1.1" || strings.HasPrefix(dstBuffer.String(), "HTTP/1.1") {
				request.Response = *request_parser.ParseHttpResponse(&dstBuffer)
			} else {
				request.Response = *frames_parser.ParseResponseFrames(&dstBuffer)
			}
		}
	}

	log.Println("Reqest:", request)
	requestChan <- request
	wg.Wait()
}
