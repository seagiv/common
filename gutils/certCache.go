package gutils

import (
	"crypto/x509"
	"fmt"
	"sync"
	"time"
)

type certItem struct {
	cert       *x509.Certificate
	expireTime time.Time
}

//CertCache interface to certificate cache
type CertCache struct {
	sync.Mutex
	duration time.Duration
	poolRoot *x509.CertPool
	data     map[string]certItem
}

var certCache *CertCache

//InitCertCache create new certificate cache
func InitCertCache(duration time.Duration, poolRoot *x509.CertPool) {
	certCache = &CertCache{
		duration: duration,
		poolRoot: poolRoot,
		data:     make(map[string]certItem),
	}
}

//Get -
func (s *CertCache) Get(keyid []byte) *x509.Certificate {
	s.Lock()
	item := s.data[fmt.Sprintf("%X", keyid)]
	defer s.Unlock()

	if item.cert != nil {
		if time.Now().After(item.expireTime) {
			return nil // expired
		}
	}

	return item.cert
}

//Set -
func (s *CertCache) Set(cert *x509.Certificate) {
	keyid := fmt.Sprintf("%X", cert.SubjectKeyId)

	s.Lock()
	s.data[keyid] = certItem{cert: cert, expireTime: time.Now().Add(s.duration)}
	defer s.Unlock()
}

func findCertificate(c *ClientConnection, keyID []byte, checkDate time.Time) (*x509.Certificate, error) {
	var err error
	var cert *x509.Certificate

	cert = certCache.Get(keyID)

	if cert != nil {
		return cert, nil
	}

	//getting certificate
	var replyGC []byte
	err = c.Call("Gateway.GetCertificateByKeyID", keyID, &replyGC)
	if err != nil {
		return nil, FormatErrorS("Gateway.GetCertificateByKeyID", "[%X] %v", keyID, err)
	}

	//parsing certificate
	cert, err = x509.ParseCertificate(replyGC)
	if err != nil {
		return nil, FormatErrorS("ParseCertificate", "%v", err)
	}

	//checking Cert signature, issuer, date etc
	var optsVerify x509.VerifyOptions

	optsVerify.Roots = certCache.poolRoot

	//RELEASE: old transactions before certificate NotBefore
	optsVerify.CurrentTime = cert.NotBefore.Add(time.Second)

	_, err = cert.Verify(optsVerify)
	if err != nil {
		return nil, FormatErrorS("Verify", "%v", err)
	}

	certCache.Set(cert)

	return cert, nil
}

//GetCertificate retrive and validate cert and its CommonName
func GetCertificate(c *ClientConnection, keyID []byte, allowedCNs []string, checkDate time.Time) (*x509.Certificate, error) {
	//getting certificate from cache or from db and verify it
	cert, err := findCertificate(c, keyID, checkDate)
	if err != nil {
		return nil, err
	}

	if !IsIn(cert.Subject.CommonName, allowedCNs) {
		return nil, FormatErrorS("checkCommonName", "CommonName [%s] not allowed, must be one of %v", cert.Subject.CommonName, allowedCNs)
	}

	if !((checkDate.After(cert.NotBefore) || checkDate.Equal(cert.NotBefore)) && checkDate.Before(cert.NotAfter)) {
		return nil, FormatErrorS("checkDate", "checkDate (%v) out of certificate valid interval", checkDate)
	}

	return cert, nil
}
