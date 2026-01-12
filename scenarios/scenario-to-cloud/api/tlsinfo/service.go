package tlsinfo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

// Service probes live TLS certificates for edge domains.
type Service interface {
	Probe(ctx context.Context, domain string) (ProbeResult, error)
}

// ProbeResult contains TLS certificate details for a domain.
type ProbeResult struct {
	Domain          string
	Valid           bool
	Validation      string
	ValidationError string
	Issuer          string
	Subject         string
	NotBefore       string
	NotAfter        string
	DaysRemaining   int
	SerialNumber    string
	SANs            []string
}

// Config controls probe behavior.
type Config struct {
	Timeout time.Duration
	Port    int
	Verify  bool
}

// Option configures the probe service.
type Option func(*Config)

// WithTimeout sets the default probe timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}

// WithPort overrides the default TLS port (443).
func WithPort(port int) Option {
	return func(cfg *Config) {
		cfg.Port = port
	}
}

// WithVerify enables full certificate verification (chain + hostname).
func WithVerify(verify bool) Option {
	return func(cfg *Config) {
		cfg.Verify = verify
	}
}

// DefaultService probes TLS endpoints using the Go TLS stack.
type DefaultService struct {
	config Config
}

// NewService constructs a TLS probe service.
func NewService(opts ...Option) *DefaultService {
	cfg := Config{Timeout: 10 * time.Second}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &DefaultService{config: cfg}
}

// Probe retrieves TLS certificate details from the given domain.
func (s *DefaultService) Probe(ctx context.Context, domain string) (ProbeResult, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ProbeResult{}, fmt.Errorf("domain is empty")
	}
	ctx, cancel := s.applyTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}

	port := 443
	if s.config.Port > 0 {
		port = s.config.Port
	}
	addr := net.JoinHostPort(domain, fmt.Sprintf("%d", port))
	dialer := &net.Dialer{Timeout: s.config.Timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return ProbeResult{}, err
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return ProbeResult{}, fmt.Errorf("no peer certificates returned")
	}
	validation := "time_only"
	var validationErr error
	now := time.Now()
	if s.config.Verify {
		validation = "full"
		validationErr = verifyCertificateChain(domain, state.PeerCertificates, now)
	}
	return buildProbeResult(domain, state.PeerCertificates[0], now, validation, validationErr), nil
}

func (s *DefaultService) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.config.Timeout <= 0 {
		return ctx, nil
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, nil
	}
	return context.WithTimeout(ctx, s.config.Timeout)
}

func buildProbeResult(domain string, cert *x509.Certificate, now time.Time, validation string, validationErr error) ProbeResult {
	if cert == nil {
		return ProbeResult{Domain: domain, Valid: false}
	}
	if strings.TrimSpace(validation) == "" {
		validation = "time_only"
	}

	notAfter := cert.NotAfter.UTC()
	notBefore := cert.NotBefore.UTC()
	validTime := now.After(notBefore) && now.Before(notAfter)
	valid := validTime
	if validation == "full" {
		if validationErr != nil {
			valid = false
		} else {
			valid = validTime
		}
	}
	days := int(notAfter.Sub(now).Hours() / 24)
	if days < 0 {
		days = 0
	}

	return ProbeResult{
		Domain:          domain,
		Valid:           valid,
		Validation:      validation,
		ValidationError: validationErrorString(validationErr),
		Issuer:          cert.Issuer.CommonName,
		Subject:         cert.Subject.String(),
		NotBefore:       formatCertTime(notBefore),
		NotAfter:        formatCertTime(notAfter),
		DaysRemaining:   days,
		SerialNumber:    formatSerial(cert.SerialNumber),
		SANs:            append([]string{}, cert.DNSNames...),
	}
}

func verifyCertificateChain(domain string, certs []*x509.Certificate, now time.Time) error {
	if len(certs) == 0 {
		return fmt.Errorf("no certificates provided")
	}
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	intermediates := x509.NewCertPool()
	for i := 1; i < len(certs); i++ {
		intermediates.AddCert(certs[i])
	}
	opts := x509.VerifyOptions{
		DNSName:       domain,
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   now,
	}
	_, err = certs[0].Verify(opts)
	return err
}

func validationErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func formatCertTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("Jan 2 15:04:05 2006 MST")
}

func formatSerial(serial *big.Int) string {
	if serial == nil {
		return ""
	}
	return serial.Text(16)
}
