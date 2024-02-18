package connection_handler

import (
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
	"track_proxy/cert_handler"
	"track_proxy/client_hello"
	"track_proxy/frames_parser"

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

func preCheckClientHelloData(rawData []byte) []byte {
	ssl_record_type := client_hello.SslRecordType[uint16(rawData[0])]
	ssl_version_major := rawData[1]
	ssl_version_minor := rawData[2]
	ssl_version := client_hello.SslVersion[uint16(ssl_version_minor)|uint16(ssl_version_major)<<8]
	client_hello_size := uint16(rawData[4]) | uint16(5)<<8

	log.Printf("SSL data headers:\n - record type: %s,\n - SSL version: %s,\n - record lenght: %d\n", ssl_record_type, ssl_version, client_hello_size)
	return rawData[5:]
}

func handleHttp() {}

func handleHttp2(conn *tls.Conn) bool {
	frames, req, err := frames_parser.ParseFrames(nil, conn)
	if err != nil {
		log.Println("Error when parsing frames")
	}

	log.Println("Parsed frames:", frames)
	log.Println("Parsed req:", req)
	return true
}

func HandleConnection(conn net.Conn, cert *x509.Certificate, key any) bool {

	var buf = make([]byte, BufferSize)
	defer func() {
		conn.Close()
		log.Println("Closing client connection")
	}()

	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error when reading contnet", err)
		return false
	}
	fmt.Println("Read content: ", string(buf))

	host := parseHost(buf)
	var hostDomain string
	if strings.Contains(host, ":") {
		hostDomain, _, err = net.SplitHostPort(host)
		if err != nil {
			log.Fatal("err splitting host ", host)
		}
	} else {
		hostDomain = host
	}
	pemCert, pemKey := cert_handler.CreateCert(hostDomain, cert, key, 240)

	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		log.Fatal(err)
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
		log.Println("Error when writing to conn ", err)

	}

	tlsConn := &ClientHelloUtlsConn{Conn: conn.(*tls.Conn)}
	tlsServerConn := tls.Server(tlsConn, tlsConfig)

	defer func() {
		tlsServerConn.Close()
		log.Println("closing TLS server conn")
	}()

	http2PrefixLen := len(frames_parser.HTTP2_PREFIX)
	requestBuffer := make([]byte, http2PrefixLen)

	_, err = tlsServerConn.Read(requestBuffer)
	if err != nil {
		fmt.Println("Error when reading contnet", err)
		return false
	}

	if string(requestBuffer) == frames_parser.HTTP2_PREFIX {
		handleHttp2(tlsServerConn)
		return true
	} else {
		// handleHttp()
		log.Println("HTTP1 request")
		return true
	}

	reqBuf := make([]byte, BufferSize)
	for {
		_, err := tlsServerConn.Read(reqBuf)
		if err != nil {
			fmt.Println("Error when reading contnet", err)
			break
		}
		log.Println("Read content: ", string(reqBuf))
	}

	hostConn, err := tls.Dial("tcp", host, &tls.Config{})
	if err != nil {
		log.Println("Error when creating connection to ", host)
		return false
	}

	framesChannel := make(chan frames_parser.Http2Frame)

	PipeHttp2(tlsServerConn, hostConn, framesChannel)
	client_hello_raw := preCheckClientHelloData(tlsConn.ClientHelloRaw)

	client_hello, err := client_hello.UnmarshallClientHello(client_hello_raw)
	if err != nil {
		log.Println("Error parsing Client Hello ", err)
		return true
	}

	log.Println("Parsed client:", client_hello)

	return true
}
