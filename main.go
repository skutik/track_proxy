package main

import (
	"log"
	"track_proxy/api_handler"
	"track_proxy/cert_handler"
	"track_proxy/connection_handler"

	"github.com/gin-gonic/gin"
	tls "github.com/refraction-networking/utls"
)

func listenProxy(addr string) {
	cert := cert_handler.LoadCert("server.crt", "server.key")
	tlsConfig := &tls.Config{
		ServerName: "localhost",
		NextProtos: []string{
			"h2", "h1/1",
		},
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	certFile, certKey, err := cert_handler.LoadX509KeyPair("RootCA.pem", "RootCA.key")
	if err != nil {
		log.Fatalf("Error loading cert %v", err)

	}
	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		log.Fatalln("Error during creation of TCP listener ", err)
	}
	defer listener.Close()

	log.Println("Starting proxy server on addr", addr)

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

func listenServer(addr string) {
	router := gin.Default()

	router.GET("/ping", api_handler.Ping)
	router.GET("/requests", api_handler.GetRequests)

	log.Println("Starting gin server on addr", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalln("Error starting gin server:", err)
	}
}

func main() {
	go listenProxy(":8000")
	go listenServer(":8001")

	select {}
}
