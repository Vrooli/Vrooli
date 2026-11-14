package target

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func validTarget() *domainv1.AppTarget {
	return &domainv1.AppTarget{
		TargetId:       "local-linux",
		ScenarioName:   "example-app",
		ArtifactDigest: "sha256:artifact",
		ContextId:      "ctx-1",
		ProcessId:      "1234",
		CdpEndpoint:    "http://127.0.0.1:9222",
		CdpTransport:   "loopback-authenticated",
		LoopbackOnly:   true,
		Authenticated:  true,
		RendererId:     "web-contents-1",
		RendererUrl:    "file:///controlled/index.html",
	}
}

func validContext() *domainv1.ValidationContext {
	return &domainv1.ValidationContext{
		ContextId:        "ctx-1",
		ScenarioName:     "example-app",
		ArtifactDigest:   "sha256:artifact",
		TargetId:         "local-linux",
		JourneyId:        "login",
		ProfileId:        "normal",
		IsolationLeaseId: "lease-1",
		ExpiresAt:        timestamppb.New(time.Unix(200, 0)),
	}
}

func TestValidateAppTargetAcceptsAuthenticatedNumericLoopback(t *testing.T) {
	if err := ValidateAppTarget(validTarget()); err != nil {
		t.Fatalf("valid target rejected: %v", err)
	}
}

func TestValidateAppTargetFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domainv1.AppTarget)
		want   string
	}{
		{"public endpoint", func(target *domainv1.AppTarget) { target.CdpEndpoint = "http://192.0.2.10:9222" }, "numeric loopback"},
		{"hostname alias", func(target *domainv1.AppTarget) { target.CdpEndpoint = "http://localhost:9222" }, "numeric loopback"},
		{"unauthenticated", func(target *domainv1.AppTarget) { target.Authenticated = false }, "authenticated"},
		{"missing port", func(target *domainv1.AppTarget) { target.CdpEndpoint = "http://127.0.0.1" }, "port"},
		{"nonloopback API URL", func(target *domainv1.AppTarget) { target.ApiUrl = "http://192.0.2.5:19349" }, "loopback"},
		{"hostname API URL", func(target *domainv1.AppTarget) { target.ApiUrl = "http://localhost:19349" }, "loopback"},
		{"API URL path", func(target *domainv1.AppTarget) { target.ApiUrl = "http://127.0.0.1:19349/api" }, "bare loopback"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateAppTarget(func() *domainv1.AppTarget {
				target := validTarget()
				test.mutate(target)
				return target
			}())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestValidateElectronContextBindsLeaseAndArtifact(t *testing.T) {
	now := time.Unix(100, 0)
	if err := ValidateElectronContext(validTarget(), validContext(), now); err != nil {
		t.Fatalf("valid context rejected: %v", err)
	}

	context := validContext()
	context.ArtifactDigest = "sha256:other"
	if err := ValidateElectronContext(validTarget(), context, now); err == nil || !strings.Contains(err.Error(), "artifact") {
		t.Fatalf("artifact mismatch error = %v", err)
	}

	context = validContext()
	context.ExpiresAt = timestamppb.New(time.Unix(100, 0))
	if err := ValidateElectronContext(validTarget(), context, now); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired lease error = %v", err)
	}
}

func TestBuildElectronLaunchArgumentsBindsLoopbackAndScopedIdentity(t *testing.T) {
	args, env, err := BuildElectronLaunchArguments(ElectronLaunchOptions{
		CDPPort:          9223,
		UserDataDir:      "/tmp/vrooli-validation/profile",
		ContextID:        "ctx-1",
		ScenarioName:     "example-app",
		ArtifactDigest:   "sha256:artifact",
		TargetID:         "local-linux",
		JourneyID:        "create-item",
		ProfileID:        "normal",
		IsolationLeaseID: "lease-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=9223",
		"--user-data-dir=/tmp/vrooli-validation/profile",
	}
	if runtime.GOOS == "linux" {
		wantArgs = append(wantArgs, "--no-sandbox")
	}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("launch args = %#v, want %#v", args, wantArgs)
	}
	if env["VROOLI_VALIDATION_CONTEXT_ID"] != "ctx-1" || env["VROOLI_VALIDATION_ISOLATION_LEASE"] != "lease-1" {
		t.Fatalf("scoped validation environment = %#v", env)
	}
}

func TestBuildElectronLaunchArgumentsRejectsIncompleteScope(t *testing.T) {
	_, _, err := BuildElectronLaunchArguments(ElectronLaunchOptions{CDPPort: 9223, UserDataDir: "/tmp/profile"})
	if err == nil {
		t.Fatal("expected incomplete scope rejection")
	}
}
