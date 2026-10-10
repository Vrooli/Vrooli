package effortauthority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// ProfileContractClient consumes a supplied existing verified TLS handle;
// it performs no enrollment, refresh or state mutation.
type ProfileContractClientConfig struct {
	URL       string
	HTTP      *http.Client
	ServerPin string
	Binding   Binding
	Profiles  map[string]string
}

type ProfileContractClient struct{ config ProfileContractClientConfig }

func NewProfileContractClient(cfg ProfileContractClientConfig) (*ProfileContractClient, error) {
	if cfg.HTTP == nil || cfg.HTTP.Jar != nil || cfg.HTTP.Timeout <= 0 || !validDigest(cfg.ServerPin) || cfg.Binding.PolicyID == "" || cfg.Binding.Epoch == 0 || cfg.Binding.Owner == "" || cfg.Binding.Client == "" || !validDigest(cfg.Binding.PolicyDigest) || cfg.Binding.Deadline.IsZero() || len(cfg.Profiles) == 0 {
		return nil, ErrRefused
	}
	tr, ok := cfg.HTTP.Transport.(*http.Transport)
	if !ok || tr.DialTLS != nil || tr.DialTLSContext != nil || tr.TLSClientConfig == nil || tr.TLSClientConfig.InsecureSkipVerify || len(tr.TLSClientConfig.Certificates) == 0 {
		return nil, ErrRefused
	}
	u, e := url.Parse(cfg.URL)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.Path != ProfileContractEndpoint || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, ErrRefused
	}
	profiles := map[string]string{}
	for key, id := range cfg.Profiles {
		if key == "" || id == "" {
			return nil, ErrRefused
		}
		profiles[key] = id
	}
	cfg.Profiles = profiles
	client := *cfg.HTTP
	transport := tr.Clone()
	transport.TLSClientConfig = tr.TLSClientConfig.Clone()
	if transport.TLSClientConfig.MaxVersion != 0 && transport.TLSClientConfig.MaxVersion < tls.VersionTLS13 {
		return nil, ErrRefused
	}
	transport.TLSClientConfig.MinVersion = tls.VersionTLS13
	transport.TLSClientConfig.Certificates = make([]tls.Certificate, len(tr.TLSClientConfig.Certificates))
	for i, offered := range tr.TLSClientConfig.Certificates {
		cert := offered // private signer handle remains owner supplied.
		cert.Certificate = make([][]byte, len(offered.Certificate))
		for j, der := range offered.Certificate {
			cert.Certificate[j] = append([]byte{}, der...)
		}
		cert.OCSPStaple = append([]byte{}, offered.OCSPStaple...)
		cert.SignedCertificateTimestamps = make([][]byte, len(offered.SignedCertificateTimestamps))
		for j, stamp := range offered.SignedCertificateTimestamps {
			cert.SignedCertificateTimestamps[j] = append([]byte{}, stamp...)
		}
		if offered.Leaf != nil {
			leaf, err := x509.ParseCertificate(offered.Leaf.Raw)
			if err != nil {
				return nil, ErrRefused
			}
			cert.Leaf = leaf
		}
		transport.TLSClientConfig.Certificates[i] = cert
	}
	if tr.TLSClientConfig.RootCAs != nil {
		transport.TLSClientConfig.RootCAs = tr.TLSClientConfig.RootCAs.Clone()
	}
	if tr.TLSClientConfig.ClientCAs != nil {
		transport.TLSClientConfig.ClientCAs = tr.TLSClientConfig.ClientCAs.Clone()
	}
	client.Transport = transport
	cfg.HTTP = &client
	return &ProfileContractClient{config: cfg}, nil
}

func (c *ProfileContractClient) read(ctx context.Context, in profileCheckRequest) (ProfileContract, error) {
	var out ProfileContract
	if c == nil || c.config.HTTP == nil || c.config.HTTP.Timeout <= 0 || c.config.ServerPin == "" {
		return out, ErrRefused
	}
	u, e := url.Parse(c.config.URL)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.Path != ProfileContractEndpoint || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return out, ErrRefused
	}
	tr, ok := c.config.HTTP.Transport.(*http.Transport)
	if !ok || tr.DialTLS != nil || tr.DialTLSContext != nil || tr.TLSClientConfig == nil || tr.TLSClientConfig.InsecureSkipVerify || len(tr.TLSClientConfig.Certificates) == 0 {
		return out, ErrRefused
	}
	raw, e := json.Marshal(in)
	if e != nil {
		return out, ErrRefused
	}
	req, e := http.NewRequestWithContext(ctx, "POST", c.config.URL, bytes.NewReader(raw))
	if e != nil {
		return out, ErrRefused
	}
	req.Header.Set("Content-Type", "application/json")
	client := *c.config.HTTP
	protectedTransport := tr.Clone()
	protectedTLS := tr.TLSClientConfig.Clone()
	previousVerify := protectedTLS.VerifyConnection
	protectedTLS.VerifyConnection = func(state tls.ConnectionState) error {
		if len(state.VerifiedChains) == 0 || len(state.PeerCertificates) == 0 {
			return ErrRefused
		}
		peer := sha256.Sum256(state.PeerCertificates[0].Raw)
		if hex.EncodeToString(peer[:]) != c.config.ServerPin {
			return ErrRefused
		}
		if previousVerify != nil {
			return previousVerify(state)
		}
		return nil
	}
	protectedTransport.TLSClientConfig = protectedTLS
	client.Transport = protectedTransport
	defer protectedTransport.CloseIdleConnections()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, e := client.Do(req)
	if e != nil {
		return out, ErrRefused
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.TLS == nil || len(resp.TLS.VerifiedChains) == 0 || len(resp.TLS.PeerCertificates) == 0 {
		return out, ErrRefused
	}
	pin := sha256.Sum256(resp.TLS.PeerCertificates[0].Raw)
	if hex.EncodeToString(pin[:]) != c.config.ServerPin {
		return out, ErrRefused
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(body) > 65536 {
		return out, ErrRefused
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(&out) != nil || dec.Decode(new(any)) != io.EOF {
		return ProfileContract{}, ErrRefused
	}
	return out, nil
}
func (c *ProfileContractClient) CheckNativeProfile(ctx context.Context, key, digest string) error {
	if c == nil || !validDigest(digest) || c.config.Profiles[key] == "" {
		return ErrRefused
	}
	out, e := c.read(ctx, profileCheckRequest{c.config.Binding, key})
	if e != nil || out.ID != c.config.Profiles[key] || out.Key != key || out.Digest != digest {
		return ErrRefused
	}
	return nil
}

func (c *ProfileContractClient) CheckInstallation(p Policy) error {
	if c == nil || !sameProfileBinding(c.config.Binding, binding(p)) || len(c.config.Profiles) != len(p.Profiles) {
		return ErrRefused
	}
	for key := range c.config.Profiles {
		if p.Profiles[key] == "" {
			return ErrRefused
		}
	}
	return nil
}
