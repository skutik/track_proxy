package client_hello

import (
	"golang.org/x/crypto/cryptobyte"
)

type ExtensionData interface{}

type ServerNameExtension struct {
	Raw        []byte `json:"raw"`
	Valid      bool   `json:"valid"`
	ServerName string `json:"serverName"`
}

type Extentison struct {
	Type    uint16        `json:"type"`
	Name    string        `json:"name"`
	Grease  bool          `json:"grease"`
	Private bool          `json:"private"`
	Data    ExtensionData `json:"data"`
}

func ParseServerNameExtension(rawData []byte) ExtensionData {
	serverNameExtension := &ServerNameExtension{Raw: rawData}
	extensionData := cryptobyte.String(rawData)
	var nameList cryptobyte.String
	if !extensionData.ReadUint16LengthPrefixed(&nameList) || nameList.Empty() {
		return serverNameExtension
	}

	for !nameList.Empty() {
		var nameType uint8
		if !nameList.ReadUint8(&nameType) {
			return serverNameExtension
		}
		var nameData cryptobyte.String
		if !nameList.ReadUint16LengthPrefixed(&nameData) || nameData.Empty() {
			return serverNameExtension
		}

		switch nameType {
		case 0:
			if serverNameExtension.ServerName != "" {
				return serverNameExtension
			}
			serverNameExtension.ServerName = string(nameData)
		}
	}
	if !extensionData.Empty() {
		return serverNameExtension
	}
	serverNameExtension.Valid = true
	return serverNameExtension

}

type SupportedGroupsExtension struct {
	Raw    []byte   `json:"raw"`
	Valid  bool     `json:"valid"`
	Groups []uint16 `json:"groups"`
}

func ParseSupportedGroupsExtension(rawData []byte) ExtensionData {
	supportedGroupsExtension := &SupportedGroupsExtension{Raw: rawData, Groups: []uint16{}}
	extensionData := cryptobyte.String(rawData)
	var groupList cryptobyte.String
	if !extensionData.ReadUint16LengthPrefixed(&groupList) || groupList.Empty() {
		return supportedGroupsExtension
	}
	for !groupList.Empty() {
		var groupCode uint16
		if !groupList.ReadUint16(&groupCode) {
			return supportedGroupsExtension
		}

		supportedGroupsExtension.Groups = append(supportedGroupsExtension.Groups, groupCode)
	}
	if !extensionData.Empty() {
		return supportedGroupsExtension
	}
	supportedGroupsExtension.Valid = true
	return supportedGroupsExtension
}

type EcPointFormatsExtension struct {
	Raw     []byte   `json:"raw"`
	Valid   bool     `json:"valid"`
	Formats []uint16 `json:"formats"`
}

func ParseEcPointFormatExtenstion(rawData []byte) ExtensionData {
	ecPointFormatsExtension := &EcPointFormatsExtension{Raw: rawData, Formats: []uint16{}}
	extensionData := cryptobyte.String(rawData)
	var formatList cryptobyte.String
	if !extensionData.ReadUint8LengthPrefixed(&formatList) || formatList.Empty() {
		return ecPointFormatsExtension
	}
	for !formatList.Empty() {
		var formatCode uint8
		if !formatList.ReadUint8(&formatCode) {
			return ecPointFormatsExtension
		}

		ecPointFormatsExtension.Formats = append(ecPointFormatsExtension.Formats, uint16(formatCode))
	}
	if !extensionData.Empty() {
		return ecPointFormatsExtension
	}
	ecPointFormatsExtension.Valid = true
	return ecPointFormatsExtension
}

type AlpnExtension struct {
	Raw       []byte   `json:"raw"`
	Valid     bool     `json:"valid"`
	Protocols []string `json:"protocols"`
}

func ParseAlpnExtension(rawData []byte) ExtensionData {
	alpnData := &AlpnExtension{Raw: rawData, Protocols: []string{}}
	extensionData := cryptobyte.String(rawData)
	var protocolList cryptobyte.String
	if !extensionData.ReadUint16LengthPrefixed(&protocolList) || protocolList.Empty() {
		return alpnData
	}
	for !protocolList.Empty() {
		var protocolName cryptobyte.String
		if !protocolList.ReadUint8LengthPrefixed(&protocolName) || protocolName.Empty() {
			return alpnData
		}

		alpnData.Protocols = append(alpnData.Protocols, string(protocolName))
	}
	if !extensionData.Empty() {
		return alpnData
	}
	alpnData.Valid = true
	return alpnData
}

type UnknownExtensionData struct {
	Raw []byte `json:"raw"`
}

type EmptyExtensionData struct {
	Raw   []byte `json:"raw"`
	Valid bool   `json:"valid"`
}

func ParseEmptyExtension(rawData []byte) ExtensionData {
	return &EmptyExtensionData{
		Raw:   rawData,
		Valid: len(rawData) == 0,
	}
}

func ParseUnknownExtensions(rawData []byte) ExtensionData {
	return &EmptyExtensionData{
		Raw: rawData,
	}
}

var extensionParsers = map[uint16]func([]byte) ExtensionData{
	0:  ParseServerNameExtension,
	10: ParseSupportedGroupsExtension,
	11: ParseEcPointFormatExtenstion,
	16: ParseAlpnExtension,
	18: ParseEmptyExtension,
	22: ParseEmptyExtension,
	23: ParseEmptyExtension,
	49: ParseEmptyExtension,
}
