package frames_parser

import (
	"bytes"
	"log"
	"net/url"
	"strings"
	"track_proxy/requests_storage"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"

	http "github.com/bogdanfinn/fhttp"
)

type Http2Frame struct {
	FrameType http2.FrameType
	Data      string
	StreamEnd bool
}

type Http2Settings struct {
}

const HTTP2_PREFIX = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

func requestFromFrames(frames []Http2Frame) http.Request {
	return http.Request{}
}

func parseHttpSetting(setting string) (string, string) {
	parts := strings.Split(setting, " = ")
	return strings.TrimLeft(parts[0], "["), strings.TrimRight(parts[1], "]")
}

func ParseFrames(framer *http2.Framer) ([]Http2Frame, *requests_storage.UnknownRecord, error) {
	var frames []Http2Frame

	var rec requests_storage.UnknownRecord
	rec.HttpVersion = "HTTP2"
	rec.HttpSetting = make(map[string]string)
	for {
		frame, err := framer.ReadFrame()
		httpFrame := Http2Frame{}
		if err != nil {
			log.Println("Error reading frame", err)
			break
		}
		var payload []byte = nil

		httpFrame.FrameType = frame.Header().Type
		switch fType := frame.(type) {
		case *http2.HeadersFrame:
			decoder := hpack.NewDecoder(2048, nil)
			hf, _ := decoder.DecodeFull(fType.HeaderBlockFragment())
			rec.Headers = make(map[string][]string)

			reqUrl := &url.URL{}
			for _, h := range hf {
				if strings.HasPrefix(h.Name, ":") {
					rec.PseudoHeadersOrder = append(rec.PseudoHeadersOrder, h.Name)
					switch h.Name {
					case ":method":
						rec.Method = h.Value
					case ":scheme":
						reqUrl.Scheme = h.Value
					case ":authority":
						reqUrl.Host, rec.Host = h.Value, h.Value
					case ":path":
						reqUrl.Path = h.Value
					default:
						log.Println("Unexpected pseudo header: ", h.Name)
					}
					continue
				}
				_, exists := rec.Headers[h.Name]
				if exists {
					rec.Headers[h.Name] = append(rec.Headers[h.Name], h.Value)
				} else {
					rec.Headers[h.Name] = []string{h.Value}
					rec.HeadersOrder = append(rec.HeadersOrder, strings.ToLower(h.Name))
				}
			}

			rec.Url = reqUrl.String()

		case *http2.WindowUpdateFrame:
			rec.HttpWindowUpdate = int(fType.Increment)

		case *http2.SettingsFrame:
			fType.ForeachSetting(func(setting http2.Setting) error {
				settingName, settingValue := parseHttpSetting(setting.String())
				rec.HttpSetting[settingName] = settingValue
				return nil
			})

		case *http2.DataFrame:
			payload = fType.Data()
			rec.Body = payload

		default:
			log.Println("Unexpected Frame type", fType)
		}

		if frame.Header().Flags.Has(http2.FlagDataEndStream) {
			httpFrame.StreamEnd = true
			frames = append(frames, httpFrame)
			log.Println("HTTP2 stream ended")
		}
		httpFrame.StreamEnd = false
		frames = append(frames, httpFrame)

	}
	return frames, &rec, nil
}

func parseframesBytes(bytes *bytes.Buffer) ([]Http2Frame, *requests_storage.UnknownRecord, error) {
	fr := http2.NewFramer(nil, bytes)
	return ParseFrames(fr)
}

func ParseResponseFrames(bytes *bytes.Buffer) *requests_storage.ResponseRecord {
	req := &requests_storage.ResponseRecord{}
	_, record, err := parseframesBytes(bytes)
	if err != err {
		log.Println("Error parsing response frames:", err)
		return req
	}
	return requests_storage.ResponseRecordFromUknown(record)
}

func ParseRequestFrames(bytes *bytes.Buffer) *requests_storage.RequestRecord {
	res := &requests_storage.RequestRecord{}
	_, record, err := parseframesBytes(bytes)
	if err != err {
		log.Println("Error parsing request frames:", err)
		return res
	}
	return requests_storage.RequestRecordFromUknown(record)

}
