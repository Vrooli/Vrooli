package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	clitest "agent-manager/cli/internal/testutil"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestBuildLocalBrowserTaskCapabilityIsSingleOriginAndBounded(t *testing.T) {
	raw, err := buildLocalBrowserTaskCapability("portal/browser-task", "http://127.0.0.1:24965")
	if err != nil {
		t.Fatal(err)
	}
	var capability localBrowserTaskCapability
	if err := json.Unmarshal([]byte(raw), &capability); err != nil {
		t.Fatal(err)
	}
	if err := validateLocalBrowserTaskCapability(capability); err != nil {
		t.Fatalf("generated capability failed its own validator: %v", err)
	}
	if capability.Permissions.Level != "act" || !capability.Permissions.View || !capability.Permissions.Control {
		t.Fatalf("permissions = %+v, want observe/control task rights", capability.Permissions)
	}
	if len(capability.Permissions.AllowedOrigins) != 1 || capability.Permissions.AllowedOrigins[0] != "http://127.0.0.1:24965" {
		t.Fatalf("allowed origins = %v, want the exact Portal origin", capability.Permissions.AllowedOrigins)
	}
	if capability.Budget.SessionMinutes != 10 || capability.Budget.Actions != 100 || capability.Budget.Bytes != 4<<20 || capability.Budget.InferenceTokens != 0 {
		t.Fatalf("browser budget is not bounded as designed: %+v", capability.Budget)
	}
	if capability.GrantReceipt != "" || capability.Permissions.AllowHistory || len(capability.Permissions.CredentialRefs) != 0 || len(capability.Permissions.ExtensionIDs) != 0 {
		t.Fatalf("generated capability contains broader authority: %+v", capability)
	}
}

func TestBuildLocalBrowserTaskCapabilityRejectsBroadOriginAndRights(t *testing.T) {
	for _, origin := range []string{
		"https://example.com",
		"http://127.0.0.1:24965/apps",
		"http://127.0.0.1:24965/?q=all",
		"http://user:pass@127.0.0.1:24965",
		"http://0.0.0.0:24965",
	} {
		t.Run(origin, func(t *testing.T) {
			if _, err := buildLocalBrowserTaskCapability("portal/browser-task", origin); err == nil {
				t.Fatalf("accepted non-exact or non-loopback origin %q", origin)
			}
		})
	}

	raw, err := buildLocalBrowserTaskCapability("portal/browser-task", "http://127.0.0.1:24965")
	if err != nil {
		t.Fatal(err)
	}
	var capability localBrowserTaskCapability
	if err := json.Unmarshal([]byte(raw), &capability); err != nil {
		t.Fatal(err)
	}
	capability.Permissions.AllowedOrigins = append(capability.Permissions.AllowedOrigins, "https://example.com")
	if err := validateLocalBrowserTaskCapability(capability); err == nil {
		t.Fatal("accepted a capability broadened to a second origin")
	}
	capability.Permissions.AllowedOrigins = []string{"http://127.0.0.1:24965"}
	capability.Permissions.Level = "unrestricted"
	if err := validateLocalBrowserTaskCapability(capability); err == nil {
		t.Fatal("accepted a broadened unrestricted permission level")
	}
}

func TestProfileCreatePersistsBoundedLocalBrowserCapability(t *testing.T) {
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		return `{"profile":{"id":"profile-1","name":"portal-browser-task","profileKey":"portal/browser-task","roleRef":"browser.agent"}}`
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	app := &App{services: NewServices(api)}
	if err := app.profileCreate([]string{
		"--name=portal-browser-task",
		"--profile-key=portal/browser-task",
		"--role-ref=browser.agent",
		"--browser-local-origin=http://127.0.0.1:24965",
	}); err != nil {
		t.Fatal(err)
	}
	requests := server.Requests()
	if len(requests) != 1 || requests[0].Method != http.MethodPost || requests[0].Path != "/api/v1/profiles" {
		t.Fatalf("profile owner requests = %+v", requests)
	}
	var create apipb.CreateProfileRequest
	if err := protojson.Unmarshal([]byte(requests[0].Body), &create); err != nil {
		t.Fatal(err)
	}
	profile := create.GetProfile()
	if profile.GetRoleRef() != "browser.agent" || profile.GetMaxTurns() != 8 || profile.GetTimeout().AsDuration().Minutes() != 15 {
		t.Fatalf("profile task bounds = role %q, turns %d, timeout %s", profile.GetRoleRef(), profile.GetMaxTurns(), profile.GetTimeout().AsDuration())
	}
	if profile.GetNetworkAccess() != domainpb.NetworkAccess_NETWORK_ACCESS_LOCALHOST || profile.GetSandboxConfig().GetMode() != domainpb.SandboxMode_SANDBOX_MODE_PROTECTED || len(profile.GetSandboxConfig().GetWritePolicy().GetPaths()) != 0 {
		t.Fatalf("browser task profile is not local-only and read-only: %+v", profile)
	}
	if got := strings.Join(profile.GetAllowedTools(), ","); got != "read,glob,grep,shell" {
		t.Fatalf("allowed tools = %q, want read,glob,grep,shell", got)
	}
	if profile.GetFeatures() == nil || profile.GetFeatures().GetBrowserCapabilityJson() == "" {
		t.Fatal("profile create did not persist the browser capability")
	}
	var capability localBrowserTaskCapability
	if err := json.Unmarshal([]byte(profile.GetFeatures().GetBrowserCapabilityJson()), &capability); err != nil {
		t.Fatal(err)
	}
	if err := validateLocalBrowserTaskCapability(capability); err != nil {
		t.Fatalf("persisted browser capability is invalid: %v", err)
	}
	if capability.GrantReceipt != "" {
		t.Fatal("profile creation supplied a run grant")
	}
}

func TestProfileCreateRejectsBrowserCapabilityWithoutBrowserRole(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	err = app.profileCreate([]string{
		"--name=portal-browser-task",
		"--profile-key=portal/browser-task",
		"--role-ref=code.default",
		"--browser-local-origin=http://127.0.0.1:24965",
	})
	if err == nil {
		t.Fatal("accepted browser capability on a non-browser role")
	}
}

func TestProfileCreateRejectsBroaderBrowserTaskExecutionRights(t *testing.T) {
	for _, args := range [][]string{
		{"--name=portal-browser-task", "--profile-key=portal/browser-task", "--role-ref=browser.agent", "--browser-local-origin=http://127.0.0.1:24965", "--skip-permissions"},
		{"--name=portal-browser-task", "--profile-key=portal/browser-task", "--role-ref=browser.agent", "--browser-local-origin=http://127.0.0.1:24965", "--allowed-tools=read,write"},
		{"--name=portal-browser-task", "--profile-key=portal/browser-task", "--role-ref=browser.agent", "--browser-local-origin=http://127.0.0.1:24965", "--timeout=0"},
	} {
		app, err := NewApp()
		if err != nil {
			t.Fatal(err)
		}
		if err := app.profileCreate(args); err == nil {
			t.Fatalf("accepted broader task profile options: %v", args)
		}
	}
}
