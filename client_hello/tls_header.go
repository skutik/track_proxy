package client_hello

type TlsHeaders struct {
	ContentType string
	Version     string
	Type        string
	LenghtBytes int
	RecordType  string
}

var ContentType = map[uint16]string{
	0x15: "ALERT",
	0x16: "HANDSHAKE",
	0x18: "HEARTBEAT",
}

var SslRecordType = map[uint16]string{

	0x14: "SSL3_RT_CHANGE_CIPHER_SPEC",
	0x15: "SSL3_RT_ALERT",
	0x16: "SSL3_RT_HANDSHAKE",
	0x17: "SSL3_RT_APPLICATION_DATA",
	0x18: "TLS1_RT_HEARTBEAT",
}

var SslVersion = map[uint16]string{
	0x0301: "TLS1_VERSION",
	0x0302: "TLS1_1_VERSION",
	0x0303: "TLS1_2_VERSION",
	0x0304: "TLS1_3_VERSION",
}
