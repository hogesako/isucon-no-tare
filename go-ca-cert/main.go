package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"
)

func main() {
	privateCaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	publicCaKey := privateCaKey.Public()

	subjectCa := pkix.Name{
		CommonName:         "ikasako CA",
		OrganizationalUnit: []string{"Ikasako Org Unit"},
		Organization:       []string{"Ikasako Org"},
		Country:            []string{"JP"},
	}

	caTpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subjectCa,
		NotAfter:              time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		NotBefore:             time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertificate, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, publicCaKey, privateCaKey)

	var f *os.File
	f, err = os.Create("server.crt")
	if err != nil {
		log.Fatalf("ERROR:%v\n", err)
	}
	err = pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: caCertificate})
	if err != nil {
		log.Fatalf("ERROR:%v\n", err)
	}
	err = f.Close()
	if err != nil {
		log.Fatalf("ERROR:%v\n", err)
	}

	f, err = os.Create("server.key")
	if err != nil {
		log.Fatalf("ERROR:%v\n", err)
	}

	derCaPrivateKey := x509.MarshalPKCS1PrivateKey(privateCaKey)

	err = pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: derCaPrivateKey})
	if err != nil {
		log.Fatalf("ERROR:%v\n", err)
	}
	err = f.Close()
	if err != nil {
		log.Fatalf("ERROR:%v\n", err)
	}
}
