package connection_handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

func forwardData(src, dst net.Conn, wg *sync.WaitGroup, bufferChan chan bytes.Buffer) {
	defer wg.Done()

	buffer := make([]byte, 1024)
	var bytesCounter int
	var framerBuffer bytes.Buffer

	teeReader := io.TeeReader(src, &framerBuffer)
	for {
		n, err := teeReader.Read(buffer)
		bytesCounter += n
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from connection:", err)
				break
			}
			log.Printf("%s received EOF\n", src.RemoteAddr().String())
			// dst.Write([]byte{})
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
	log.Println("Data:", framerBuffer.Bytes())
	bufferChan <- framerBuffer
	log.Println("Closing connection to", dst.RemoteAddr().String())
	dst.Close()
}

func PipeHttp2(srcConn net.Conn, destConn net.Conn, externalWg *sync.WaitGroup, srcBuffer, dstBuffer chan bytes.Buffer) error {
	defer externalWg.Done()
	var wg sync.WaitGroup
	wg.Add(2)

	go forwardData(srcConn, destConn, &wg, srcBuffer)
	go forwardData(destConn, srcConn, &wg, dstBuffer)

	wg.Wait()
	return nil
}
