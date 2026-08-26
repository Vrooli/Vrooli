package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlease"
	"github.com/vrooli/vrooli/internal/artifactledger"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func volumeSourceLooksLikeFile(volume ResourceVolume) bool {
	sourceBase := filepath.Base(filepath.FromSlash(volume.Source))
	targetBase := filepath.Base(filepath.FromSlash(volume.Target))
	if strings.HasPrefix(sourceBase, ".") || strings.HasPrefix(targetBase, ".") {
		return true
	}
	return strings.Contains(sourceBase, ".") || strings.Contains(targetBase, ".")
}

type externalCLIDriver struct{}

func (externalCLIDriver) Name() string { return "external-cli" }

type nativeCLIDriver struct{}

func (nativeCLIDriver) Name() string { return "native-cli" }

func (d externalCLIDriver) Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	status := Status{
		Resource:   item,
		StatusCode: StatusCodeOK,
		Message:    "not installed",
	}
	if err := ensureSupportedPlatform(manifest); err != nil {
		status.StatusCode = StatusCodeUnsupportedPlatform
		status.Message = err.Error()
		status.ProbeError = err.Error()
		return status, nil
	}

	binary := externalCLIBinary(manifest)
	if _, err := lookPathCommandFn(binary); err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = fmt.Sprintf("%s is unavailable", binary)
		status.ProbeError = err.Error()
		return status, nil
	}
	if err := probeExternalCLICommand(ctx, controller, manifest); err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = fmt.Sprintf("%s is unavailable", binary)
		status.ProbeError = err.Error()
		healthy := false
		status.Healthy = &healthy
		status.Health = "unhealthy"
		return status, nil
	}

	status.Installed = true
	status.Running = true
	healthy := true
	status.Healthy = &healthy
	status.Health = "healthy"
	status.Message = "available"

	if fast {
		return status, nil
	}
	if len(manifest.HealthChecks) > 0 {
		health, err := controller.runResourceHealthChecks(ctx, manifest)
		if err != nil {
			status.StatusCode = StatusCodeCommandError
			status.Message = "health checks failed"
			status.ProbeError = err.Error()
			return status, nil
		}
		status = applyHealthToStatus(status, health)
		if strings.TrimSpace(status.Message) == "" {
			if health.Healthy {
				status.Message = "healthy"
			} else {
				status.Message = "unhealthy"
			}
		}
	}

	return status, nil
}

func (d externalCLIDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	if err := ensureSupportedPlatform(manifest); err != nil {
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Platform",
			Err:       err,
		}
	}

	switch action {
	case "status":
		status, err := d.Status(ctx, controller, item, manifest, !containsString(args, "--no-fast"))
		if err != nil {
			return err
		}
		if containsString(args, "--format") && nextArgValue(args, "--format") == "json" {
			return json.NewEncoder(stdout).Encode(map[string]any{
				"installed": status.Installed,
				"running":   status.Running,
				"healthy":   status.Healthy,
				"health":    status.Health,
				"message":   status.Message,
			})
		}
		_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
		return err
	case "install":
		return runInstallCommand(ctx, controller, manifest)
	case "start", "restart":
		status, err := d.Status(ctx, controller, item, manifest, false)
		if err != nil {
			return err
		}
		if status.Running && status.Healthy != nil && *status.Healthy {
			_, err := fmt.Fprintf(stdout, "%s does not require a managed start step\n", item.Name)
			return err
		}
		if usesManagedDiscoveredProvider(manifest) {
			return &Error{
				Code:      ErrorCodeCommandUnavailable,
				Resource:  item.Name,
				Operation: action,
				Category:  "Provider",
				Err:       fmt.Errorf("managed-discovered provider will not install or adopt an unverified host tool; run explicit install or select a Vrooli-managed fallback"),
			}
		}
		return runInstallCommand(ctx, controller, manifest)
	case "stop":
		_, err := fmt.Fprintf(stdout, "%s does not run as a managed background service\n", item.Name)
		return err
	case "uninstall":
		return uninstallNativeCLI(controller, manifest)
	case "logs":
		if manifest.Capabilities.SupportsLogs {
			candidates := managedLogCandidates(controller, manifest)
			if len(candidates) > 0 {
				_, err := fmt.Fprintf(stdout, "%s managed logs may be available under:\n- %s\n", item.Name, strings.Join(candidates, "\n- "))
				return err
			}
		}
		_, err := fmt.Fprintf(stdout, "%s does not expose managed logs through the resource control plane\n", item.Name)
		return err
	default:
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Driver",
			Err:       fmt.Errorf("action %q is not supported by driver %q", action, d.Name()),
		}
	}
}

func usesManagedDiscoveredProvider(manifest ResourceManifest) bool {
	if manifest.ProviderPolicy == nil {
		return false
	}
	mode, err := manifest.ProviderPolicy.ResolveProvider(resourcedeployment.ProviderRequest{})
	return err == nil && mode == resourcedeployment.ProviderManagedDiscovered
}

func (d nativeCLIDriver) Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	if manifest.ManagedService != nil && len(manifest.ManagedService.DataArtifacts) > 0 {
		fast = false
	}
	return externalCLIDriver{}.Status(ctx, controller, item, manifest, fast)
}

func (d nativeCLIDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	if action == "start" || action == "restart" {
		status, err := d.Status(ctx, controller, item, manifest, false)
		if err != nil {
			return err
		}
		if status.Healthy == nil || !*status.Healthy {
			message := status.Message
			if strings.TrimSpace(status.ProbeError) != "" {
				message += ": " + status.ProbeError
			}
			if strings.TrimSpace(message) == "" {
				message = "readiness check failed"
			}
			return &Error{Code: ErrorCodeCommandUnavailable, Resource: item.Name, Operation: action, Category: "Readiness", Err: fmt.Errorf("%s: %s", item.Name, message)}
		}
		if err := gateAcceleratorReadiness(ctx, manifest, stderr); err != nil {
			return err
		}
		if err := (externalCLIDriver{}).Run(ctx, controller, item, manifest, action, args, stdout, stderr); err != nil {
			return err
		}

		return verifyStartedPlacement(ctx, controller, manifest, stderr)
	}
	return externalCLIDriver{}.Run(ctx, controller, item, manifest, action, args, stdout, stderr)
}

func probeExternalCLICommand(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	binary := strings.TrimSpace(externalCLIBinary(manifest))
	if binary == "" {
		return nil
	}
	args := append([]string(nil), manifest.VersionArgs...)
	if len(args) == 0 {
		return nil
	}

	cmd := shell.Command(shell.Spec{
		Name:  binary,
		Args:  args,
		Dir:   controller.Root,
		Env:   resourceEnvForResource(controller.Root, controller.Home, manifest.Name),
		Stdin: nil,
	})
	result := runCommandResource(ctx, cmd)
	if result.err != nil {
		if detail := strings.TrimSpace(string(result.output)); detail != "" {
			return fmt.Errorf("%w: %s", result.err, detail)
		}
	}
	return result.err
}

// credentialGapMessage gives the operator one line that names both what is
// wrong and how to fix it. The two classes stay distinct: a host condition
// tells the operator to repair the session, an unset value tells them to
// provision it.
func credentialGapMessage(gaps resourceenv.CredentialResolution) string {
	names := make([]string, 0, len(gaps.Missing))
	for _, gap := range gaps.Missing {
		names = append(names, gap.Env)
	}
	first := gaps.Missing[0]
	switch first.Reason {
	case resourceenv.GapProviderUnavailable:
		return "credential store unreachable, so " + strings.Join(names, ", ") + " could not be read: " + first.Remediation
	case resourceenv.GapProviderAbsent:
		return "no credential backend on this host, so " + strings.Join(names, ", ") + " could not be read: " + first.Remediation
	default:
		return "missing credentials: " + strings.Join(names, ", ") + "; " + first.Remediation
	}
}

type cloudAPIDriver struct{}

func (cloudAPIDriver) Name() string { return "cloud-api" }

func (d cloudAPIDriver) Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	status := Status{
		Resource:   item,
		Installed:  true,
		Running:    true,
		StatusCode: StatusCodeOK,
		Message:    "configured",
	}
	if err := ensureSupportedPlatform(manifest); err != nil {
		status.StatusCode = StatusCodeUnsupportedPlatform
		status.Message = err.Error()
		status.ProbeError = err.Error()
		return status, nil
	}

	gaps, err := resourceenv.ResolveCredentialGaps(manifest)
	if err != nil {

		healthy := false
		status.Healthy = &healthy
		status.Health = "unhealthy"
		status.StatusCode = StatusCodeCommandError
		status.Message = "credential declaration is invalid"
		status.ProbeError = err.Error()
		return status, nil
	}
	if len(gaps.Missing) > 0 {
		healthy := false
		status.Healthy = &healthy
		status.Health = "unhealthy"
		status.Message = credentialGapMessage(gaps)
		if gaps.Provider != credentialauthority.ProviderAvailable {

			status.StatusCode = StatusCodeCommandError
			status.ProbeError = gaps.Missing[0].Detail
		}
		return status, nil
	}

	healthy := true
	status.Healthy = &healthy
	status.Health = "healthy"

	if fast {
		return status, nil
	}
	if len(manifest.HealthChecks) > 0 {
		health, err := controller.runResourceHealthChecks(ctx, manifest)
		if err != nil {
			status.StatusCode = StatusCodeCommandError
			status.Message = "health checks failed"
			status.ProbeError = err.Error()
			return status, nil
		}
		status = applyHealthToStatus(status, health)
		if strings.TrimSpace(status.Message) == "" {
			if health.Healthy {
				status.Message = "healthy"
			} else {
				status.Message = "unhealthy"
			}
		}
	}

	return status, nil
}

func (d cloudAPIDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	if err := ensureSupportedPlatform(manifest); err != nil {
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Platform",
			Err:       err,
		}
	}

	switch action {
	case "status":
		status, err := d.Status(ctx, controller, item, manifest, !containsString(args, "--no-fast"))
		if err != nil {
			return err
		}
		if containsString(args, "--format") && nextArgValue(args, "--format") == "json" {
			return json.NewEncoder(stdout).Encode(map[string]any{
				"installed": status.Installed,
				"running":   status.Running,
				"healthy":   status.Healthy,
				"health":    status.Health,
				"message":   status.Message,
			})
		}
		_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
		return err
	case "install":
		return runInstallCommand(ctx, controller, manifest)
	case "start", "restart":
		_, err := fmt.Fprintf(stdout, "%s is a hosted API and does not have a local start step\n", item.Name)
		return err
	case "stop":
		_, err := fmt.Fprintf(stdout, "%s is a hosted API and does not have a local stop step\n", item.Name)
		return err
	case "logs":
		if manifest.Capabilities.SupportsLogs {
			candidates := managedLogCandidates(controller, manifest)
			if len(candidates) > 0 {
				_, err := fmt.Fprintf(stdout, "%s managed logs may be available under:\n- %s\n", item.Name, strings.Join(candidates, "\n- "))
				return err
			}
		}
		_, err := fmt.Fprintf(stdout, "%s does not expose managed logs through the resource control plane\n", item.Name)
		return err
	default:
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Driver",
			Err:       fmt.Errorf("action %q is not supported by driver %q", action, d.Name()),
		}
	}
}

func externalCLIBinary(manifest ResourceManifest) string {
	if strings.TrimSpace(manifest.Binary) != "" {
		return strings.TrimSpace(manifest.Binary)
	}
	return manifest.Name
}

func uninstallNativeCLI(controller *Controller, manifest ResourceManifest) error {
	runtimeHome, err := repocontract.VrooliUserRoot(controller.Home)
	if err != nil {
		return fmt.Errorf("resolve runtime home: %w", err)
	}
	binDir := filepath.Join(runtimeHome, "bin")
	binary := filepath.Join(binDir, externalCLIBinary(manifest))
	cleanBinary, err := filepath.Abs(binary)
	if err != nil || filepath.Dir(cleanBinary) != binDir {
		return fmt.Errorf("refuse to uninstall unsafe native CLI path %q", binary)
	}
	// Resource manifests share ~/.vrooli/bin with the control-plane and
	// scenario CLIs. A malformed or future manifest must never turn resource
	// uninstall into a root-control-plane removal.
	switch filepath.Base(cleanBinary) {
	case "vrooli", "vrooli-api", "vrooli-agent-launcher", "vrooli-policy-runner":
		return fmt.Errorf("refuse to uninstall protected control-plane CLI %q", filepath.Base(cleanBinary))
	}
	ledger, err := artifactledger.New(controller.Home)
	if err != nil {
		return fmt.Errorf("prepare native CLI removal ledger: %w", err)
	}
	for _, artifact := range []struct {
		kind string
		path string
	}{
		{kind: "binary", path: cleanBinary},
		{kind: "manifest", path: cliutil.InstalledManifestPath(cleanBinary)},
		{kind: "build-metadata", path: cliutil.InstalledBuildMetadataPath(cleanBinary)},
	} {
		err := ledger.Guard(artifactledger.Removal{
			Path:      artifact.path,
			Subject:   cleanBinary,
			Kind:      artifact.kind,
			Component: "resources.uninstallNativeCLI",
			Predicate: "operator requested uninstall of the named resource CLI",
		}, func() error { return os.Remove(artifact.path) })
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("uninstall native CLI %s: %w", manifest.Name, err)
		}
	}
	if err := artifactlease.Remove(cleanBinary); err != nil {
		return fmt.Errorf("remove native CLI ownership record %s: %w", manifest.Name, err)
	}
	return nil
}

func runInstallCommand(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	if manifest.ManagedService != nil && len(manifest.ManagedService.DataArtifacts) > 0 {
		if err := ensureManagedServiceDataArtifacts(ctx, controller, manifest); err != nil {
			return err
		}
	}
	if manifest.CLI != nil && manifest.CLI.Enabled {
		if sourceBuild := manifest.CLI.SourceBuild; sourceBuild != nil {
			return runSourceBuild(ctx, controller, manifest, sourceBuild)
		}
	}
	command := manifest.Install.Command
	if len(command) == 0 {
		command = manifest.Install.Platforms[manifestpkg.CurrentPlatform()]
	}
	if len(command) == 0 {
		return nil
	}

	stderrTail := newTailBuffer(16 << 10)
	cmd := shell.Command(shell.Spec{
		Name:   command[0],
		Args:   command[1:],
		Dir:    controller.Root,
		Env:    resourceEnvForResource(controller.Root, controller.Home, manifest.Name),
		Stdout: io.Discard,
		Stderr: stderrTail,
	})
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err := cmd.Start(); err != nil {
		return err
	}
	waitErr := cmd.Wait()
	if runCtx.Err() != nil {
		return runCtx.Err()
	}
	if waitErr != nil {
		if detail := strings.TrimSpace(stderrTail.String()); detail != "" {
			return fmt.Errorf("%w\n%s", waitErr, detail)
		}
	}
	return waitErr
}

// runSourceBuild is the Go-native source-checkout installation path. It is
// intentionally separate from deployed resource distribution: developers with
// a checkout need Go to build changed source, while desktop bundles consume
// signed prebuilt artifacts and never reach this code.
func runSourceBuild(ctx context.Context, controller *Controller, manifest ResourceManifest, sourceBuild *scenario.CLISourceBuildConfig) error {
	if sourceBuild.Kind != "go_module" {
		return fmt.Errorf("unsupported resource source build kind %q", sourceBuild.Kind)
	}
	resourceRoot := filepath.Join(controller.Root, "resources", manifest.Name)
	moduleDir := filepath.Join(resourceRoot, filepath.FromSlash(manifest.CLI.Adapter.ModuleDir))
	manifestPath := filepath.Join(resourceRoot, "resource.json")
	installerDir := filepath.Join(controller.Root, "packages", "cli-core")
	runtimeHome, err := repocontract.VrooliUserRoot(controller.Home)
	if err != nil {
		return fmt.Errorf("resolve runtime home: %w", err)
	}
	installDir := filepath.Join(runtimeHome, "bin")
	spec := cliutil.CanonicalResourceGoModuleFreshnessSpec(resourceRoot, moduleDir, manifest.CLI.Command, manifest.CLI.Freshness.Inputs)
	args := cliutil.GoModuleInstallerArgs(moduleDir, manifestPath, manifest.CLI.Command, installDir, spec)
	stderrTail := newTailBuffer(16 << 10)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = installerDir
	cmd.Env = resourceEnvForResource(controller.Root, controller.Home, manifest.Name)
	cmd.Stdout = io.Discard
	cmd.Stderr = stderrTail
	if err := runSourceBuildCommandFn(cmd); err != nil {
		if detail := strings.TrimSpace(stderrTail.String()); detail != "" {
			return fmt.Errorf("source-build resource CLI: %w\n%s", err, detail)
		}
		return fmt.Errorf("source-build resource CLI: %w", err)
	}
	return nil
}

// tailBuffer is an io.Writer that retains only the last max bytes written —
// enough to surface the tail of a failing command's stderr (where the error
// reason typically lives) without buffering unbounded output.
type tailBuffer struct {
	max int
	buf []byte
}

func newTailBuffer(max int) *tailBuffer { return &tailBuffer{max: max} }

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = b.buf[len(b.buf)-b.max:]
	}
	return len(p), nil
}

func (b *tailBuffer) String() string { return string(b.buf) }
