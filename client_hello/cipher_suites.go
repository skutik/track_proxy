package client_hello

import (
	"fmt"

	"github.com/refraction-networking/utls/dicttls"
)

func ParseCipherSuite(cipherSuite uint16) string {
	cipherName, ok := dicttls.DictCipherSuiteValueIndexed[cipherSuite]
	if !ok {
		fmt.Println("Cipher mapping not found for value", cipherSuite)
		return ""
	}

	return cipherName
}
