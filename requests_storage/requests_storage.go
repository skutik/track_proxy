package requests_storage

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"track_proxy/client_hello"

	http "github.com/bogdanfinn/fhttp"
	"github.com/google/uuid"
)

var (
	Lock   sync.Mutex
	RwLock sync.RWMutex
)
var Storage = make(RequestStorage)

type RequestStorage map[string]Request

type RequestError struct{}

type RequestRecord struct {
	Method             string                       `json:"method"`
	HttpVersion        string                       `json:"httpVersion"`
	Url                string                       `json:"url"`
	Headers            map[string][]string          `json:"headers"`
	Host               string                       `json:"host"`
	Body               []byte                       `json:"body"`
	StartTimestamp     int64                        `json:"startTimestamp"`
	FinishTimestamp    int64                        `json:"finishTimestamp"`
	Schema             string                       `json:"schema"`
	HttpSetting        map[string]string            `json:"httpSetting"`
	HttpWindowUpdate   int                          `json:"httpWindowUpdate"`
	ClientHello        client_hello.ClientHelloData `json:"clientHello"`
	HeadersOrder       []string                     `json:"headersOrder"`
	PseudoHeadersOrder []string                     `json:"presudoHeadersOrder"`
	Error              string                       `json:"error"`
}

type ResponseRecord struct {
	StatusCode  int                 `json:"statusCode"`
	HttpVersion string              `json:"httpVersion"`
	Headers     map[string][]string `json:"headers"`
	Body        []byte              `json:"body"`
	Error       string              `json:"error"`
}

type Request struct {
	Id       string         `json:"id"`
	Request  RequestRecord  `json:"request"`
	Response ResponseRecord `json:"response"`
}

type UnknownRecord struct {
	Method             string
	HttpVersion        string
	Url                string
	Headers            map[string][]string
	Host               string
	Body               []byte
	StartTimestamp     int64
	FinishTimestamp    int64
	Schema             string
	HttpSetting        map[string]string
	HttpWindowUpdate   int
	HeadersOrder       []string
	PseudoHeadersOrder []string
	StatusCode         int
	Error              string
}

func NewRequest() Request {
	req := Request{}
	req.Id = uuid.New().String()
	req.Request.StartTimestamp = time.Now().UnixNano()
	return req
}

func (reqStorage RequestStorage) AddRequestToStorage(req Request) error {
	if len(req.Id) == 0 {
		req.Id = uuid.New().String()
	}

	Lock.Lock()
	defer Lock.Unlock()

	_, exists := reqStorage[req.Id]
	if exists {
		return fmt.Errorf("request ID %s is already included in storage", req.Id)
	}
	reqStorage[req.Id] = req
	return nil
}

func (reqStorage RequestStorage) GetRequests() []Request {
	requests, _ := reqStorage.GetRequestSinceId("")
	return requests
}

func (reqStorage RequestStorage) GetRequestSinceId(lastId string) ([]Request, string) {
	requests := []Request{}
	request := Request{}
	i := 0

	RwLock.Lock()
	defer RwLock.Unlock()

	for _, request := range reqStorage {
		requests = append(requests, request)
	}

	if len(requests) == 0 {
		return requests, ""
	}

	sort.Slice(requests, func(i, j int) bool {
		return requests[i].Request.StartTimestamp < requests[j].Request.StartTimestamp
	})

	if len(lastId) != 0 {
		for i, request = range requests {
			if request.Id == lastId {
				break
			}
		}

		if len(requests) == (i + 1) {
			return []Request{}, requests[len(requests)-1].Id
		}

		requests = requests[i+1:]

	}
	return requests, requests[len(requests)-1].Id

}

func (reqStorage RequestStorage) GetRequestById(reqId string) (Request, error) {
	RwLock.Lock()
	defer RwLock.Unlock()
	req, exists := reqStorage[reqId]
	if !exists {
		return Request{}, fmt.Errorf("request with ID %s not found", reqId)
	}
	return req, nil

}

func (reqStorage RequestStorage) GetCurlForRequest(reqId string) (string, error) {
	req, err := reqStorage.GetRequestById(reqId)
	if err != nil {
		return "", err
	}

	return req.Request.GetCurlCommand(), nil
}

func ResponseRecordFromUknown(unknownRecord *UnknownRecord) *ResponseRecord {
	return &ResponseRecord{
		StatusCode:  unknownRecord.StatusCode,
		HttpVersion: unknownRecord.HttpVersion,
		Headers:     unknownRecord.Headers,
		Body:        unknownRecord.Body,
		Error:       unknownRecord.Error,
	}
}

func RequestRecordFromUknown(unknownRecord *UnknownRecord) *RequestRecord {

	return &RequestRecord{
		Method:             unknownRecord.Method,
		HttpVersion:        unknownRecord.HttpVersion,
		Url:                unknownRecord.Url,
		Headers:            unknownRecord.Headers,
		Host:               unknownRecord.Host,
		Body:               unknownRecord.Body,
		Schema:             unknownRecord.Schema,
		HttpSetting:        unknownRecord.HttpSetting,
		HttpWindowUpdate:   unknownRecord.HttpWindowUpdate,
		HeadersOrder:       unknownRecord.HeadersOrder,
		PseudoHeadersOrder: unknownRecord.PseudoHeadersOrder,
		Error:              unknownRecord.Error,
	}
}

// TODO: Move request process to connection handler
func (req *RequestRecord) ProcessRequest() (*http.Response, error) {

	emptyResp := http.Response{}

	// transport := &http.Transport{}
	c := &http.Client{
		// Transport: transport,
	}

	bodyReader := bytes.NewReader(req.Body)
	r, err := http.NewRequest(req.Method, req.Url, bodyReader)
	if err != nil {
		return &emptyResp, fmt.Errorf("Cannot create new request for %s %s [%s]", req.Method, req.Url, req.HttpVersion)
	}

	r.Header = req.Headers
	r.Header[http.HeaderOrderKey] = req.HeadersOrder
	r.Header[http.PHeaderOrderKey] = req.PseudoHeadersOrder

	req.StartTimestamp = time.Now().UnixNano()
	resp, err := c.Do(r)
	req.FinishTimestamp = time.Now().UnixNano()
	if err != nil {
		return &emptyResp, fmt.Errorf("Error when processing request %s %s [%s]", req.Method, req.Url, req.HttpVersion)
	}
	return resp, nil
}

func (req *RequestRecord) GetCurlCommand() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("curl -X %s '%s'", req.Method, req.Url))
	for headerName, headerValues := range req.Headers {
		if strings.HasPrefix(headerName, ":") {
			continue
		}

		for _, headerValue := range headerValues {
			builder.WriteString(fmt.Sprintf(" -H '%s: %s'", headerName, headerValue))
		}
	}

	if len(req.Body) > 0 {
		builder.WriteString(fmt.Sprintf(" --data '%s'", req.Body))
	}
	builder.WriteString(fmt.Sprintf(" --%s", strings.ToLower(req.HttpVersion)))
	return builder.String()
}

// func ResponseRecordFromResponse(res *Response) ResponseRecord {
// 	responseRecord := ResponseRecord{}
// 	responseRecord.StatusCode = res.StatusCode

// 	responseRecord.Headers = make(map[string][]string)
// 	for headerName, headerValue := range res.Header {
// 		responseRecord.Headers[headerName] = headerValue
// 	}
// 	defer res.Body.Close()

// 	bodyBytes, err := io.ReadAll(res.Body)
// 	if err != nil {
// 		log.Println("Error reading response body", err)
// 	}

// 	responseRecord.HttpVersion = res.Proto
// 	responseRecord.Body = bodyBytes
// 	return responseRecord
// }

// func RequestFromFrames(frames []frames_parser.Http2Frame) (RequestType, error) {
// 	parsedRequest := RequestType{}
// 	for frame := range frames {
// 		log.Println("Processing frame: ", frame)
// 	}

// 	return parsedRequest, fmt.Errorf("unable to parse request from frames")
// }
