package client_hello

import (
	"encoding/json"
	"fmt"

	"github.com/refraction-networking/utls/dicttls"
	"golang.org/x/crypto/cryptobyte"
)

type ProtocolVersion uint16

func (v ProtocolVersion) Hi() uint8 {
	return uint8(v >> 8)
}

func (v ProtocolVersion) Lo() uint8 {
	return uint8(v)
}

func (v ProtocolVersion) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]uint8{v.Hi(), v.Lo()})
}

type CompressionMethod uint8

func (m CompressionMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint16(m))
}

type ClientHelloData struct {
	Raw []byte `json:"raw"`

	Version            ProtocolVersion     `json:"version"`
	Radnom             []byte              `json:"random"`
	SessionID          []byte              `json:"sessionId"`
	CipherSuites       []string            `json:"ciphersuites"`
	CompressionMethods []CompressionMethod `json:"compressionMethods"`
	Extensions         []Extentison        `json:"extensions"`
}

func UnmarshallClientHello(rawBytes []byte) (*ClientHelloData, error) {

	clientHelloData := &ClientHelloData{
		Raw: rawBytes,
	}

	handshakeMessage := cryptobyte.String(rawBytes)

	var messageType uint8
	if !handshakeMessage.ReadUint8(&messageType) || messageType != 1 {
		return nil, fmt.Errorf("bytes are not corresponding to ClientHello, provided message type: %d", messageType)
	}

	var clientHello cryptobyte.String
	if !handshakeMessage.ReadUint24LengthPrefixed(&clientHello) || !handshakeMessage.Empty() {
		return nil, fmt.Errorf("prefix is not empty %v", handshakeMessage)
	}

	if !clientHello.ReadUint16((*uint16)(&clientHelloData.Version)) {
		return nil, fmt.Errorf("missing version info")
	}

	if !clientHello.ReadBytes(&clientHelloData.Radnom, 32) {
		return nil, fmt.Errorf("missing random info")
	}

	if !clientHello.ReadUint8LengthPrefixed((*cryptobyte.String)(&clientHelloData.SessionID)) {
		return nil, fmt.Errorf("missing session ID info")
	}

	var cipherSuites cryptobyte.String
	if !clientHello.ReadUint16LengthPrefixed(&cipherSuites) {
		return nil, fmt.Errorf("missing cipher suites data")
	}

	for !cipherSuites.Empty() {
		var cipherSuite uint16
		if !cipherSuites.ReadUint16(&cipherSuite) {
			return nil, fmt.Errorf("cipher suite not found")
		}
		cipherName := ParseCipherSuite(cipherSuite)
		if len(cipherName) > 1 {
			clientHelloData.CipherSuites = append(clientHelloData.CipherSuites, cipherName)
		}
	}

	var compressionMethods cryptobyte.String
	if !clientHello.ReadUint8LengthPrefixed(&compressionMethods) {
		return nil, fmt.Errorf("missing compression methods data")
	}

	clientHelloData.CompressionMethods = []CompressionMethod{}
	for !compressionMethods.Empty() {
		var compressionMethod uint8
		if !compressionMethods.ReadUint8(&compressionMethod) {
			return nil, fmt.Errorf("compression method not found")
		}
		clientHelloData.CompressionMethods = append(clientHelloData.CompressionMethods, CompressionMethod(compressionMethod))
	}

	clientHelloData.Extensions = []Extentison{}

	var extensions cryptobyte.String
	if !clientHello.ReadUint16LengthPrefixed(&extensions) {
		return nil, fmt.Errorf("missing extensions data")
	}

	for !extensions.Empty() {
		var extensionType uint16
		var extensionData cryptobyte.String
		if !extensions.ReadUint16(&extensionType) || !extensions.ReadUint16LengthPrefixed(&extensionData) {
			return nil, fmt.Errorf("extension not found")
		}

		extensionParser := extensionParsers[extensionType]
		if extensionParser == nil {
			extensionParser = ParseUnknownExtensions
		}
		extensionInfo := extensionParser(extensionData)
		clientHelloData.Extensions = append(clientHelloData.Extensions, Extentison{
			Type: extensionType,
			Name: dicttls.DictExtTypeValueIndexed[extensionType],
			Data: extensionInfo,
		})

	}

	return clientHelloData, nil
}
