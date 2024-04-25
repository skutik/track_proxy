package connection_handler

import (
	"bufio"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"track_proxy/cert_handler"
	"track_proxy/client_hello"
	"track_proxy/requests_storage"

	tls "github.com/refraction-networking/utls"
)

const BufferSize = 1024 * 4
const OK_RESPONSE = "HTTP/1.1 200 OK\r\n\r\n"
const HOST_TIMEOUT = time.Second * 30

type ClientHelloUtlsConn struct {
	*tls.Conn
	ClientHelloRaw []byte
}

func (c *ClientHelloUtlsConn) Read(b []byte) (int, error) {
	if len(c.ClientHelloRaw) == 0 {
		// Capture the first read, which should include the ClientHello
		n, err := c.Conn.Read(b)
		if err != nil {
			return n, err
		}
		c.ClientHelloRaw = append(c.ClientHelloRaw, b[:n]...)
		return n, err
	}
	return c.Conn.Read(b)
}

func parseHost(data []byte) string {
	strData := string(data)
	parts := strings.Split(strData, "\n")
	connectInfo := strings.TrimSpace(parts[0])
	parts = strings.Split(connectInfo, " ")
	return parts[1]
}

func handleConnectRequest(conn net.Conn, cert *x509.Certificate, key any, req *http.Request, errChan chan error) {
	host := req.Host
	var hostDomain string
	var err error
	if strings.Contains(host, ":") {
		hostDomain, _, err = net.SplitHostPort(host)
		if err != nil {
			errChan <- err
			return
		}
	} else {
		hostDomain = host
	}
	pemCert, pemKey := cert_handler.CreateCert(hostDomain, cert, key, 240)

	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		errChan <- err
		return
	}

	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:               tls.VersionTLS13,
		Certificates:             []tls.Certificate{tlsCert},
		NextProtos: []string{
			"h2", "http/1.1",
		},
		InsecureSkipVerify: true,
	}

	_, err = conn.Write([]byte(OK_RESPONSE))
	if err != nil {
		errChan <- fmt.Errorf("error when writing to conn %v", err)
		return
	}

	tlsConn := &ClientHelloUtlsConn{Conn: conn.(*tls.Conn)}
	tlsServerConn := tls.Server(tlsConn, tlsConfig)

	defer func() {
		tlsServerConn.Close()
		log.Println("closing TLS server conn")
	}()

	hostConn, err := tls.Dial("tcp", host, &tls.Config{})
	if err != nil {
		errChan <- fmt.Errorf("error when creating connection to %v", host)
		return
	}

	defer func() {
		log.Println("Closing connection to host", host)
		defer hostConn.Close()
	}()

	log.Println("Starting pipe")
	requestChan := make(chan requests_storage.Request)

	var wg sync.WaitGroup

	wg.Add(1)
	go PipeHttp(tlsServerConn, hostConn, &wg, requestChan)
	request := <-requestChan

	clientHelloData, err := client_hello.UnmarshallClientHello(tlsConn.ClientHelloRaw)
	if err != nil {
		log.Println("error parsing client hello data:", err)
	} else {
		request.Request.ClientHello = *clientHelloData
	}
	log.Println("client hello data:", request.Request.ClientHello)
	requests_storage.RwLock.Lock()
	requests_storage.Storage = append(requests_storage.Storage, request)
	requests_storage.RwLock.Unlock()
	close(requestChan)
	wg.Wait()
}

func handlerDirectRequest(conn net.Conn, req *http.Request, errChan chan error) {
	client := http.Client{}
	req.RequestURI = ""
	res, err := client.Do(req)
	if err != nil {
		errChan <- fmt.Errorf("error processing http request: %s", err)
		return
	}
	err = res.Write(conn)
	if err != nil {
		errChan <- fmt.Errorf("error writing to client connection %s", err)
		return
	}
}

func HandleConnection(conn net.Conn, cert *x509.Certificate, key any) bool {
	defer func() {
		conn.Close()
		log.Println("Closing client connection")
	}()

	connReader := bufio.NewReader(conn)
	req, err := http.ReadRequest(connReader)
	if err != nil {
		fmt.Println("Error when reading request", err)
		return false
	}

	errChan := make(chan error)
	if req.Method == http.MethodConnect {
		go handleConnectRequest(conn, cert, key, req, errChan)
	} else {
		go handlerDirectRequest(conn, req, errChan)
	}

	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Println("error processing connection to host", err)
				return false
			}
			return true
		}
	}

	// _, err := conn.Read(buf)
	// if err != nil {
	// 	fmt.Println("Error when reading content", err)
	// 	return false
	// }
	// fmt.Println("Read content: ", string(buf))

	// host := parseHost(buf)
	// host := parseHost([]byte(req.Host))
}
