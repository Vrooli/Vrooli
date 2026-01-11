package vps

import (
	"strings"
	"testing"

	"scenario-to-cloud/domain"
)

func TestBuildCaddyfileWithDNS01(t *testing.T) {
	manifest := domain.CloudManifest{
		Edge: domain.ManifestEdge{
			Domain: "example.com",
			Caddy: domain.ManifestCaddy{
				Enabled: true,
				Email:   "ops@example.com",
			},
		},
	}
	cfg := buildCaddyTLSConfig(manifest, map[string]string{
		domain.CloudflareAPITokenKey: "token123",
	})
	got := BuildCaddyfile("example.com", 3000, cfg)
	if !strings.Contains(got, "dns cloudflare token123") {
		t.Fatalf("expected DNS-01 block, got: %s", got)
	}
	if !strings.Contains(got, "acme_ca https://acme-v02.api.letsencrypt.org/directory") {
		t.Fatalf("expected acme_ca in global options, got: %s", got)
	}
	if !strings.Contains(got, "email ops@example.com") {
		t.Fatalf("expected email in global options, got: %s", got)
	}
}

func TestBuildCaddyfileWithoutDNS01(t *testing.T) {
	manifest := domain.CloudManifest{
		Edge: domain.ManifestEdge{
			Domain: "example.com",
			Caddy:  domain.ManifestCaddy{Enabled: true},
		},
	}
	cfg := buildCaddyTLSConfig(manifest, nil)
	got := BuildCaddyfile("example.com", 3000, cfg)
	if strings.Contains(got, "dns cloudflare") {
		t.Fatalf("did not expect DNS-01 block, got: %s", got)
	}
	if strings.Contains(got, "tls {") {
		t.Fatalf("did not expect TLS block without DNS-01, got: %s", got)
	}
}

func TestBuildCaddyGlobalOptionsDefaultsACMECA(t *testing.T) {
	cfg := CaddyTLSConfig{}
	got := buildCaddyGlobalOptions(cfg)
	if !strings.Contains(got, "acme_ca https://acme-v02.api.letsencrypt.org/directory") {
		t.Fatalf("expected default acme_ca, got: %s", got)
	}
}

func TestBuildCaddyTLSBlockMissingToken(t *testing.T) {
	cfg := CaddyTLSConfig{DNSProvider: "cloudflare"}
	got := buildCaddyTLSBlock(cfg)
	if got != "" {
		t.Fatalf("expected empty TLS block when token is missing, got: %s", got)
	}
}

func TestBuildCaddyfileDefaultsDomain(t *testing.T) {
	cfg := CaddyTLSConfig{}
	got := BuildCaddyfile("", 3000, cfg)
	if !strings.Contains(got, "example.com") {
		t.Fatalf("expected default domain example.com, got: %s", got)
	}
}

func TestCaddyACMEOriginUnreachableHintDNS01(t *testing.T) {
	logs := "2025/01/01 00:00:00 acme: error: remaining=[dns-01] no solvers available"
	hint := caddyACMEOriginUnreachableHint(logs, false)
	if !strings.Contains(hint, "DNS-01") {
		t.Fatalf("expected DNS-01 hint, got: %s", hint)
	}
}

func TestCaddyACMEOriginUnreachableHintProxyError(t *testing.T) {
	logs := "2025/01/01 00:00:00 acme: error: 522 origin error"
	hint := caddyACMEOriginUnreachableHint(logs, true)
	if !strings.Contains(hint, "origin unreachable") {
		t.Fatalf("expected origin unreachable hint, got: %s", hint)
	}
}

func TestCaddyACMEOriginUnreachableHintNoACME(t *testing.T) {
	hint := caddyACMEOriginUnreachableHint("all good", false)
	if hint != "" {
		t.Fatalf("expected empty hint, got: %s", hint)
	}
}

func TestCaddyACMEOriginUnreachableHintIncludesLogs(t *testing.T) {
	logs := strings.Join([]string{
		"2025/01/01 00:00:00 acme: error: 522 origin error",
		"2025/01/01 00:00:01 acme: error: 522 origin error",
		"2025/01/01 00:00:02 acme: error: 522 origin error",
		"2025/01/01 00:00:03 acme: error: 522 origin error",
	}, "\n")
	hint := caddyACMEOriginUnreachableHint(logs, true)
	if strings.Count(hint, "2025/01/01") > 3 {
		t.Fatalf("expected log snippet capped to 3 lines, got: %s", hint)
	}
}
