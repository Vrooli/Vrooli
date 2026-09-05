package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	"github.com/vrooli/vrooli/internal/testenv"
)

func withCredentialStore(t *testing.T, store securestore.Store) *credentialauthority.Authority {
	t.Helper()
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previous := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = previous })
	return authority
}

func openrouterCloudManifest() ResourceManifest {
	return ResourceManifest{
		Name:     "openrouter",
		Driver:   "cloud-api",
		Endpoint: "https://openrouter.ai/api/v1",
		Credentials: manifestpkg.ResourceCredentials{Descriptors: []manifestpkg.CredentialDescriptor{{
			LogicalID: "vrooli/openrouter",
			Field:     "api-key",
			Env:       "OPENROUTER_API_KEY",
			Label:     "OpenRouter API Key",
			Required:  true,
			ObtainURL: "https://openrouter.ai/keys",
		}}},
	}
}

// A credential condition makes the resource unhealthy and never makes the
// status call fail. The two conditions get different messages because they
// have different fixes.
func TestCloudAPIStatusSeparatesHostFaultFromUnsetValue(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		store       securestore.Store
		wantMessage string
		wantCode    string
	}{
		{
			name:        "unreachable store names the host fix",
			store:       securestore.Unavailable("secret-tool: Could not connect: Permission denied"),
			wantMessage: "credential store unreachable",
			wantCode:    StatusCodeCommandError,
		},
		{
			name:        "no backend names what to install",
			store:       securestore.Absent("no adapter for this platform"),
			wantMessage: "no credential backend on this host",
			wantCode:    StatusCodeCommandError,
		},
		{
			name:        "unset value names the provision command",
			store:       testenv.NewCredentialStore(securestore.ErrNotFound),
			wantMessage: "vrooli credentials provision --identity vrooli/openrouter --field api-key",
			wantCode:    StatusCodeOK,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			withCredentialStore(t, testCase.store)

			status, err := cloudAPIDriver{}.Status(context.Background(), &Controller{}, Resource{Name: "openrouter"},
				openrouterCloudManifest(), true)
			if err != nil {
				t.Fatalf("Status() error = %v, want a status rather than a failure", err)
			}
			if status.Healthy == nil || *status.Healthy {
				t.Fatalf("Healthy = %v, want unhealthy", status.Healthy)
			}
			if !strings.Contains(status.Message, testCase.wantMessage) {
				t.Fatalf("Message = %q, want it to contain %q", status.Message, testCase.wantMessage)
			}
			if !strings.Contains(status.Message, "OPENROUTER_API_KEY") {
				t.Fatalf("Message = %q, want it to name the variable", status.Message)
			}
			if status.StatusCode != testCase.wantCode {
				t.Fatalf("StatusCode = %q, want %q", status.StatusCode, testCase.wantCode)
			}
		})
	}
}

// TestCloudAPIStatusRecoversAfterProvisioningWithoutRestart is the
// start-then-configure guarantee at the health boundary: the resource must go
// healthy in the same process that reported it unhealthy.
func TestCloudAPIStatusRecoversAfterProvisioningWithoutRestart(t *testing.T) {
	authority := withCredentialStore(t, testenv.NewCredentialStore(securestore.ErrNotFound))
	manifest := openrouterCloudManifest()

	before, err := cloudAPIDriver{}.Status(context.Background(), &Controller{}, Resource{Name: "openrouter"}, manifest, true)
	if err != nil {
		t.Fatal(err)
	}
	if before.Healthy == nil || *before.Healthy {
		t.Fatalf("Healthy = %v before provisioning, want unhealthy", before.Healthy)
	}

	if err := authority.Put("vrooli/openrouter", "api-key", "sk-provisioned-after-start"); err != nil {
		t.Fatal(err)
	}

	after, err := cloudAPIDriver{}.Status(context.Background(), &Controller{}, Resource{Name: "openrouter"}, manifest, true)
	if err != nil {
		t.Fatal(err)
	}
	if after.Healthy == nil || !*after.Healthy {
		t.Fatalf("Healthy = %v after provisioning, want healthy with no control-plane restart", after.Healthy)
	}
	if strings.Contains(after.Message, "OPENROUTER_API_KEY") {
		t.Fatalf("Message = %q, want the credential gap gone", after.Message)
	}
}

// The status message is operator-facing output, so it must never carry the
// value it is reporting on.
func TestCloudAPIStatusNeverPrintsACredentialValue(t *testing.T) {
	authority := withCredentialStore(t, testenv.NewCredentialStore(securestore.ErrNotFound))
	if err := authority.Put("vrooli/openrouter", "api-key", "sk-secret-value"); err != nil {
		t.Fatal(err)
	}
	status, err := cloudAPIDriver{}.Status(context.Background(), &Controller{}, Resource{Name: "openrouter"},
		openrouterCloudManifest(), true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(status.Message+status.ProbeError, "sk-secret-value") {
		t.Fatalf("status leaked a credential value: %+v", status)
	}
}
