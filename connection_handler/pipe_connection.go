package connection_handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"track_proxy/frames_parser"
)

func forwardData(src, dst net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	buffer := make([]byte, 1024)
	var bytesCounter int
	var buf bytes.Buffer

	teeReader := io.TeeReader(src, &buf)

	for {
		n, err := teeReader.Read(buffer)
		bytesCounter += n
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from connection:", err)
			}

			log.Println("EOF from connection", src.RemoteAddr().String())
			break
		}

		log.Printf("Processed data (%s -> %s, %d bytes)\n", src.RemoteAddr().String(), dst.RemoteAddr().String(), n)
		_, err = dst.Write(buffer[:n])
		if err != nil {
			fmt.Println("Error writing to connection:", err)
			break
		}
	}

	log.Println("Total bytes transferred:", bytesCounter)
	frames_parser.ParseFrames(nil, &buf)
	log.Println("Closing connection to", dst.RemoteAddr().String())
	dst.Close()
}

func PipeHttp2(srcConn net.Conn, destConn net.Conn, framesChannel chan frames_parser.Http2Frame) error {

	var wg sync.WaitGroup
	wg.Add(2)

	go forwardData(srcConn, destConn, &wg)
	go forwardData(destConn, srcConn, &wg)

	wg.Wait()
	return nil
}
