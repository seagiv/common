package gutils

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

//InfoTLS -TODO-
var InfoTLS struct {
	ImCA *x509.Certificate
	Pair tls.Certificate
}

// LoadPublicKey -TODO-
func LoadPublicKey(fileName string) (*rsa.PublicKey, error) {
	publicPEM, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read PEM file [%s]", fileName)
	}

	blockPEM, _ := pem.Decode(publicPEM)
	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(blockPEM.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)

	return rsaPublicKey, nil
}

// LoadPrivateKeyFromPEM -TODO-
func LoadPrivateKeyFromPEM(PEM []byte) (*rsa.PrivateKey, error) {
	blockPEM, _ := pem.Decode([]byte(PEM))
	if blockPEM == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(blockPEM.Bytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// LoadPrivateKeyFromFile -TODO-
func LoadPrivateKeyFromFile(fileName string) (*rsa.PrivateKey, error) {
	PEM, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read PEM file [%s]", fileName)
	}

	return LoadPrivateKeyFromPEM(PEM)
}

// LoadPrivateKeyFromStorage -TODO-
func LoadPrivateKeyFromStorage(s *Storage, name string) (*rsa.PrivateKey, error) {
	PEM, ok := s.Get(name)

	if !ok {
		return nil, fmt.Errorf("%s not found in storage", name)
	}

	return LoadPrivateKeyFromPEM(PEM)
}

//LoadCertificateFromPEM -TODO-
func LoadCertificateFromPEM(PEM []byte) (*x509.Certificate, error) {
	var err error

	blockPEM, _ := pem.Decode(PEM)
	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(blockPEM.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

//LoadCertificateFromFile -TODO-
func LoadCertificateFromFile(fileName string) (*x509.Certificate, error) {
	PEM, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read PEM file [%s]", fileName)
	}

	return LoadCertificateFromPEM(PEM)
}

//LoadCertificateFromStorage -TODO-
func LoadCertificateFromStorage(s *Storage, name string) (*x509.Certificate, error) {
	PEM, ok := s.Get(name)

	if !ok {
		return nil, fmt.Errorf("%s not found in storage", name)
	}

	return LoadCertificateFromPEM(PEM)
}

//LoadX509KeyPairFromFile -TODO-
func LoadX509KeyPairFromFile(fnCert, fnKey string) (tls.Certificate, error) {
	var pairEmpty tls.Certificate

	certPEM, err := ioutil.ReadFile(fnCert)
	if err != nil {
		return pairEmpty, fmt.Errorf("failed to read PEM file [%s]", fnCert)
	}

	keyPEM, err := ioutil.ReadFile(fnKey)
	if err != nil {
		return pairEmpty, fmt.Errorf("failed to read PEM file [%s]", fnKey)
	}

	return tls.X509KeyPair(certPEM, keyPEM)
}

//LoadX509KeyPairFromStorage -TODO-
func LoadX509KeyPairFromStorage(s *Storage, certName, keyName string) (tls.Certificate, error) {
	certPEM, ok := s.Get(certName)

	if !ok {
		return tls.Certificate{}, fmt.Errorf("%s not found in storage", certName)
	}

	keyPEM, ok := s.Get(keyName)

	if !ok {
		return tls.Certificate{}, fmt.Errorf("%s not found in storage", keyName)
	}

	return tls.X509KeyPair(certPEM, keyPEM)
}

//InitTLS -TODO-
func InitTLS(certCA *x509.Certificate, certTLS tls.Certificate, serverName string) (*tls.Config, error) {
	certPool := x509.NewCertPool()
	//certPool.AppendCertsFromPEM(certCA)

	certPool.AddCert(certCA)

	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  certPool,
		RootCAs:    certPool,
		ServerName: serverName,
		//KeyLogWriter: os.Stdout,
	}

	tlsConfig.BuildNameToCertificate()

	tlsConfig.Certificates = make([]tls.Certificate, 1)
	tlsConfig.Certificates[0] = certTLS

	return tlsConfig, nil
}

//GetHash -TODO-
func GetHash(message []byte, hash crypto.Hash) []byte {
	pssh := hash.New()

	pssh.Write(message)

	return pssh.Sum(nil)
}

//GetSignature -TODO-
func GetSignature(message []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	var opts rsa.PSSOptions

	opts.SaltLength = rsa.PSSSaltLengthAuto

	return rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, GetHash(message, crypto.SHA256), &opts)
}

//VerifySignature --TODO--
func VerifySignature(message, signature []byte, publicKey *rsa.PublicKey) error {
	var opts rsa.PSSOptions

	opts.SaltLength = rsa.PSSSaltLengthAuto

	return rsa.VerifyPSS(publicKey, crypto.SHA256, GetHash(message, crypto.SHA256), signature, &opts)
}

//GetRandomBuffer -TODO-
func GetRandomBuffer(n int) ([]byte, error) {
	buffer := make([]byte, n)

	_, err := io.ReadFull(rand.Reader, buffer)

	return buffer, err
}

//EncryptGCM -TODO-
func EncryptGCM(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

//DecryptGCM -TODO-
func DecryptGCM(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
