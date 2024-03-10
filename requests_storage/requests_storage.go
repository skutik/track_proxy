package requests_storage

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"
	"track_proxy/client_hello"

	http "github.com/bogdanfinn/fhttp"
)

type RequestRecord struct {
	Method             string                         `json:"method"`
	HttpVersion        string                         `json:"httpVersion"`
	Url                string                         `json:"url"`
	Headers            map[string][]string            `json:"headers"`
	Host               string                         `json:"host"`
	RequestBody        []byte                         `json:"body"`
	StartTimestamp     int64                          `json:"startTimestamp"`
	FinishTimestamp    int64                          `json:"finishTimestamp"`
	Schema             string                         `json:"schema"`
	HttpSetting        map[string]string              `json:"httpSetting"`
	HttpWindowUpdate   int                            `json:"httpWindowUpdate"`
	ClientHello        []client_hello.ClientHelloData `json:"clientHello"`
	HeadersOrder       []string                       `json:"headersOrder"`
	PseudoHeadersOrder []string                       `json:"presudoHeadersOrder"`
}

type ResponseRecord struct {
	StatusCode   int                 `json:"statusCode"`
	HttpVersion  string              `json:"httpVersion"`
	Headers      map[string][]string `json:"headers"`
	ResponseBody []byte              `json:"body"`
}

type Request struct {
	Request  RequestRecord
	Response ResponseRecord
}

// TODO: Move request process to connection handler
func (req *RequestRecord) ProcessRequest() (*http.Response, error) {

	emptyResp := http.Response{}

	// transport := &http.Transport{}
	c := &http.Client{
		// Transport: transport,
	}

	bodyReader := bytes.NewReader(req.RequestBody)
	r, err := http.NewRequest(req.Method, req.Url, bodyReader)
	if err != nil {
		return &emptyResp, fmt.Errorf("Cannot create new request for %s %s [%s]", req.Method, req.Url, req.HttpVersion)
	}

	r.Header = req.Headers
	r.Header[http.HeaderOrderKey] = req.HeadersOrder
	r.Header[http.PHeaderOrderKey] = req.PseudoHeadersOrder

	req.StartTimestamp = time.Now().Unix()
	resp, err := c.Do(r)
	req.FinishTimestamp = time.Now().Unix()
	if err != nil {
		return &emptyResp, fmt.Errorf("Error when processing request %s %s [%s]", req.Method, req.Url, req.HttpVersion)
	}
	return resp, nil
}

func ResponseRecordFromResponse(res *http.Response) ResponseRecord {
	responseRecord := ResponseRecord{}
	responseRecord.StatusCode = res.StatusCode

	responseRecord.Headers = make(map[string][]string)
	for headerName, headerValue := range res.Header {
		responseRecord.Headers[headerName] = headerValue
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("Error reading response body", err)
	}

	responseRecord.HttpVersion = res.Proto
	responseRecord.ResponseBody = bodyBytes
	return responseRecord
}

// func RequestFromFrames(frames []frames_parser.Http2Frame) (RequestType, error) {
// 	parsedRequest := RequestType{}
// 	for frame := range frames {
// 		log.Println("Processing frame: ", frame)
// 	}

// 	return parsedRequest, fmt.Errorf("unable to parse request from frames")
// }