package passkeys

import "testing"

func TestLoadConfigRejectsRPIDOutsideOrigin(t *testing.T) {
	t.Setenv("LPBS_WEBAUTHN_RP_ID", "attacker.example")
	if _, err := LoadConfig("https://vrooli.com", "LPBS", true); err == nil {
		t.Fatal("expected RP ID mismatch")
	}
}

func TestLoadConfigAllowsParentRPIDAndProductionHTTPSOrigins(t *testing.T) {
	t.Setenv("LPBS_WEBAUTHN_RP_ID", "example.com")
	t.Setenv("LPBS_WEBAUTHN_EXTRA_ORIGINS", "https://www.example.com")
	config, err := LoadConfig("https://login.example.com", "LPBS", true)
	if err != nil || config.RPID != "example.com" || len(config.RPOrigins) != 2 {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestLoadConfigRejectsHTTPExtraOriginInProduction(t *testing.T) {
	t.Setenv("LPBS_WEBAUTHN_EXTRA_ORIGINS", "http://localhost:8080")
	if _, err := LoadConfig("https://vrooli.com", "LPBS", true); err == nil {
		t.Fatal("expected insecure origin rejection")
	}
}
