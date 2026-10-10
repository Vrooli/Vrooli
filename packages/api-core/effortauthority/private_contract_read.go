package effortauthority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

func parseContractURL(raw, path string) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.Path != path || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, ErrRefused
	}
	return u, nil
}
func privateContractRead(ctx context.Context, cfg AcceptanceContractClientConfig, endpoint string, in any, out any) error {
	if cfg.HTTP == nil || cfg.HTTP.Timeout <= 0 || cfg.ServerPin == "" {
		return ErrRefused
	}
	u, e := url.Parse(cfg.URL)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.Path != endpoint || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return ErrRefused
	}
	tr, ok := cfg.HTTP.Transport.(*http.Transport)
	if !ok || tr.DialTLS != nil || tr.DialTLSContext != nil || tr.TLSClientConfig == nil || tr.TLSClientConfig.InsecureSkipVerify || len(tr.TLSClientConfig.Certificates) == 0 {
		return ErrRefused
	}
	raw, e := json.Marshal(in)
	if e != nil {
		return ErrRefused
	}
	req, e := http.NewRequestWithContext(ctx, "POST", cfg.URL, bytes.NewReader(raw))
	if e != nil {
		return ErrRefused
	}
	req.Header.Set("Content-Type", "application/json")
	client := *cfg.HTTP
	protectedTransport := tr.Clone()
	protectedTLS := tr.TLSClientConfig.Clone()
	previousVerify := protectedTLS.VerifyConnection
	protectedTLS.VerifyConnection = func(state tls.ConnectionState) error {
		if len(state.VerifiedChains) == 0 || len(state.PeerCertificates) == 0 {
			return ErrRefused
		}
		peer := sha256.Sum256(state.PeerCertificates[0].Raw)
		if hex.EncodeToString(peer[:]) != cfg.ServerPin {
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
		return ErrRefused
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.TLS == nil || len(resp.TLS.VerifiedChains) == 0 || len(resp.TLS.PeerCertificates) == 0 {
		return ErrRefused
	}
	pin := sha256.Sum256(resp.TLS.PeerCertificates[0].Raw)
	if hex.EncodeToString(pin[:]) != cfg.ServerPin {
		return ErrRefused
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(body) > 65536 {
		return ErrRefused
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(out) != nil || dec.Decode(new(any)) != io.EOF {
		return ErrRefused
	}
	return nil
}
