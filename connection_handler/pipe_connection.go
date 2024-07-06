package connection_handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
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

	log.Println("Total bytes transferred:", bytesCounter, "("+src.RemoteAddr().String()+" -> "+dst.RemoteAddr().String()+")")
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

	request := requests_storage.NewRequest()
	go forwardData(srcConn, destConn, &wg, srcBufferChan)
	go forwardData(destConn, srcConn, &wg, dstBufferChan)
	for i := 0; i < 2; i++ {
		select {
		case srcBuffer := <-srcBufferChan:
			log.Println("Processing src buffer")
			buff := make([]byte, len(http2.ClientPreface))
			_, err := srcBuffer.Read(buff)
			if err != nil {
				log.Println("error reading src stream")
			}

			var parsedRequest requests_storage.RequestRecord
			if string(buff) == http2.ClientPreface {
				parsedRequest = *frames_parser.ParseRequestFrames(&srcBuffer)
			} else {
				srcBytes := append(buff, srcBuffer.Bytes()...)
				parsedRequest = *request_parser.ParseHttpRequest(bytes.NewBuffer(srcBytes))
			}
			request.Request.Method = parsedRequest.Method
			request.Request.HttpVersion = parsedRequest.HttpVersion
			request.Request.Url = parsedRequest.Url
			request.Request.Headers = parsedRequest.Headers
			request.Request.Host = parsedRequest.Host
			request.Request.Body = parsedRequest.Body
			request.Request.Schema = parsedRequest.Schema
			request.Request.HttpSetting = parsedRequest.HttpSetting
			request.Request.HttpWindowUpdate = parsedRequest.HttpWindowUpdate
			request.Request.HeadersOrder = parsedRequest.HeadersOrder
			request.Request.PseudoHeadersOrder = parsedRequest.PseudoHeadersOrder
			request.Request.Error = parsedRequest.Error

			requestProtocol = request.Request.HttpVersion

		case dstBuffer := <-dstBufferChan:
			request.Request.FinishTimestamp = time.Now().UnixNano()
			log.Println("Processing dst buffer")
			if requestProtocol == "HTTP/1.1" || strings.HasPrefix(dstBuffer.String(), "HTTP/1.1") {
				request.Response = *request_parser.ParseHttpResponse(&dstBuffer)
			} else {
				request.Response = *frames_parser.ParseResponseFrames(&dstBuffer)
			}
		}
	}

	requestChan <- request
	wg.Wait()
}
