package resourcehandlers

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
	"github.com/vrooli/vrooli/internal/securestore"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
	"github.com/vrooli/vrooli/internal/testenv"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/rootcli/rootclitest"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

type testContext struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func TestInspectDataHelpDoesNotRequireControllerOrCredential(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
			handler := resourceInspectDataHandler(rootcli.HandlerDeps[testContext]{
				Stdout: func(c testContext) io.Writer { return c.stdout },
			})
			if err := handler(ctx, nil, []string{flag}); err != nil {
				t.Fatalf("help must succeed without loading a manifest or credential: %v", err)
			}
			for _, expected := range []string{"inspect-data", "--git-blob-sha1", "--max-bytes", "--logical-id", "--out"} {
				if !strings.Contains(ctx.stdout.String(), expected) {
					t.Fatalf("help omits %s: %s", expected, ctx.stdout.String())
				}
			}
		})
	}
}

func TestConformance(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	handler := RootHandler(rootcli.HandlerDeps[testContext]{
		Stdout:  func(ctx testContext) io.Writer { return ctx.stdout },
		Stderr:  func(ctx testContext) io.Writer { return ctx.stderr },
		Globals: func(testContext) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		OutputFormat: func(testContext) (cliout.Format, error) {
			return cliout.FormatHuman, nil
		},
	})

	rootclitest.AssertHelpWithNoArgs(t, func() error { return handler(ctx, nil) }, ctx.stdout, "vrooli resource")
}

func TestEnforceResourceHostRequirementsSkipsNonMutatingActions(t *testing.T) {
	called := 0
	prev := enforceHostRequirementsFn
	enforceHostRequirementsFn = func(_ vrooliruntime.Options) (vrooliruntime.Report, error) {
		called++
		return vrooliruntime.Report{}, nil
	}
	t.Cleanup(func() { enforceHostRequirementsFn = prev })

	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{
		Stdout: func(c testContext) io.Writer { return c.stdout },
		Stderr: func(c testContext) io.Writer { return c.stderr },
	}
	controller := &resources.Controller{Root: "/root", Home: "/home"}

	for _, action := range []string{"status", "logs", "stop", "uninstall"} {
		if err := enforceResourceHostRequirements(ctx, deps, controller, "postgres", action); err != nil {
			t.Fatalf("enforceResourceHostRequirements(%s): %v", action, err)
		}
	}
	if called != 0 {
		t.Fatalf("enforce must not fire for non-mutating actions; calls=%d", called)
	}
}

func TestEnforceResourceHostRequirementsFiresForMutatingActions(t *testing.T) {
	prev := enforceHostRequirementsFn
	var captured []vrooliruntime.Options
	enforceHostRequirementsFn = func(opts vrooliruntime.Options) (vrooliruntime.Report, error) {
		captured = append(captured, opts)
		return vrooliruntime.Report{}, nil
	}
	t.Cleanup(func() { enforceHostRequirementsFn = prev })

	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{
		Stdout: func(c testContext) io.Writer { return c.stdout },
		Stderr: func(c testContext) io.Writer { return c.stderr },
	}
	controller := &resources.Controller{Root: "/root", Home: "/home"}

	for _, action := range []string{"install", "start", "restart"} {
		if err := enforceResourceHostRequirements(ctx, deps, controller, "postgres", action); err != nil {
			t.Fatalf("enforceResourceHostRequirements(%s): %v", action, err)
		}
	}
	if len(captured) != 3 {
		t.Fatalf("expected 3 enforce calls, got %d", len(captured))
	}
	for _, opts := range captured {
		if opts.Resources != "postgres" {
			t.Fatalf("Resources = %q, want postgres", opts.Resources)
		}
		if opts.Environment != "development" {
			t.Fatalf("Environment = %q, want development", opts.Environment)
		}
		if opts.Scenarios != "none" {
			t.Fatalf("Scenarios = %q, want none", opts.Scenarios)
		}
		if opts.When != "develop" {
			t.Fatalf("When = %q, want develop", opts.When)
		}
		if opts.Label != "resource:postgres" {
			t.Fatalf("Label = %q, want resource:postgres", opts.Label)
		}
		if !opts.AutoInstall {
			t.Fatal("AutoInstall must be true")
		}
	}
}

func TestEnforceResourceHostRequirementsPropagatesErrors(t *testing.T) {
	prev := enforceHostRequirementsFn
	enforceHostRequirementsFn = func(_ vrooliruntime.Options) (vrooliruntime.Report, error) {
		return vrooliruntime.Report{}, errors.New("docker missing")
	}
	t.Cleanup(func() { enforceHostRequirementsFn = prev })

	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{
		Stdout: func(c testContext) io.Writer { return c.stdout },
		Stderr: func(c testContext) io.Writer { return c.stderr },
	}
	controller := &resources.Controller{Root: "/root", Home: "/home"}

	err := enforceResourceHostRequirements(ctx, deps, controller, "postgres", "install")
	if err == nil || !strings.Contains(err.Error(), "docker missing") {
		t.Fatalf("expected docker missing error, got %v", err)
	}
}

func TestEnforceResourceHostRequirementsUsesControllerEnvironment(t *testing.T) {
	prev := enforceHostRequirementsFn
	var captured vrooliruntime.Options
	enforceHostRequirementsFn = func(opts vrooliruntime.Options) (vrooliruntime.Report, error) {
		captured = opts
		return vrooliruntime.Report{}, nil
	}
	t.Cleanup(func() { enforceHostRequirementsFn = prev })

	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{
		Stdout: func(c testContext) io.Writer { return c.stdout },
		Stderr: func(c testContext) io.Writer { return c.stderr },
	}
	controller := &resources.Controller{Root: "/root", Home: "/home", Environment: "production"}

	if err := enforceResourceHostRequirements(ctx, deps, controller, "postgres", "install"); err != nil {
		t.Fatalf("enforceResourceHostRequirements: %v", err)
	}
	if captured.Environment != "production" {
		t.Fatalf("Environment = %q, want production", captured.Environment)
	}
}

func TestEnforceResourceHostRequirementsUsesProductionForCLIArtifact(t *testing.T) {
	t.Setenv("VROOLI_CLI_ARTIFACT_MODE", "1")
	prev := enforceHostRequirementsFn
	var captured vrooliruntime.Options
	enforceHostRequirementsFn = func(opts vrooliruntime.Options) (vrooliruntime.Report, error) {
		captured = opts
		return vrooliruntime.Report{}, nil
	}
	t.Cleanup(func() { enforceHostRequirementsFn = prev })

	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{
		Stdout: func(c testContext) io.Writer { return c.stdout },
		Stderr: func(c testContext) io.Writer { return c.stderr },
	}
	controller := &resources.Controller{Root: "/root", Home: "/home", Environment: "development"}

	if err := enforceResourceHostRequirements(ctx, deps, controller, "postgres", "start"); err != nil {
		t.Fatalf("enforceResourceHostRequirements: %v", err)
	}
	if captured.Environment != "production" {
		t.Fatalf("Environment = %q, want production", captured.Environment)
	}
}

func TestSingleResourceLifecyclePrecancelHasNoSideEffects(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	sideEffects := 0
	deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }, Stderr: func(c testContext) io.Writer { return c.stderr }, OperationContext: func(testContext) context.Context { return parent }, EnsureCLI: func(testContext, string) error { sideEffects++; return errors.New("unexpected CLI installation") }}
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	err := singleResourceControlHandler(deps, "restart")(ctx, resources.NewController(t.TempDir(), t.TempDir()), []string{"fixture"})
	if !errors.Is(err, context.Canceled) || sideEffects != 0 {
		t.Fatalf("cancelled lifecycle performed work: sideEffects=%d err=%v", sideEffects, err)
	}
}

func TestSingleResourceLifecycleCancelsActualNativeDriverProbe(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "probe-entered")
	binary := filepath.Join(root, "fake-cli")
	t.Setenv("VROOLI_CANCEL_TEST_PROBE", marker)
	installMarker := filepath.Join(root, "unexpected-install")
	t.Setenv("VROOLI_CANCEL_TEST_INSTALL", installMarker)
	if err := os.WriteFile(binary, []byte(shelltest.BashShebang()+"if [ \"$1\" = install ]; then printf unexpected > \"$VROOLI_CANCEL_TEST_INSTALL\"; exit 0; fi\nprintf entered > \"$VROOLI_CANCEL_TEST_PROBE\"\nexec sleep 2\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	testscenario.WriteProjectResourceConfig(t, root, "cancel-fixture", true)
	testresource.WriteResourceManifest(t, root, "cancel-fixture", testresource.ResourceManifest("cancel-fixture", testresource.WithResourceDriver("external-cli"), testresource.WithResourceBinary(binary), testresource.WithResourceVersionArgs("--version"), testresource.WithResourceInstall(manifestpkg.ResourceInstall{Command: []string{binary, "install"}})))
	previous := enforceHostRequirementsFn
	enforceHostRequirementsFn = func(vrooliruntime.Options) (vrooliruntime.Report, error) { return vrooliruntime.Report{}, nil }
	t.Cleanup(func() { enforceHostRequirementsFn = previous })
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }, Stderr: func(c testContext) io.Writer { return c.stderr }, OperationContext: func(testContext) context.Context { return parent }}
	done := make(chan error, 1)
	go func() {
		done <- singleResourceControlHandler(deps, "restart")(ctx, resources.NewController(root, t.TempDir()), []string{"cancel-fixture"})
	}()
	entered := time.NewTicker(time.Millisecond)
	defer entered.Stop()
	deadline := time.After(time.Second)
wait:
	for {
		select {
		case <-entered.C:
			if _, err := os.Stat(marker); err == nil {
				break wait
			}
		case err := <-done:
			t.Fatalf("driver did not enter fake probe: %v", err)
		case <-deadline:
			t.Fatal("fake probe did not start")
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("actual handler/driver lost cancellation: %v", err)
		}
	case <-time.After(time.Millisecond * 500):
		err := <-done
		t.Fatalf("driver probe ignored cancellation: %v", err)
	}
	if _, err := os.Stat(installMarker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled probe fell through to install: %v", err)
	}
}

func TestInspectDataActualHandlerRejectsInvalidArgumentsWithoutController(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }}
	handler := buildResourceAcquisitionCommandHandlers(deps)["inspect-data"]
	if handler == nil {
		t.Fatal("inspect-data not registered")
	}
	for _, args := range [][]string{{}, {"fixture"}, {"fixture", "--url", "https://example.test/data"}, {"fixture", "--bogus"}} {
		if err := handler(ctx, nil, args); err == nil {
			t.Fatalf("invalid args accepted %q", args)
		}
	}
}

func TestInspectDataActualHandlerPublishesVerifiedMetadataAndJSONReceipt(t *testing.T) {
	root := t.TempDir()
	manifest := testresource.ResourceManifest("inspect-fixture", testresource.WithResourceDriver("managed-service"))
	manifest.Credentials = manifestpkg.ResourceCredentials{Descriptors: []manifestpkg.CredentialDescriptor{{LogicalID: "vrooli/inspect-fixture", Field: "token"}}}
	testresource.WriteResourceManifest(t, root, manifest.Name, manifest)
	authority, err := credentialauthority.NewAuthority(testenv.NewCredentialStore(securestore.ErrNotFound))
	if err != nil {
		t.Fatal(err)
	}
	if err = authority.Put("vrooli/inspect-fixture", "token", "handler-private-token"); err != nil {
		t.Fatal(err)
	}
	oldAuthority := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = oldAuthority })
	body := []byte(`{"valid":true}`)
	git := fmt.Sprintf("%x", sha1.Sum(append([]byte(fmt.Sprintf("blob %d%c", len(body), 0)), body...)))
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer handler-private-token" {
			t.Error("missing auth")
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	manifest.ManagedService.DataArtifacts = []resourcedeployment.ManagedServiceDataArtifact{{Name: "config", Path: "config.json", Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: srv.URL, Auth: &binaryfetch.AcquisitionAuth{Scheme: "bearer", LogicalID: "vrooli/inspect-fixture", Field: "token"}}}}}}
	testresource.WriteResourceManifest(t, root, manifest.Name, manifest)

	oldClient := binaryfetch.HTTPClient
	binaryfetch.HTTPClient = srv.Client()
	t.Cleanup(func() { binaryfetch.HTTPClient = oldClient })
	ctx := testContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	deps := rootcli.HandlerDeps[testContext]{Stdout: func(c testContext) io.Writer { return c.stdout }}
	out := filepath.Join(t.TempDir(), "config.json")
	err = buildResourceAcquisitionCommandHandlers(deps)["inspect-data"](ctx, resources.NewController(root, t.TempDir()), []string{manifest.Name, "--url", srv.URL, "--git-blob-sha1", git, "--max-bytes", "1024", "--out", out, "--logical-id", "vrooli/inspect-fixture", "--field", "token"})
	if err != nil {
		t.Fatal(err)
	}
	var receipt binaryfetch.DataInspectionReceipt
	if err = json.Unmarshal(ctx.stdout.Bytes(), &receipt); err != nil || receipt.Bytes != int64(len(body)) || receipt.SHA256 != fmt.Sprintf("%x", sha256.Sum256(body)) {
		t.Fatalf("wrong receipt %+v %v", receipt, err)
	}
	if strings.Contains(ctx.stdout.String()+ctx.stderr.String(), "handler-private-token") {
		t.Fatal("secret in output")
	}
	got, err := os.ReadFile(out)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatal("wrong published data", err)
	}
}
