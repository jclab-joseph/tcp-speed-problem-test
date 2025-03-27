package certutil

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
)

func GetSpkiHashFromCertDer(certBytes []byte) (string, error) {
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return "", err
	}
	return GetSpkiHash(cert.PublicKey)
}

func GetSpkiHash(publicKey crypto.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write(publicKeyBytes)
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
