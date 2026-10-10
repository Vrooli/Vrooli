package effortauthority

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Certificate bytes are deliberately synthetic parser data, never credentials.
func TestReadinessProfileClientFreezesOfferedTLSAndProfileTables(t *testing.T) {
	_, _, binding, _ := profileHandlerFixture(t)
	offeredTLS := &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{[]byte("synthetic-certificate-bytes")}}}}
	tr := &http.Transport{TLSClientConfig: offeredTLS}
	offeredHTTP := &http.Client{Transport: tr, Timeout: time.Second}
	cfg := ProfileContractClientConfig{URL: "https://owner.example" + ProfileContractEndpoint, HTTP: offeredHTTP, ServerPin: Digest("synthetic-server-pin"), Binding: binding, Profiles: map[string]string{"bounded": "owner-profile-id"}}
	c, e := NewProfileContractClient(cfg)
	if e != nil {
		t.Fatal(e)
	}
	cfg.Profiles["bounded"] = "foreign"
	offeredHTTP.Timeout = 0
	offeredTLS.InsecureSkipVerify = true
	offeredTLS.Certificates[0].Certificate[0][0] = 'X'
	frozen := c.config.HTTP.Transport.(*http.Transport).TLSClientConfig
	if c.config.Profiles["bounded"] != "owner-profile-id" || c.config.HTTP.Timeout != time.Second || frozen.InsecureSkipVerify {
		t.Fatal("offered mutable configuration changed frozen client")
	}
	if string(frozen.Certificates[0].Certificate[0]) != "synthetic-certificate-bytes" {
		t.Fatal("frozen TLS certificate bytes alias offered configuration")
	}
}

// This new client contract fixture performs a genuine in-process TLS handshake
// over net.Pipe. It does not listen or connect on a network interface, run the
// older joined helper, enroll peers, or qualify host/provider networking.
func TestReadinessProfileClientVerifiedTLSContractWithoutListener(t *testing.T) {
	cert, pool, pin := transportCertificate(t)
	p := readinessInstallationFixture().Policy
	spy := &profileReadSpy{out: ProfileContract{ID: "profile-id", Key: "bounded", Digest: p.Profiles["bounded"]}}
	h, e := NewProfileContractHandler(spy, map[string]map[string]ProfileContractGrant{pin: {p.ID: {Binding: binding(p), Profiles: map[string]string{"bounded": "profile-id"}}}})
	if e != nil {
		t.Fatal(e)
	}
	serverTLS := &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			conn := tls.Server(server, serverTLS)
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
			if conn.HandshakeContext(ctx) != nil {
				return
			}
			request, err := http.ReadRequest(bufio.NewReader(conn))
			if err != nil {
				return
			}
			defer request.Body.Close()
			state := conn.ConnectionState()
			request.TLS = &state
			result := httptest.NewRecorder()
			h.ServeHTTP(result, request)
			_ = result.Result().Write(conn)
		}()
		return client, nil
	}
	cfg := ProfileContractClientConfig{URL: "https://127.0.0.1" + ProfileContractEndpoint, ServerPin: pin, Binding: binding(p), Profiles: map[string]string{"bounded": "profile-id"}, HTTP: &http.Client{Timeout: time.Second, Transport: &http.Transport{DialContext: dial, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{cert}}}}}
	c, e := NewProfileContractClient(cfg)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.CheckNativeProfile(context.Background(), "bounded", p.Profiles["bounded"]); e != nil || spy.calls != 1 {
		t.Fatal("verified transport refused", e, spy.calls)
	}
	spy.out.Digest = Digest("owner drift")
	if c.CheckNativeProfile(context.Background(), "bounded", p.Profiles["bounded"]) == nil || spy.calls != 2 {
		t.Fatal("owner drift accepted")
	}
	cfg.ServerPin = Digest("wrong server")
	wrong, e := NewProfileContractClient(cfg)
	if e != nil {
		t.Fatal(e)
	}
	if wrong.CheckNativeProfile(context.Background(), "bounded", p.Profiles["bounded"]) == nil || spy.calls != 2 {
		t.Fatal("wrong server reached owner")
	}
}
