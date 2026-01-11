package tlsinfo

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestProbeReturnsCertificateDetails(t *testing.T) {
	cert, err := generateSelfSignedCert("127.0.0.1")
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				if tlsConn, ok := c.(*tls.Conn); ok {
					_ = tlsConn.Handshake()
				}
			}(conn)
		}
	}()

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse listener addr: %v", err)
	}
	port, err := parsePort(portStr)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(WithPort(port), WithTimeout(2*time.Second))
	result, err := svc.Probe(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	if result.Domain != "127.0.0.1" {
		t.Fatalf("expected domain 127.0.0.1, got %q", result.Domain)
	}
	if result.Issuer == "" || result.Subject == "" {
		t.Fatalf("expected issuer and subject, got issuer=%q subject=%q", result.Issuer, result.Subject)
	}
	if result.NotAfter == "" || result.NotBefore == "" {
		t.Fatalf("expected not_before and not_after timestamps")
	}
}

func generateSelfSignedCert(ip string) (tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	now := time.Now().Add(-time.Hour)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "Scenario-to-Cloud Test Cert",
		},
		NotBefore:             now,
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		template.IPAddresses = []net.IP{parsedIP}
	} else {
		template.DNSNames = []string{ip}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	return tls.X509KeyPair(certPEM, keyPEM)
}

func parsePort(value string) (int, error) {
	return strconv.Atoi(value)
}
