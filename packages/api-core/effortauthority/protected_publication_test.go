package effortauthority

import (
	"crypto/tls"
	"testing"
)

func TestProtectedPublicationIncompleteRefusesWithoutListener(t *testing.T) {
	for _, role := range []string{"", "unknown", "am", "pm", "issuer", "acceptance"} {
		plan, err := PrepareProtectedPublication(ProtectedPublicationConfig{Role: role, Address: "127.0.0.1:18443"})
		if err == nil || plan != nil {
			t.Fatalf("incomplete role %q published plan", role)
		}
	}
	var plan *ProtectedPublicationPlan
	if publication, err := plan.Start(nil); err == nil || publication != nil {
		t.Fatal("nil plan published listener")
	}
}

func TestProtectedPublicationFreezesVerifiedRoleWithoutListening(t *testing.T) {
	cert, pool, pin := transportCertificate(t)
	installation := readinessInstallationFixture()
	installation.ServerPins["am"] = pin
	frozen, err := installation.Freeze()
	if err != nil {
		t.Fatal(err)
	}
	offered := &tls.Config{Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}
	plan, err := PrepareProtectedPublication(ProtectedPublicationConfig{Role: "am", Address: "127.0.0.1:18443", Installation: installation, TLS: offered})
	if err != nil || plan == nil {
		t.Fatalf("verified role refused: %v", err)
	}
	if plan.Role() != "am" || plan.config.MinVersion != tls.VersionTLS13 || plan.config.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatal("prepared transport weakened")
	}
	installation.Policy.ClientKey[0] ^= 1
	installation.ServerPins["am"] = Digest("changed")
	offered.ClientAuth = tls.NoClientCert
	offered.Certificates[0].Certificate[0][0] ^= 1
	if plan.config.ClientAuth != tls.RequireAndVerifyClientCert || plan.config.Certificates[0].Certificate[0][0] == offered.Certificates[0].Certificate[0][0] {
		t.Fatal("prepared transport aliases caller state")
	}
	if plan.CheckInstallation(frozen) != nil || plan.CheckInstallation(installation) == nil {
		t.Fatal("complete installation binding lost")
	}
}

func TestProtectedPublicationRejectsPinAndTrustOverridesBeforeListener(t *testing.T) {
	cases := map[string]func(*ProtectedPublicationConfig){
		"server-pin":       func(c *ProtectedPublicationConfig) { c.Installation.ServerPins["am"] = Digest("foreign") },
		"anonymous-client": func(c *ProtectedPublicationConfig) { c.TLS.ClientAuth = tls.NoClientCert },
		"verification-callback": func(c *ProtectedPublicationConfig) {
			c.TLS.VerifyConnection = func(tls.ConnectionState) error { return nil }
		},
		"certificate-callback": func(c *ProtectedPublicationConfig) {
			c.TLS.GetCertificate = func(*tls.ClientHelloInfo) (*tls.Certificate, error) { return nil, nil }
		},
		"weak-max-version": func(c *ProtectedPublicationConfig) { c.TLS.MaxVersion = tls.VersionTLS12 },
		"unbound-hostname": func(c *ProtectedPublicationConfig) { c.Address = "localhost:18443" },
		"privileged-port":  func(c *ProtectedPublicationConfig) { c.Address = "127.0.0.1:443" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cert, pool, pin := transportCertificate(t)
			i := readinessInstallationFixture()
			i.ServerPins["am"] = pin
			c := ProtectedPublicationConfig{Role: "am", Address: "127.0.0.1:18443", Installation: i, TLS: &tls.Config{Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}}
			mutate(&c)
			if plan, err := PrepareProtectedPublication(c); err == nil || plan != nil {
				t.Fatal("invalid transport published plan")
			}
		})
	}
}
