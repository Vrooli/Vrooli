package scenariocli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
)

func misModedHub() *lifecycle.AuthenticationStatus {
	return &lifecycle.AuthenticationStatus{
		Profile:        "hybrid",
		SupportedModes: []string{"personal_local", "shared_provider"},
		Configured:     lifecycle.AuthModeResolution{Mode: "shared_provider", Source: lifecycle.AuthModeSourceManifest},
		Processes: []lifecycle.ProcessAuthentication{
			{Step: "start-api", PID: 4321, Mode: "personal_local"},
			{Step: "start-ui", PID: 4322, Mode: "shared_provider", Source: lifecycle.AuthModeSourceManifest, Providers: "cloudflare_access,scenario_authenticator"},
		},
		Mismatch: true,
	}
}

func TestStatusShowsConfiguredAuthModeSourceAndMisModedProcess(t *testing.T) {
	resp := StatusResponse{Single: &StatusSingleOutput{
		Scenario:       StatusItemOutput{Name: "notification-hub", Status: "running"},
		Info:           InfoScenarioData{Name: "notification-hub", Path: "/repo/scenarios/notification-hub"},
		Authentication: misModedHub(),
	}}
	var human bytes.Buffer
	if err := RenderStatusResponse(&human, cliout.FormatHuman, resp); err != nil {
		t.Fatalf("render human: %v", err)
	}
	for _, want := range []string{
		"Authentication: profile=hybrid mode=shared_provider source=manifest-default",
		"start-api pid=4321 mode=personal_local source=unrecorded",
		"start-ui pid=4322 mode=shared_provider source=manifest-default providers=cloudflare_access,scenario_authenticator",
		"[WARN]    A running process does not carry the configured authentication mode shared_provider",
	} {
		if !strings.Contains(human.String(), want) {
			t.Fatalf("missing %q in:\n%s", want, human.String())
		}
	}

	var encoded bytes.Buffer
	if err := RenderStatusResponse(&encoded, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("render json: %v", err)
	}
	var payload struct {
		Authentication struct {
			ConfiguredMode   string `json:"configured_mode"`
			ConfiguredSource string `json:"configured_source"`
			Mismatch         bool   `json:"mismatch"`
			Processes        []struct {
				Step string `json:"step"`
				Mode string `json:"mode"`
			} `json:"processes"`
		} `json:"authentication"`
	}
	if err := json.Unmarshal(encoded.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v\n%s", err, encoded.String())
	}
	auth := payload.Authentication
	if auth.ConfiguredMode != "shared_provider" || auth.ConfiguredSource != "manifest-default" || !auth.Mismatch || len(auth.Processes) != 2 || auth.Processes[0].Mode != "personal_local" {
		t.Fatalf("authentication json = %+v\n%s", auth, encoded.String())
	}
}

func TestStatusOmitsAuthenticationForScenarioWithoutProfile(t *testing.T) {
	var human bytes.Buffer
	WriteStatusHuman(&human, StatusSingleOutput{Info: InfoScenarioData{Name: "plain"}})
	if strings.Contains(human.String(), "Authentication:") {
		t.Fatalf("unexpected authentication section:\n%s", human.String())
	}
}

func TestStatusNamesPersonalLocalRiskAndRefusal(t *testing.T) {
	cases := []struct {
		name  string
		auth  *lifecycle.AuthenticationStatus
		human string
		field string
		value string
	}{
		{
			name:  "personal_local risk",
			auth:  &lifecycle.AuthenticationStatus{Profile: "hybrid", Configured: lifecycle.AuthModeResolution{Mode: lifecycle.AuthModePersonalLocal, Source: lifecycle.AuthModeSourceManifest}, Risk: lifecycle.PersonalLocalRisk},
			human: "[WARN]    Authentication mode personal_local: " + lifecycle.PersonalLocalRisk,
			field: "risk", value: lifecycle.PersonalLocalRisk,
		},
		{
			name:  "refused mode",
			auth:  &lifecycle.AuthenticationStatus{Profile: "cloudflare_access", Configured: lifecycle.AuthModeResolution{Mode: lifecycle.AuthModePersonalLocal, Source: lifecycle.AuthModeSourceManifest}, Refused: "default_mode personal_local is refused"},
			human: "[ERROR]   The next start refuses this authentication mode: default_mode personal_local is refused",
			field: "refused", value: "default_mode personal_local is refused",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := StatusResponse{Single: &StatusSingleOutput{Info: InfoScenarioData{Name: "console"}, Authentication: tc.auth}}
			var human bytes.Buffer
			if err := RenderStatusResponse(&human, cliout.FormatHuman, resp); err != nil {
				t.Fatalf("render human: %v", err)
			}
			if !strings.Contains(human.String(), tc.human) {
				t.Fatalf("missing %q in:\n%s", tc.human, human.String())
			}
			var encoded bytes.Buffer
			if err := RenderStatusResponse(&encoded, cliout.FormatJSON, resp); err != nil {
				t.Fatalf("render json: %v", err)
			}
			var payload struct {
				Authentication map[string]any `json:"authentication"`
			}
			if err := json.Unmarshal(encoded.Bytes(), &payload); err != nil {
				t.Fatalf("decode json: %v", err)
			}
			if payload.Authentication[tc.field] != tc.value {
				t.Fatalf("authentication.%s = %v\n%s", tc.field, payload.Authentication[tc.field], encoded.String())
			}
		})
	}
}
