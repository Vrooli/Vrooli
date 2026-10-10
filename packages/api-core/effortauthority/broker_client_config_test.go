package effortauthority

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"
)

func TestReadinessFrozenBrokerRejectsWrongPrivateCarrierAndMetadataMutation(t *testing.T) {
	_, _, _, p, _, _ := authorityFixture(t)
	cert, pool, pin := transportCertificate(t)
	httpClient := &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, Certificates: []tls.Certificate{cert}}}}
	cfg := BrokerClientConfig{URL: "https://fixture.invalid" + BrokerEndpoint, HTTP: httpClient, ServerPin: pin, Bindings: map[string]Binding{p.ID: binding(p)}}
	c, e := NewBrokerClient(cfg)
	if e != nil {
		t.Fatal(e)
	}
	delete(cfg.Bindings, p.ID)
	httpClient.Timeout = 0
	httpClient.Transport.(*http.Transport).TLSClientConfig.Certificates[0].Certificate[0][0] ^= 1
	c.URL = "http://wrong"
	c.ServerPin = "wrong"
	c.Bindings = map[string]Binding{}
	current := c.snapshot()
	if current.URL != cfg.URL || current.ServerPin != pin || current.Bindings[p.ID] != binding(p) || current.HTTP.Timeout != time.Second || current.HTTP.Transport.(*http.Transport).TLSClientConfig.Certificates[0].Certificate[0][0] == cert.Certificate[0][0] {
		t.Fatal("snapshot aliased")
	}
	for _, mode := range []string{"jar", "tls-bypass", "http", "wrong-path", "binding"} {
		t.Run(mode, func(t *testing.T) {
			cert, pool, pin := transportCertificate(t)
			client := &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, Certificates: []tls.Certificate{cert}}}}
			cfg := BrokerClientConfig{URL: "https://fixture.invalid" + BrokerEndpoint, HTTP: client, ServerPin: pin, Bindings: map[string]Binding{p.ID: binding(p)}}
			switch mode {
			case "jar":
				client.Jar, _ = cookiejar.New(nil)
			case "tls-bypass":
				client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify = true
			case "http":
				cfg.URL = "http://fixture.invalid" + BrokerEndpoint
			case "wrong-path":
				cfg.URL = "https://fixture.invalid/wrong"
			case "binding":
				b := binding(p)
				b.PolicyDigest = "bad"
				cfg.Bindings[p.ID] = b
			}
			if _, e := NewBrokerClient(cfg); e == nil {
				t.Fatal("invalid private startup carrier accepted")
			}
		})
	}
}
