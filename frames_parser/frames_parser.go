package frames_parser

import (
	"io"
	"log"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

type Http2Frame struct {
	FrameType http2.FrameType
	Data      string
	StreamEnd bool
}

const HTTP2_PREFIX = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

func ParseFrames(w io.Writer, r io.Reader) ([]Http2Frame, error) {
	var frames []Http2Frame
	http2PrefixBuffer := make([]byte, len(HTTP2_PREFIX))
	_, err := r.Read(http2PrefixBuffer)
	if err != nil {
		log.Println("Error reading conn while parsing frames", err)
		return nil, err
	}

	if string(http2PrefixBuffer) != HTTP2_PREFIX {
		log.Println("Not HTTP2 request")
		return nil, err
	}

	framer := http2.NewFramer(w, r)
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
			for _, h := range hf {
				log.Printf("%s\n", h.Name+":"+h.Value)
			}

		case *http2.DataFrame:
			payload = fType.Data()
			httpFrame.Data = string(payload)

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
	return frames, nil
}
