package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
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
	tlsConfig := &tls.Config{
		ServerName: strings.Split(host, ":")[0],
	}
	tlsConn := tls.Client(hostConn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		fmt.Println("Error during destination handshake")

	}

	_, err = conn.Read(buf)
	if err != nil {
		fmt.Println("Error when reading conn: ", err)
		return false
	}
	fmt.Println("Data: ", buf)

	// _, err = conn.Read(buf)
	// if err != nil {
	// 	fmt.Println("Error when reading contnet", err)
	// 	return false
	// }
	// fmt.Println("Read content: ", string(buf))

	go processConn(conn, hostConn, buf)
	// go processConn(hostConn, conn, nil)

	// return false

	// _, err := conn.Read(initBuf)
	// if err != nil {
	// 	log.Printf("Error reading conn: %v \n", err)
	// 	return false
	// }
	// log.Printf("Reading request: %s", string(buf))

	// tlsConn := conn.(*tls.Conn)

	// state := tlsConn.ConnectionState()

	// log.Printf("Hello client: %v", tlsConn)
	// log.Printf("Hello state: %v", state)

	return false
}

func main() {

	// cert := loadCert("server.crt", "server.key")
	// tlsConfig := &tls.Config{
	// 	ServerName: "localhost",
	// 	NextProtos: []string{
	// 		"h2",
	// 	},
	// 	Certificates:       []tls.Certificate{cert},
	// 	InsecureSkipVerify: true,
	// }

	listener, err := net.Listen("tcp", ":8000")
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
