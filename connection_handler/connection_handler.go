package connection_handler

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"track_proxy/cert_handler"
	"track_proxy/client_hello"
	"track_proxy/frames_parser"
	"track_proxy/requests_storage"

	http "github.com/bogdanfinn/fhttp"
	tls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
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

func prepareHeaders(response *requests_storage.ResponseRecord) []byte {
	var headersBuffer bytes.Buffer

	encoder := hpack.NewEncoder(&headersBuffer)
	encoder.WriteField(hpack.HeaderField{Name: ":status", Value: strconv.Itoa(response.StatusCode)})
	if response.Body != nil {
		encoder.WriteField(hpack.HeaderField{Name: "content-length", Value: strconv.Itoa(len(response.Body))})
	}
	for key, values := range response.Headers {
		key = strings.ToLower(key)
		if key == "content-encoding" {
			continue
		}
		for _, value := range values {
			log.Println("adding header:", key, value)
			encoder.WriteField(hpack.HeaderField{Name: key, Value: value})
		}
	}
	return headersBuffer.Bytes()
}

// func writeHeadersFrame(framer *http2.Framer, streamID uint32, headers []byte) {
// 	err := framer.WriteHeaders(http2.HeadersFrameParam{
// 		StreamID:      streamID,
// 		EndHeaders:    true,
// 		BlockFragment: headers,
// 	})
// 	if err != nil {
// 		log.Println("Error when writing headers to conn", err)
// 	}
// }

// func writeDataFrame(framer *http2.Framer, streamID uint32, data []byte) {
// 	err := framer.WriteData(streamID, true, data)
// 	if err != nil {
// 		log.Println("Error when writing data to conn", err)
// 	}
// }

// func handleHttp2(conn *tls.Conn) (*requests_storage.Request, error) {
// 	framer := http2.NewFramer(conn, conn)
// 	frames, reqRecord, err := frames_parser.ParseFrames(framer)
// 	framer.WriteRawFrame(http2.FrameSettings, 0, 0, []byte{})
// 	if err != nil {
// 		log.Println("Error when parsing frames")
// 	}

// 	log.Println("Parsed frames:", frames)
// 	log.Println("Parsed req:", reqRecord)

// 	resp, err := reqRecord.ProcessRequest()

// 	if err != nil {
// 		log.Println("Error processing request:", resp)
// 		return &requests_storage.Request{}, fmt.Errorf(err.Error())
// 	}
// 	respRecord := requests_storage.ResponseRecordFromResponse(resp)
// 	log.Println("Response record:", respRecord.Headers)

// 	// writeHeadersFrame(framer, 1, prepareHeaders(&respRecord))

// 	// stringResponse := stringifyResponse(resp, respRecord.ResponseBody)
// 	// log.Println("Http response:\n", stringResponse)
// 	// conn.Write([]byte(stringResponse))
// 	err = framer.WriteHeaders(http2.HeadersFrameParam{
// 		StreamID:      1,
// 		EndHeaders:    true,
// 		BlockFragment: prepareHeaders(&respRecord),
// 		EndStream:     respRecord.Body == nil,
// 	})
// 	if err != nil {
// 		log.Println("Error when writing headers to conn", err)
// 	}

// 	if respRecord.Body != nil {
// 		err := framer.WriteData(1, true, respRecord.Body)
// 		if err != nil {
// 			log.Println("Error when writing data to conn", err)
// 		}
// 	}
// 	return &requests_storage.Request{
// 		Request:  reqRecord,
// 		Response: respRecord,
// 	}, nil
// }

func stringifyResponse(res *http.Response, body []byte) string {
	var responseString strings.Builder
	fmt.Fprintf(&responseString, "%s %d\r\n", strings.TrimRight(res.Proto, ".0"), res.StatusCode)
	res.Header.Write(&responseString)
	if body != nil {
		fmt.Fprintf(&responseString, "\r\n%s", body)
	}

	return responseString.String()
}

func HandleConnection(conn net.Conn, cert *x509.Certificate, key any, storage []requests_storage.Request) bool {

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

	hostConn, err := tls.Dial("tcp", host, &tls.Config{})
	if err != nil {
		log.Println("Error when creating connection to ", host)
		return false
	}

	defer func() {
		log.Println("Closing connection to host", host)
		defer hostConn.Close()

	}()

	if string(requestBuffer) == http2.ClientPreface {

		// incomingFramesChan := make(chan frames_parser.Http2Frame)
		// outcomingFramesChan := make(chan frames_parser.Http2Frame)

		log.Println("Starting pipe")
		hostConn.Write([]byte(http2.ClientPreface))

		srcBuffer := make(chan bytes.Buffer)
		dstBuffer := make(chan bytes.Buffer)

		var wg sync.WaitGroup
		var request requests_storage.Request

		wg.Add(1)
		go PipeHttp2(tlsServerConn, hostConn, &wg, srcBuffer, dstBuffer)

		for {
			select {
			case srcStream := <-srcBuffer:
				log.Println("Processing src stream")
				fr := http2.NewFramer(nil, &srcStream)
				frames, rec, err := frames_parser.ParseFrames(fr)
				request.Request = requests_storage.RequestRecordFromUknown(rec)
				if err != nil {
					log.Println("Error when parsing src frames:", err)
				}

				log.Println("Parsed src frames", frames)

			case dstStream := <-dstBuffer:
				log.Println("Processing dst stream")
				fr := http2.NewFramer(nil, &dstStream)
				frames, rec, err := frames_parser.ParseFrames(fr)
				request.Response = requests_storage.ResponseRecordFromUknown(rec)
				if err != nil {
					log.Println("Error when parsing src frames:", err)
				}

				log.Println("Parsed dst frames", frames)
			default:
				log.P
			}
		}

		// request, err := handleHttp2(tlsServerConn)
		// storage = append(storage, *request)
		// if err != nil {
		// 	log.Println("Error when processing request")
		// 	return false
		// }
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

	client_hello_raw := preCheckClientHelloData(tlsConn.ClientHelloRaw)

	client_hello, err := client_hello.UnmarshallClientHello(client_hello_raw)
	if err != nil {
		log.Println("Error parsing Client Hello ", err)
		return true
	}

	log.Println("Parsed client:", client_hello)

	return true
}
