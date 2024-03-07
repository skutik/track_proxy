package frames_parser

import (
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

func ParseFrames(framer *http2.Framer) ([]Http2Frame, requests_storage.RequestRecord, error) {
	var frames []Http2Frame

	var req requests_storage.RequestRecord
	req.HttpVersion = "HTTP2"
	req.HttpSetting = make(map[string]string)
	for {
		frame, err := framer.ReadFrame()
		httpFrame := Http2Frame{}
		if err != nil {
			log.Println("Error reading frame", err)
			break
		}
		log.Println("Frame data:", frame)
		log.Println("Frame header:", frame.Header().Type)
		log.Println("Frame flags:", frame.Header().Flags)

		var payload []byte = nil

		httpFrame.FrameType = frame.Header().Type
		switch fType := frame.(type) {
		case *http2.HeadersFrame:
			decoder := hpack.NewDecoder(2048, nil)
			hf, _ := decoder.DecodeFull(fType.HeaderBlockFragment())
			req.Headers = make(map[string][]string)

			log.Println("Headers streamID", frame.Header().StreamID)

			reqUrl := &url.URL{}
			for _, h := range hf {
				log.Printf("Found header: %s\n", h.Name+":"+h.Value)
				if strings.HasPrefix(h.Name, ":") {
					req.PseudoHeadersOrder = append(req.PseudoHeadersOrder, h.Name)
					switch h.Name {
					case ":method":
						req.Method = h.Value
					case ":scheme":
						// scheme = h.Value
						reqUrl.Scheme = h.Value
					case ":authority":
						reqUrl.Host, req.Host = h.Value, h.Value
					case ":path":
						reqUrl.Path = h.Value
					default:
						log.Println("Unexpected pseudo header: ", h.Name)
					}
					continue
				}
				_, exists := req.Headers[h.Name]
				if exists {
					req.Headers[h.Name] = append(req.Headers[h.Name], h.Value)
				} else {
					req.Headers[h.Name] = []string{h.Value}
					req.HeadersOrder = append(req.HeadersOrder, strings.ToLower(h.Name))
				}
			}

			req.Url = reqUrl.String()

		case *http2.WindowUpdateFrame:
			log.Println("Frame update: ", fType.Increment)
			req.HttpWindowUpdate = int(fType.Increment)

		case *http2.SettingsFrame:
			fType.ForeachSetting(func(setting http2.Setting) error {
				settingName, settingValue := parseHttpSetting(setting.String())
				req.HttpSetting[settingName] = settingValue
				return nil
			})

		case *http2.DataFrame:
			payload = fType.Data()
			log.Println("Data streamID", frame.Header().StreamID)
			req.RequestBody = payload

		default:
			log.Println("Unexpected Frame type", fType)
		}

		log.Println("Frame parsed")
		if frame.Header().Flags.Has(http2.FlagDataEndStream) {
			httpFrame.StreamEnd = true
			frames = append(frames, httpFrame)
			log.Println("HTTP2 stream ended")
			break
		}
		httpFrame.StreamEnd = false
		frames = append(frames, httpFrame)

	}
	return frames, req, nil
}
