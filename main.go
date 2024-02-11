package main

import (
	"log"
	"track_proxy/cert_handler"
	"track_proxy/connection_handler"

	tls "github.com/refraction-networking/utls"
)

func main() {

	cert := cert_handler.LoadCert("server.crt", "server.key")
	tlsConfig := &tls.Config{
		ServerName: "localhost",
		NextProtos: []string{
			"h2",
		},
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	certFile, certKey, err := cert_handler.LoadX509KeyPair("RootCA.pem", "RootCA.key")
	if err != nil {
		log.Fatalf("Error loading cert %v", err)

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
			success := connection_handler.HandleConnection(conn, certFile, certKey)
			log.Printf("Connecion %v success status: %v \n", conn.RemoteAddr().String(), success)
		}()
	}
}
