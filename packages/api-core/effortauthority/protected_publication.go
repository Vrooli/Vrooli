package effortauthority

import (
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	lifecycle "github.com/vrooli/api-core/server"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ProtectedPublicationConfig is owner-supplied existing transport custody and
// a nonsecret installation descriptor. It creates no identity, trust or grant.
// The role's handler still verifies its exact policy and enrolled peer pin.
type ProtectedPublicationConfig struct {
	Role         string
	Address      string
	Installation Installation
	TLS          *tls.Config
}

// ProtectedPublicationPlan is immutable; preparation opens no listener.
type ProtectedPublicationPlan struct {
	role, address      string
	config             *tls.Config
	installationDigest string
	deadline           time.Time
}

func PrepareProtectedPublication(c ProtectedPublicationConfig) (*ProtectedPublicationPlan, error) {
	switch c.Role {
	case "am", "pm", "issuer", "acceptance":
	default:
		return nil, ErrRefused
	}
	descriptor := c.Installation
	descriptor.Enabled = true
	frozen, e := descriptor.Freeze()
	if e != nil {
		return nil, e
	}
	host, port, e := net.SplitHostPort(c.Address)
	if e != nil || net.ParseIP(host) == nil {
		return nil, ErrRefused
	}
	n, e := strconv.Atoi(port)
	if e != nil || n < 1024 || n > 65535 {
		return nil, ErrRefused
	}
	offered := c.TLS
	if offered == nil || offered.ClientCAs == nil || len(offered.Certificates) != 1 || offered.ClientAuth != tls.RequireAndVerifyClientCert || offered.InsecureSkipVerify || offered.GetCertificate != nil || offered.GetConfigForClient != nil || offered.GetClientCertificate != nil || offered.VerifyPeerCertificate != nil || offered.VerifyConnection != nil || (offered.MaxVersion != 0 && offered.MaxVersion < tls.VersionTLS13) {
		return nil, ErrRefused
	}
	cert := offered.Certificates[0]
	if len(cert.Certificate) == 0 || cert.PrivateKey == nil {
		return nil, ErrRefused
	}
	leaf, e := x509.ParseCertificate(cert.Certificate[0])
	if e != nil || time.Now().Before(leaf.NotBefore) || !time.Now().Before(leaf.NotAfter) {
		return nil, ErrRefused
	}
	key, ok := cert.PrivateKey.(crypto.Signer)
	if !ok {
		return nil, ErrRefused
	}
	actual, e := x509.MarshalPKIXPublicKey(key.Public())
	if e != nil {
		return nil, ErrRefused
	}
	expected, e := x509.MarshalPKIXPublicKey(leaf.PublicKey)
	if e != nil || string(actual) != string(expected) {
		return nil, ErrRefused
	}
	pin := sha256.Sum256(cert.Certificate[0])
	role := c.Role
	if role == "acceptance" && !validDigest(frozen.Builds[role]) {
		return nil, ErrRefused
	}
	if hex.EncodeToString(pin[:]) != frozen.ServerPins[role] {
		return nil, ErrRefused
	}
	copy := offered.Clone()
	copy.MinVersion = tls.VersionTLS13
	copy.ClientAuth = tls.RequireAndVerifyClientCert
	copy.ClientCAs = offered.ClientCAs.Clone()
	copy.RootCAs = nil
	cert.Certificate = make([][]byte, len(offered.Certificates[0].Certificate))
	for i, b := range offered.Certificates[0].Certificate {
		cert.Certificate[i] = append([]byte(nil), b...)
	}
	cert.OCSPStaple = append([]byte(nil), cert.OCSPStaple...)
	cert.SignedCertificateTimestamps = make([][]byte, len(offered.Certificates[0].SignedCertificateTimestamps))
	for i, b := range offered.Certificates[0].SignedCertificateTimestamps {
		cert.SignedCertificateTimestamps[i] = append([]byte(nil), b...)
	}
	cert.Leaf = nil
	copy.Certificates = []tls.Certificate{cert}
	return &ProtectedPublicationPlan{role: c.Role, address: c.Address, config: copy, installationDigest: Digest(frozen), deadline: frozen.Policy.Deadline}, nil
}
func (p *ProtectedPublicationPlan) Role() string {
	if p == nil {
		return ""
	}
	return p.role
}

// Start is the explicit owner startup effect. Callers pass only their prepared
// role mux; it is never the public sibling API handler. No anonymous fallback.
func (p *ProtectedPublicationPlan) Start(h http.Handler) (*ProtectedPublication, error) {
	if p == nil || h == nil || !time.Now().Before(p.deadline) {
		return nil, ErrRefused
	}
	listener, e := net.Listen("tcp", p.address)
	if e != nil {
		return nil, e
	}
	server := &http.Server{Handler: h, TLSConfig: p.config.Clone(), ReadTimeout: 5 * time.Second, ReadHeaderTimeout: 3 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	running := &ProtectedPublication{server: server, listener: listener, done: make(chan error, 1)}
	go func() {
		e := server.Serve(tls.NewListener(listener, server.TLSConfig))
		if errors.Is(e, http.ErrServerClosed) {
			e = nil
		}
		running.done <- e
		close(running.done)
	}()
	return running, nil
}

type ProtectedPublication struct {
	server   *http.Server
	listener net.Listener
	done     chan error
	once     sync.Once
}

func (p *ProtectedPublication) Done() <-chan error {
	if p == nil {
		return nil
	}
	return p.done
}
func (p *ProtectedPublication) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	var e error
	p.once.Do(func() {
		e = p.server.Shutdown(ctx)
		if e != nil {
			_ = p.listener.Close()
		}
	})
	return e
}

// CheckInstallation compares the complete frozen nonsecret owner descriptor;
// changing read/live phase never changes policy, build or transport binding.
func (p *ProtectedPublicationPlan) CheckInstallation(c Installation) error {
	c.Enabled = true
	f, e := c.Freeze()
	if p == nil || e != nil || Digest(f) != p.installationDigest {
		return ErrRefused
	}
	return nil
}

// RunPublic couples the already-started protected endpoint to the existing
// public service lifecycle. Losing either listener stops both publications;
// it never cancels work, signals a native unit or revokes an approval.
func (p *ProtectedPublication) RunPublic(cfg lifecycle.Config) error {
	if p != nil {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = p.Close(ctx)
		}()
	}

	if p == nil || cfg.Handler == nil || cfg.StartServer != nil || cfg.ShutdownServer != nil {
		return ErrRefused
	}
	public := &http.Server{Handler: cfg.Handler, ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout, IdleTimeout: cfg.IdleTimeout}
	if public.ReadTimeout == 0 {
		public.ReadTimeout = 30 * time.Second
	}
	if public.WriteTimeout == 0 {
		public.WriteTimeout = 30 * time.Second
	}
	if public.IdleTimeout == 0 {
		public.IdleTimeout = 120 * time.Second
	}
	cfg.StartServer = func(addr string) error {
		listener, e := net.Listen("tcp", addr)
		if e != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = p.Close(ctx)
			return e
		}
		publicDone := make(chan error, 1)
		go func() { publicDone <- public.Serve(listener) }()
		select {
		case e := <-publicDone:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = p.Close(ctx)
			return e
		case e := <-p.done:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = public.Shutdown(ctx)
			if e == nil {
				return ErrRefused
			}
			return e
		}
	}
	cfg.ShutdownServer = func(ctx context.Context) error { return errors.Join(public.Shutdown(ctx), p.Close(ctx)) }
	return lifecycle.Run(cfg)
}
