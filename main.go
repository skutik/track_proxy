package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	tls "github.com/refraction-networking/utls"
)

func apiHandler(w http.ResponseWriter, r *http.Request) {

}

type proxy struct{}

type proxyError struct {
	Err string `json:"error"`
}

const BufferSize = 1024 * 4
const HTTP2_PREFIX = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
const OK_RESPONSE = "HTTP/1.1 200 OK\r\n\r\n"

func (*proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != "CONNECT" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errMsg := fmt.Sprintf("methods except CONNECT are not allowe, used method %s", r.Method)
		log.Printf("%s\n", errMsg)
		payload := &proxyError{
			Err: errMsg,
		}
		json.NewEncoder(w).Encode(payload)
	}

	log.Printf("Request: %v", r)
	log.Printf("HTTP method: %s", r.Method)
}

func loadCert(certFile string, keyFile string) tls.Certificate {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Error %v occured during loading of cert %s and key %s", err, certFile, keyFile)
	}
	return cert
}

func parseHost(data []byte) string {
	strData := string(data)
	parts := strings.Split(strData, "\n")
	connectInfo := strings.TrimSpace(parts[0])
	parts = strings.Split(connectInfo, " ")
	return parts[1]

}

func processConn(readConn net.Conn, writeConn net.Conn, buf []byte) {
	var connBuf []byte
	if buf != nil {
		connBuf = buf
	} else {
		connBuf = make([]byte, BufferSize)
	}
	for {
		n, err := readConn.Read(connBuf)
		if err != nil {
			fmt.Println("Error when reading conn: ", err)
			return
		}
		_, err = writeConn.Write(connBuf[:n])
		if err != nil {
			fmt.Println("Error when writing conn: ", err)
			return
		}
	}
}

func pipe(srcConn, destConn net.Conn) error {
	done := make(chan error, 1)

	cp := func(r, w net.Conn) {
		n, err := io.Copy(r, w)
		fmt.Printf("copied %d bytes from %s to %s", n, r.RemoteAddr(), w.RemoteAddr())
		done <- err
	}

	go cp(srcConn, destConn)
	go cp(destConn, srcConn)
	err1 := <-done
	err2 := <-done

	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}
	return nil
}

func handleConnection(conn net.Conn) bool {

	var buf = make([]byte, BufferSize)
	defer func() {
		conn.Close()
		fmt.Println("Closing client connection")
	}()

	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error when reading contnet", err)
		return false
	}
	fmt.Println("Read content: ", string(buf))

	host := parseHost(buf)

	// tlsConn := conn.(*tls.Conn)

	tlsConn := conn.(*tls.Conn)

	// handshakeBuf := make([]byte, len(tlsConn.hand))

	// handshakeRaw := copy(tlsConn.hand, handshakeBuf)

	// clientHelloSpec := tls.ClientHelloSpec{}

	// handshakeData := clientHelloSpec.FromRaw(handshakeRaw)

	// fmt.Printf("Handshake %v \n", handshakeData)

	_, err = conn.Write([]byte(OK_RESPONSE))
	if err != nil {
		fmt.Println("Error when writing to conn ", err)

	}

	hostConn, err := net.Dial("tcp", host)

	if err != nil {
		fmt.Println("Error when creating connection to ", host)
		return false
	}

	defer func() {
		hostConn.Close()
		fmt.Println("Closing client connection")
	}()

	err = pipe(tlsConn, hostConn)
	if err != nil {
		fmt.Println("Error when reading conn: ", err)

	}

	return true

}

func main() {

	cert := loadCert("server.crt", "server.key")
	tlsConfig := &tls.Config{
		ServerName: "localhost",
		NextProtos: []string{
			"h2",
		},
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	listener, err := tls.Listen("tcp", ":8000", tlsConfig)
	if err != nil {
		log.Fatalf("Error during creation of TCP listener %v", err)
	}

	defer listener.Close()
	log.Printf("Starting server on port 8000\n")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection %v \n", err)
		}

		log.Printf("Processing connection %v \n", conn)
		go func() {
			success := handleConnection(conn)
			log.Printf("Connecion %v success status: %v \n", conn, success)
		}()
	}

	// proxyHandler := &proxy{}
	// err := http.ListenAndServe(":8000", proxyHandler)
	// log.Printf("Creating server\n")
	// if err != nil {
	// 	log.Fatalf("Proxy failed: %v", err)
	// }
}
