// Package system provides CLI commands for system operations.
package system

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/emptypb"
)

const appName = "scenario-to-desktop"

type Commands struct {
	deps       support.Dependencies
	records    recordsRPC
	system     systemRPC
	operations operationsRPC
}

type systemRPC interface {
	ListTemplates(context.Context, *connect.Request[domainv1.ListTemplatesRequest]) (*connect.Response[domainv1.ListTemplatesResponse], error)
	GetTemplate(context.Context, *connect.Request[domainv1.GetTemplateRequest]) (*connect.Response[domainv1.TemplateConfigResponse], error)
	CheckWine(context.Context, *connect.Request[domainv1.CheckWineRequest]) (*connect.Response[domainv1.WineCheckResponse], error)
	InstallWine(context.Context, *connect.Request[domainv1.InstallWineRequest]) (*connect.Response[domainv1.WineInstallResponse], error)
	GetWineInstallStatus(context.Context, *connect.Request[domainv1.GetWineInstallStatusRequest]) (*connect.Response[domainv1.WineInstallStatusResponse], error)
}

type operationsRPC interface {
	ListDesktopScenarioStatus(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.DesktopScenarioStatusResponse], error)
}

type recordsRPC interface {
	ListDesktopRecords(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.DesktopRecordsResponse], error)
	MoveDesktopRecord(context.Context, *connect.Request[domainv1.MoveDesktopRecordRequest]) (*connect.Response[domainv1.MoveDesktopRecordResponse], error)
	DeleteDesktopScenario(context.Context, *connect.Request[domainv1.DeleteDesktopScenarioRequest]) (*connect.Response[domainv1.DeleteDesktopScenarioResponse], error)
}

func New(deps support.Dependencies) *Commands {
	app := deps.Core()
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	return &Commands{deps: deps, records: domainconnect.NewDesktopRecordsServiceClient(httpClient, baseURL), system: domainconnect.NewSystemServiceClient(httpClient, baseURL), operations: domainconnect.NewOperationsServiceClient(httpClient, baseURL)}
}

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	cmds := New(deps)
	return []cliapp.CommandGroup{
		{Title: "Templates", Commands: []cliapp.Command{
			(cliapp.Command{Name: "templates", NeedsAPI: true, Description: "List available desktop templates"}).WithPrimitive(cmds.templatesListPrimitive()),
			(cliapp.Command{Name: "template", NeedsAPI: true, Description: "Get template details: template <type>", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "type", Required: true, Description: "Template type"}}}}).WithPrimitive(cmds.templateGetPrimitive()),
		}},
		{Title: "Records", Commands: []cliapp.Command{
			(cliapp.Command{Name: "records", NeedsAPI: true, Description: "List desktop generation records"}).WithPrimitive(cmds.recordsListPrimitive()),
			(cliapp.Command{Name: "records-move", NeedsAPI: true, Description: "Move desktop wrapper: records-move <id> [--target <path>]", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Desktop record ID"}}, Flags: []cliapp.Flag{{Name: "target", Default: "destination", Description: "Move target"}, {Name: "path", Description: "Custom destination path"}}}}).WithPrimitive(cmds.recordsMovePrimitive()),
			(cliapp.Command{Name: "records-delete", NeedsAPI: true, Description: "Delete desktop app: records-delete <scenario>", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}}}}).WithPrimitive(cmds.recordsDeletePrimitive()),
		}},
		{Title: "Download", Commands: []cliapp.Command{
			(cliapp.Command{Name: "download", NeedsAPI: true, Description: "Download built package: download <scenario> <platform> [--output <path>]", Args: cliapp.ArgSchema{
				Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}, {Name: "platform", Required: true, Description: "Target platform (win, mac, or linux)"}},
				Flags:       []cliapp.Flag{{Name: "output", Description: "Output file path (defaults to the current directory)"}},
			}}).WithPrimitive(cmds.downloadPrimitive()),
		}},
		{Title: "Scenarios", Commands: []cliapp.Command{
			(cliapp.Command{Name: "desktop-status", NeedsAPI: true, Description: "List desktop build status and artifacts", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "name", Description: "Filter by scenario name"}}}}).WithPrimitive(cmds.desktopStatusPrimitive()),
		}},
		{Title: "Validation", Commands: []cliapp.Command{
			(cliapp.Command{Name: "validate-artifact", Description: "Validate a placed desktop artifact and write a bounded evidence bundle", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
				{Name: "artifact", Required: true, Description: "Placed artifact path"}, {Name: "artifact-digest", Required: true, Description: "Expected sha256:<hex> digest"}, {Name: "scenario", Required: true, Description: "Scenario name"}, {Name: "journey", Required: true, Description: "Journey identifier"}, {Name: "profile", Default: "normal", Description: "Validation environment profile"}, {Name: "evidence-output", Required: true, Description: "Evidence archive output path"},
			}}}).WithPrimitive(cliapp.Action(func(ctx cliapp.OperationContext) (string, error) {
				return "", validateArtifact(ctx.Flag("artifact"), ctx.Flag("artifact-digest"), ctx.Flag("scenario"), ctx.Flag("journey"), ctx.Flag("profile"), ctx.Flag("evidence-output"))
			}, func(_ cliapp.OperationContext, _ string) cliapp.MutationReport {
				return cliapp.MutationReport{Result: []string{"Validation evidence bundle written"}}
			})),
		}},
	}
}

// validateArtifact performs the target-side, caller-verifiable portion of the
// remote contract. Digest validation happens before launch. The command is
// intentionally batch-shaped: it launches the packaged application in the
// target session, records the result, and exits; live desktop streaming is not
// part of the Bridge contract.
func validateArtifact(artifact, expected, scenario, journey, profile, output string) error {
	artifact = strings.TrimSpace(artifact)
	data, err := os.ReadFile(artifact)
	if err != nil {
		return fmt.Errorf("artifact_read_failed: %w", err)
	}
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if !strings.EqualFold(strings.TrimSpace(expected), digest) {
		return fmt.Errorf("artifact_digest_mismatch: want %s got %s", expected, digest)
	}
	launch := launchArtifact(artifact)
	status := "failed"
	if launch.Err == nil {
		status = "passed"
	}
	bundle := map[string]any{
		"journey-sidecar.json":   map[string]any{"scenario": scenario, "journey": journey, "profile": profile, "status": status, "steps": []string{"launch"}},
		"launch-trace.json":      map[string]any{"artifact": filepath.Base(artifact), "digest": digest, "status": status, "command": launch.Command, "exit_code": launch.ExitCode, "duration_ms": launch.Duration.Milliseconds(), "output": launch.Output, "error": errorString(launch.Err)},
		"machine-assertion.json": map[string]any{"status": status, "os": runtime.GOOS, "architecture": runtime.GOARCH, "target_owned": true, "observed_at": time.Now().UTC().Format(time.RFC3339Nano)},
	}
	if err := writeValidationBundle(output, bundle); err != nil {
		return err
	}
	if launch.Err != nil {
		return fmt.Errorf("validation_not_passed: %w", launch.Err)
	}
	return nil
}

type launchResult struct {
	Command  string
	Output   string
	ExitCode int
	Duration time.Duration
	Err      error
}

func launchArtifact(artifact string) launchResult {
	if runtime.GOOS == "darwin" && strings.EqualFold(filepath.Ext(artifact), ".zip") {
		return launchDarwinZip(artifact)
	}
	if runtime.GOOS == "darwin" && strings.EqualFold(filepath.Ext(artifact), ".dmg") {
		return launchDarwinDiskImage(artifact)
	}
	command, args := artifactCommand(artifact)
	if command == "" {
		return launchResult{Err: fmt.Errorf("unsupported desktop artifact %q", filepath.Ext(artifact))}
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = filepath.Dir(artifact)
	output, err := cmd.CombinedOutput()
	result := launchResult{Command: strings.Join(append([]string{command}, args...), " "), Output: truncate(string(output), 8192), Duration: time.Since(started), Err: err}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		result.ExitCode = -1
	}
	if ctx.Err() != nil {
		result.Err = fmt.Errorf("launch timed out after %s", 45*time.Second)
	}
	return result
}

// launchDarwinZip validates and extracts a packaged macOS zip into a private
// temporary directory, then launches the contained app. Electron's mac build
// commonly emits a zip rather than a dmg, so treating the archive as a first-
// class target keeps artifact validation aligned with the pipeline output.
func launchDarwinZip(artifact string) launchResult {
	started := time.Now()
	root, err := os.MkdirTemp("", "scenario-to-desktop-maczip-")
	if err != nil {
		return launchResult{Command: "unzip + open -W", Duration: time.Since(started), Err: err}
	}
	defer os.RemoveAll(root)
	archive, err := zip.OpenReader(artifact)
	if err != nil {
		return launchResult{Command: "unzip + open -W", Duration: time.Since(started), Err: err}
	}
	defer archive.Close()
	for _, entry := range archive.File {
		target := filepath.Join(root, filepath.FromSlash(entry.Name))
		if target != root && !strings.HasPrefix(target, root+string(filepath.Separator)) {
			return launchResult{Command: "unzip + open -W", Duration: time.Since(started), Err: fmt.Errorf("archive entry escapes extraction root: %q", entry.Name)}
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return launchResult{Command: "unzip + open -W", Duration: time.Since(started), Err: err}
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return launchResult{Command: "unzip + open -W", Duration: time.Since(started), Err: err}
		}
		in, err := entry.Open()
		if err != nil {
			return launchResult{Command: "unzip + open -W", Duration: time.Since(started), Err: err}
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o750)
		if err == nil {
			_, err = io.Copy(out, in)
		}
		_ = in.Close()
		_ = out.Close()
		if err != nil {
			return launchResult{Command: "unzip + open -W", Duration: time.Since(started), Err: err}
		}
	}
	apps, err := filepath.Glob(filepath.Join(root, "*.app"))
	if err != nil || len(apps) == 0 {
		return launchResult{Command: "unzip + open -W", Duration: time.Since(started), Err: fmt.Errorf("archive contains no top-level .app")}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	prepareOutput, prepareErr := prepareDarwinApp(ctx, apps[0])
	if prepareErr != nil {
		return launchResult{Command: "xattr + codesign + open -W", Output: truncate(prepareOutput, 8192), Duration: time.Since(started), Err: prepareErr, ExitCode: 1}
	}
	cmd := exec.CommandContext(ctx, "open", "-W", apps[0])
	output, launchErr := cmd.CombinedOutput()
	result := launchResult{Command: "xattr + codesign + open -W", Output: truncate(prepareOutput+string(output), 8192), Duration: time.Since(started), Err: launchErr}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		result.ExitCode = -1
	}
	if ctx.Err() != nil {
		result.Err = fmt.Errorf("launch timed out after %s", 45*time.Second)
	}
	result.Duration = time.Since(started)
	return result
}

// prepareDarwinApp applies only artifact-scoped launch mitigations. It never
// changes Gatekeeper policy and only touches the extracted app bundle supplied
// by the validator.
func prepareDarwinApp(ctx context.Context, app string) (string, error) {
	var output strings.Builder
	quarantine := exec.CommandContext(ctx, "xattr", "-p", "com.apple.quarantine", app)
	if quarantineOutput, err := quarantine.CombinedOutput(); err == nil {
		output.WriteString("quarantine: ")
		output.Write(quarantineOutput)
		remove := exec.CommandContext(ctx, "xattr", "-d", "com.apple.quarantine", app)
		removeOutput, removeErr := remove.CombinedOutput()
		output.WriteString("quarantine-remove: ")
		output.Write(removeOutput)
		if removeErr != nil {
			return output.String(), fmt.Errorf("remove artifact quarantine: %w", removeErr)
		}
	}

	sign := exec.CommandContext(ctx, "codesign", "--force", "--deep", "--sign", "-", app)
	signOutput, signErr := sign.CombinedOutput()
	output.WriteString("codesign: ")
	output.Write(signOutput)
	if signErr != nil {
		return output.String(), fmt.Errorf("ad-hoc sign artifact: %w", signErr)
	}
	if ctx.Err() != nil {
		return output.String(), ctx.Err()
	}
	return output.String(), nil
}

// launchDarwinDiskImage mounts a disk image read-only, finds the packaged app,
// and waits for that app to exit. Opening a .dmg in Finder is not validation:
// it only proves that Finder accepted the container.
func launchDarwinDiskImage(artifact string) launchResult {
	started := time.Now()
	mountpoint, err := os.MkdirTemp("", "scenario-to-desktop-dmg-")
	if err != nil {
		return launchResult{Command: "hdiutil attach + cp + xattr + codesign + open -W", Duration: time.Since(started), Err: err}
	}
	defer os.RemoveAll(mountpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	attach := exec.CommandContext(ctx, "hdiutil", "attach", "-nobrowse", "-readonly", "-mountpoint", mountpoint, artifact)
	attachOutput, attachErr := attach.CombinedOutput()
	result := launchResult{Command: "hdiutil attach + cp + xattr + codesign + open -W", Output: truncate(string(attachOutput), 8192), Duration: time.Since(started), Err: attachErr}
	if attachErr != nil {
		return result
	}
	defer func() {
		_ = exec.Command("hdiutil", "detach", mountpoint, "-force").Run()
	}()

	apps, err := filepath.Glob(filepath.Join(mountpoint, "*.app"))
	if err != nil || len(apps) == 0 {
		result.Err = fmt.Errorf("disk image contains no top-level .app")
		result.Duration = time.Since(started)
		return result
	}
	stagedRoot, stageErr := os.MkdirTemp("", "scenario-to-desktop-dmg-app-")
	if stageErr != nil {
		result.Err = stageErr
		result.Duration = time.Since(started)
		return result
	}
	defer os.RemoveAll(stagedRoot)
	stagedApp := filepath.Join(stagedRoot, filepath.Base(apps[0]))
	copyApp := exec.CommandContext(ctx, "cp", "-R", apps[0], stagedApp)
	copyOutput, copyErr := copyApp.CombinedOutput()
	result.Output = truncate(result.Output+string(copyOutput), 8192)
	if copyErr != nil {
		result.Err = fmt.Errorf("stage disk-image app: %w", copyErr)
		result.Duration = time.Since(started)
		return result
	}
	prepareOutput, prepareErr := prepareDarwinApp(ctx, stagedApp)
	result.Output = truncate(result.Output+prepareOutput, 8192)
	if prepareErr != nil {
		result.Err = prepareErr
		result.Duration = time.Since(started)
		return result
	}
	open := exec.CommandContext(ctx, "open", "-W", stagedApp)
	openOutput, openErr := open.CombinedOutput()
	result.Output = truncate(result.Output+string(openOutput), 8192)
	result.Err = openErr
	if open.ProcessState != nil {
		result.ExitCode = open.ProcessState.ExitCode()
	} else {
		result.ExitCode = -1
	}
	if ctx.Err() != nil {
		result.Err = fmt.Errorf("launch timed out after %s", 45*time.Second)
	}
	result.Duration = time.Since(started)
	return result
}

func artifactCommand(artifact string) (string, []string) {
	ext := strings.ToLower(filepath.Ext(artifact))
	switch runtime.GOOS {
	case "darwin":
		switch ext {
		case ".app":
			return "open", []string{"-W", artifact}
		case ".dmg":
			return "open", []string{"-W", artifact}
		}
	case "windows":
		if ext == ".exe" {
			return artifact, nil
		}
	default:
		if ext == ".appimage" || ext == ".bin" {
			return artifact, []string{"--version"}
		}
	}
	return "", nil
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "…"
}

func writeValidationBundle(output string, members map[string]any) error {
	if strings.TrimSpace(output) == "" {
		return fmt.Errorf("evidence_output_required")
	}
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("evidence_archive_create_failed: %w", err)
	}
	defer file.Close()
	gz := gzip.NewWriter(file)
	defer gz.Close()
	tarWriter := tar.NewWriter(gz)
	defer tarWriter.Close()
	for name, value := range members {
		payload, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return fmt.Errorf("evidence_member_encode_failed: %w", marshalErr)
		}
		header := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(payload))}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := io.Copy(tarWriter, strings.NewReader(string(payload))); err != nil {
			return err
		}
	}
	return nil
}

func WineRegister(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{Name: "wine", Description: "Wine for Windows builds on Linux (run 'wine help' for details)", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "check", Description: "Check Wine installation status"}).WithPrimitive(cmds.wineCheckPrimitive()),
		(cliapp.Command{Name: "install", Description: "Install Wine: install --method <flatpak|appimage>", Args: cliapp.ArgSchema{
			Flags: []cliapp.Flag{{Name: "method", Required: true, Description: "Installation method", Values: []string{"flatpak", "flatpak-auto", "appimage"}}},
		}}).WithPrimitive(cmds.wineInstallPrimitive()),
		(cliapp.Command{Name: "status", Description: "Get Wine install status: status <id>", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "install_id", Required: true, Description: "Installation ID"}}}}).WithPrimitive(cmds.wineStatusPrimitive()),
	}}
}

type desktopBuildArtifact struct {
	Platform, FileName         string
	SizeBytes                  int64
	RelativePath, AbsolutePath string
}
type desktopScenarioStatus struct {
	Name, DisplayName, Version string
	Built                      bool
	Platforms                  []string
	BuildArtifacts             []desktopBuildArtifact
}
type desktopStatusResponse struct {
	Scenarios []desktopScenarioStatus
	Stats     struct{ Total, WithDesktop, Built, WebOnly int }
}

func desktopStatusResponseFromProto(value *domainv1.DesktopScenarioStatusResponse) desktopStatusResponse {
	result := desktopStatusResponse{}
	if value == nil {
		return result
	}
	if stats := value.GetStats(); stats != nil {
		result.Stats.Total, result.Stats.WithDesktop, result.Stats.Built, result.Stats.WebOnly = int(stats.GetTotal()), int(stats.GetWithDesktop()), int(stats.GetBuilt()), int(stats.GetWebOnly())
	}
	for _, item := range value.GetScenarios() {
		scenario := desktopScenarioStatus{Name: item.GetName(), DisplayName: item.GetDisplayName(), Version: item.GetVersion(), Built: item.GetBuilt(), Platforms: item.GetPlatforms()}
		for _, artifact := range item.GetBuildArtifacts() {
			scenario.BuildArtifacts = append(scenario.BuildArtifacts, desktopBuildArtifact{Platform: artifact.GetPlatform(), FileName: artifact.GetFileName(), SizeBytes: artifact.GetSizeBytes(), RelativePath: artifact.GetRelativePath(), AbsolutePath: artifact.GetAbsolutePath()})
		}
		result.Scenarios = append(result.Scenarios, scenario)
	}
	return result
}

func filterScenariosByName(scenarios []desktopScenarioStatus, name string) []desktopScenarioStatus {
	filtered := make([]desktopScenarioStatus, 0)
	for _, scenario := range scenarios {
		if scenario.Name == name {
			filtered = append(filtered, scenario)
		}
	}
	return filtered
}

func desktopScenarioLine(scenario desktopScenarioStatus) string {
	name := scenario.Name
	if strings.TrimSpace(scenario.DisplayName) != "" {
		name = fmt.Sprintf("%s (%s)", name, scenario.DisplayName)
	}
	version := scenario.Version
	if strings.TrimSpace(version) == "" {
		version = "unknown"
	}
	status := "not built"
	if scenario.Built {
		status = "built"
	}
	parts := []string{fmt.Sprintf("%s v%s [%s]", name, version, status)}
	if len(scenario.Platforms) > 0 {
		parts = append(parts, fmt.Sprintf("platforms=%s", strings.Join(scenario.Platforms, ", ")))
	}
	if len(scenario.BuildArtifacts) > 0 {
		artifacts := make([]string, 0, len(scenario.BuildArtifacts))
		for _, artifact := range scenario.BuildArtifacts {
			fileName := artifact.FileName
			if fileName == "" {
				fileName = artifact.RelativePath
			}
			artifacts = append(artifacts, fmt.Sprintf("%s=%s (%d bytes)", artifact.Platform, fileName, artifact.SizeBytes))
		}
		parts = append(parts, "artifacts="+strings.Join(artifacts, "; "))
	}
	return strings.Join(parts, " | ")
}
