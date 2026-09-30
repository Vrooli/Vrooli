package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"

	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
)

type ValidationIssue struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type ResourceValidationItem struct {
	Name         string            `json:"name"`
	ManifestPath string            `json:"manifest_path,omitempty"`
	Driver       string            `json:"driver,omitempty"`
	Issues       []ValidationIssue `json:"issues,omitempty"`
}

type ResourceValidationReport struct {
	Count  int                      `json:"count"`
	Passed bool                     `json:"passed"`
	Items  []ResourceValidationItem `json:"items,omitempty"`
	Issues []ValidationIssue        `json:"issues,omitempty"`
}

func (c *Controller) ValidateResources(name string) (ResourceValidationReport, error) {
	report := ResourceValidationReport{
		Passed: true,
		Items:  []ResourceValidationItem{},
		Issues: []ValidationIssue{},
	}
	if strings.TrimSpace(name) != "" {
		path := filepath.Join(c.Root, "resources", name, "resource.json")
		if data, readErr := os.ReadFile(path); readErr == nil {
			var fields map[string]json.RawMessage
			if json.Unmarshal(data, &fields) == nil {
				if raw, present := fields["driver"]; present {
					var driver string
					if json.Unmarshal(raw, &driver) == nil && !slices.Contains([]string{"managed-service", "external-cli", "cloud-api", "native-cli"}, strings.TrimSpace(driver)) {
						var runtimeFields map[string]json.RawMessage
						_ = json.Unmarshal(data, &runtimeFields)
						if runtime, hasRuntime := runtimeFields["runtime"]; !hasRuntime || string(runtime) == "{}" {
							return ResourceValidationReport{Count: 1, Passed: false, Items: []ResourceValidationItem{{Name: name, ManifestPath: path, Driver: driver, Issues: []ValidationIssue{{Severity: "error", Message: fmt.Sprintf("unknown_driver: driver %q is not in the managed resource archetype enum", driver)}}}}}, nil
						}
					}
				}
			}
		}
	}

	items, err := c.Discover()
	if err != nil {
		message := err.Error()
		if strings.Contains(message, "driver ") && strings.Contains(message, " is invalid") {
			return ResourceValidationReport{
				Count:  1,
				Passed: false,
				Items:  []ResourceValidationItem{{Name: name, ManifestPath: filepath.Join(c.Root, "resources", name, "resource.json"), Issues: []ValidationIssue{{Severity: "error", Message: "unknown_driver: " + message}}}},
			}, nil
		}
		return ResourceValidationReport{}, err
	}
	if strings.TrimSpace(name) != "" {
		filtered := items[:0]
		for _, item := range items {
			if item.Name == name {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	report.Count = len(items)
	rootContract, err := scenario.LoadServiceManifest(filepath.Join(c.Root, ".vrooli", "service.json"))
	if err != nil {
		return ResourceValidationReport{}, err
	}

	portOwners := map[string]string{}
	for _, item := range items {
		if strings.TrimSpace(item.ManifestPath) == "" {
			continue
		}
		manifest, err := manifestpkg.Load(item.ManifestPath)
		if err != nil {
			return ResourceValidationReport{}, err
		}
		entry := ResourceValidationItem{
			Name:         item.Name,
			ManifestPath: item.ManifestPath,
			Driver:       manifest.Driver,
			Issues:       []ValidationIssue{},
		}
		for _, requiredFile := range []string{"resource.json", "README.md"} {
			path := filepath.Join(c.Root, "resources", item.Name, requiredFile)
			if info, statErr := os.Stat(path); statErr != nil || info.IsDir() {
				entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "missing_required_file: " + requiredFile})
			}
		}
		resourceDir := filepath.Join(c.Root, "resources", item.Name)
		cliDir := filepath.Join(resourceDir, "cli")
		if info, statErr := os.Stat(cliDir); statErr != nil || !info.IsDir() {
			if manifest.CLI != nil && manifest.CLI.Enabled {
				entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "cli_declared_without_module: cli.enabled requires a cli/ directory"})
			} else {
				entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "missing_required_dir: cli"})
			}
		} else if entries, readErr := os.ReadDir(cliDir); readErr == nil && len(entries) == 0 {
			entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "empty_required_dir: cli"})
		}
		_, hasTestDir := statDir(resourceDir, "test")
		_, hasTestsDir := statDir(resourceDir, "tests")
		if hasTestDir && hasTestsDir {
			entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "conflicting_test_dir: use either test/ or tests/, not both"})
		}
		if data, readErr := os.ReadFile(item.ManifestPath); readErr == nil {
			var fields map[string]json.RawMessage
			if json.Unmarshal(data, &fields) == nil {
				entry.Issues = append(entry.Issues, storageSurfaceIssues(fields)...)
				if _, present := fields["template"]; present {
					entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "unknown_manifest_field: template is not a resource archetype field"})
				}
			}
		}
		if _, registeredInRoot := rootContract.Dependencies.Resources[item.Name]; !registeredInRoot {
			entry.Issues = append(entry.Issues, ValidationIssue{
				Severity: "error",
				Message:  "resource_absent_from_contract: resource is not declared in .vrooli/service.json",
			})
		}
		entry.Issues = append(entry.Issues, composedAcquisitionIssues(resourceDir, manifest)...)
		entry.Issues = append(entry.Issues, artifactDigestAgreementIssues(manifest)...)
		for _, issue := range resourceenv.ValidateResourceManifest(c.Root, manifest) {
			entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: issue})
		}
		if usesSingleManifestContract(manifest) {
			for _, relPath := range []string{
				"config/schema.json",
				"config/runtime.json",
				"config/defaults.json",
				"config/exports.sh",
			} {
				path := filepath.Join(c.Root, "resources", item.Name, filepath.FromSlash(relPath))
				if _, err := os.Stat(path); err == nil {
					entry.Issues = append(entry.Issues, ValidationIssue{
						Severity: "error",
						Message:  fmt.Sprintf("deprecated file retained for single-manifest resource: %s", relPath),
					})
				}
			}
		}
		for _, port := range manifest.Ports {
			hostPort := port.Host
			if hostPort <= 0 {
				hostPort = port.Container
			}
			if hostPort <= 0 {
				continue
			}
			key := resourcePortOwnerKey(port, hostPort)
			if owner, exists := portOwners[key]; exists && owner != item.Name {
				message := fmt.Sprintf("host port %s overlaps with resource %s", key, owner)
				entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: message})
				report.Issues = append(report.Issues, ValidationIssue{Severity: "error", Message: fmt.Sprintf("%s: %s", item.Name, message)})
			} else {
				portOwners[key] = item.Name
			}
		}
		if len(entry.Issues) > 0 {
			report.Passed = false
		}
		report.Items = append(report.Items, entry)
	}
	sort.Slice(report.Items, func(i, j int) bool {
		return report.Items[i].Name < report.Items[j].Name
	})
	if strings.TrimSpace(name) != "" && len(report.Items) == 0 {
		return ResourceValidationReport{}, fmt.Errorf("resource %s not found", name)
	}
	return report, nil
}

func resourcePortOwnerKey(port manifestpkg.ResourcePort, hostPort int) string {
	hostIP := strings.TrimSpace(port.HostIP)
	if hostIP == "" {
		hostIP = "*"
	}
	protocol := strings.ToLower(strings.TrimSpace(port.Protocol))
	if protocol == "" {
		protocol = "tcp"
	}
	return fmt.Sprintf("%s:%d/%s", hostIP, hostPort, protocol)
}

func statDir(parent, name string) (os.FileInfo, bool) {
	info, err := os.Stat(filepath.Join(parent, name))
	return info, err == nil && info.IsDir()
}

func usesSingleManifestContract(manifest ResourceManifest) bool {
	version := strings.TrimSpace(manifest.TemplateVersion)
	return version != "" && version != "1"
}

type ScenarioEnvValidationReport struct {
	Scenario        string                       `json:"scenario"`
	Values          map[string]string            `json:"values"`
	Issues          []ValidationIssue            `json:"issues,omitempty"`
	ResourceReports []resourceenv.ResourceReport `json:"resource_reports,omitempty"`
	Passed          bool                         `json:"passed"`
}

func (c *Controller) ValidateScenarioEnvironment(name string) (ScenarioEnvValidationReport, error) {
	item, err := scenario.Load(c.Root, name, scenario.SandboxEnv{})
	if err != nil {
		return ScenarioEnvValidationReport{}, err
	}
	resolution, issues, err := resourceenv.ValidateScenario(c.Root, c.Home, item.Slug, item.Manifest)
	if err != nil {
		return ScenarioEnvValidationReport{}, err
	}
	report := ScenarioEnvValidationReport{
		Scenario:        item.Slug,
		Values:          resolution.Values,
		ResourceReports: resolution.Resources,
		Passed:          len(issues) == 0,
	}
	for _, warning := range resolution.Warnings {
		report.Issues = append(report.Issues, ValidationIssue{Severity: "warning", Message: warning})
	}
	for _, issue := range issues {
		report.Issues = append(report.Issues, ValidationIssue{Severity: "error", Message: issue})
	}
	return report, nil
}

// storageSurfaceIssues enforces the one requirement the shared storageSurface
// definition states and this validator never read: a declared storage block
// carries an entries map. resource.json and a scenario's .vrooli/service.json
// both $ref common.schema.json#/definitions/storageSurface, so a resource that
// invents its own shape is describing its disk footprint in a vocabulary no
// consumer can read — which is how a resource ends up installing outside its
// declared root with every gate green.
func storageSurfaceIssues(fields map[string]json.RawMessage) []ValidationIssue {
	raw, present := fields["storage"]
	if !present {
		return nil
	}
	var surface struct {
		Entries map[string]json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(raw, &surface); err != nil {
		return []ValidationIssue{{Severity: "error", Message: "invalid_storage_surface: storage must be an object matching common.schema.json#/definitions/storageSurface"}}
	}
	if surface.Entries == nil {
		return []ValidationIssue{{Severity: "error", Message: "invalid_storage_surface: storage.entries is required by common.schema.json#/definitions/storageSurface"}}
	}
	return nil
}

// composedAcquisitionIssues enforces that a composed artifact can actually be
// rebuilt into the same bytes it declares.
//
// A `composed` acquisition pins the OUTPUT tree with artifact_sha256, but the
// output is only as reproducible as its inputs. kyutai-stt shipped a
// python-wheels step whose "lockfile" held eight direct requirements with no
// hashes, while the installer runs `--no-deps` — so the staged tree was neither
// the declared closure nor reproducible, its artifact_sha256 became unreachable
// the moment the resolver moved, and the resource failed verification with no
// path back. It read as "installed: no" while its server was healthy on the
// GPU, and every dictation session silently fell back to a CPU batch engine.
//
// The two rules below are what separated kokoro (which survived the same
// window) from kyutai-stt:
//
//   - a hash-pinned lockfile, so `--require-hashes` makes the install exact;
//   - an explicit index, so a CUDA build cannot silently become a CPU build
//     that satisfies the same version constraint.
func composedAcquisitionIssues(resourceDir string, manifest ResourceManifest) []ValidationIssue {
	issues := []ValidationIssue{}
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return issues
	}
	if !strings.EqualFold(strings.TrimSpace(manifest.ManagedService.Acquisition.Kind), "composed") {
		return issues
	}
	seen := map[string]struct{}{}
	for _, target := range manifest.ManagedService.Acquisition.Targets {
		for _, step := range target.Compose {
			if !strings.EqualFold(strings.TrimSpace(step.Kind), "python-wheels") {
				continue
			}
			lock := strings.TrimSpace(step.Lockfile)
			if lock == "" {
				continue
			}
			if _, done := seen[lock]; done {
				continue
			}
			seen[lock] = struct{}{}
			data, err := os.ReadFile(filepath.Join(resourceDir, filepath.FromSlash(lock)))
			if err != nil {
				issues = append(issues, ValidationIssue{
					Severity: "error",
					Message:  "unreadable_wheel_lockfile: " + lock + " is declared by a composed acquisition but cannot be read",
				})
				continue
			}
			if !strings.Contains(string(data), "--hash=sha256:") {
				issues = append(issues, ValidationIssue{
					Severity: "error",
					Message: "unpinned_wheel_lockfile: " + lock + " has no --hash entries, so the composed tree cannot be" +
						" reproduced and its artifact_sha256 cannot stay true; regenerate with `uv pip compile --generate-hashes`",
				})
			}
			// The index only carries risk where an accelerated build exists to
			// lose. For a resource that declares no acceleration there is no CPU
			// variant to silently fall back to, and warning anyway would make
			// the signal noise.
			if strings.TrimSpace(step.IndexURL) == "" && declaresAcceleration(manifest) {
				issues = append(issues, ValidationIssue{
					Severity: "warning",
					Message: "implicit_wheel_index: " + lock + " is installed from the default index while this resource" +
						" declares accelerated backends, so an accelerated wheel may silently resolve to a CPU one;" +
						" declare index_url explicitly",
				})
			}
		}
	}
	return issues
}

// declaresAcceleration reports whether a manifest claims any backend other than
// plain CPU.
func declaresAcceleration(manifest ResourceManifest) bool {
	declaration := manifest.EffectiveAcceleration()
	if declaration == nil {
		return false
	}
	for _, backend := range declaration.Backends {
		if !strings.EqualFold(strings.TrimSpace(backend), "cpu") && strings.TrimSpace(backend) != "" {
			return true
		}
	}
	return false
}

// artifactDigestAgreementIssues catches a stale copy of a digest that is
// declared in two places.
//
// A managed-service manifest states the artifact digest twice: once per
// platform in managed_service.artifact.sha256_by_platform, and once per
// acquisition target in artifact_sha256. Launch verification reads the TARGET
// digest, so an edit that updates only the platform map changes nothing and an
// edit that updates only the target leaves a stale claim behind. Neither is
// reported anywhere.
//
// The two are not required to be equal: a platform may carry several
// fact-predicated targets (kokoro declares a CUDA and a CPU tree for
// linux-amd64) and the platform map holds only one digest. The rule that holds
// in every case is weaker and still catches the staleness: the platform digest
// must match SOME target that can resolve on that platform.
func artifactDigestAgreementIssues(manifest ResourceManifest) []ValidationIssue {
	issues := []ValidationIssue{}
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return issues
	}
	byPlatform := manifest.ManagedService.Artifact.SHA256ByPlatform
	if len(byPlatform) == 0 {
		return issues
	}
	platforms := make([]string, 0, len(byPlatform))
	for platform := range byPlatform {
		platforms = append(platforms, platform)
	}
	slices.Sort(platforms)
	for _, platform := range platforms {
		declared := strings.ToLower(strings.TrimSpace(byPlatform[platform]))
		if declared == "" {
			continue
		}
		candidates := []string{}
		for _, target := range manifest.ManagedService.Acquisition.Targets {
			digest := strings.ToLower(strings.TrimSpace(target.ArtifactSHA256))
			if digest == "" || !targetCoversPlatform(target, platform) {
				continue
			}
			candidates = append(candidates, digest)
		}
		if len(candidates) == 0 {
			continue
		}
		if !slices.Contains(candidates, declared) {
			issues = append(issues, ValidationIssue{
				Severity: "error",
				Message: "artifact_digest_disagreement: sha256_by_platform[" + platform + "] declares " + shortDigest(declared) +
					" but no acquisition target for that platform declares it; launch verification reads the target digest," +
					" so this copy is stale and silently does nothing",
			})
		}
	}
	return issues
}

// targetCoversPlatform reports whether a target's os/arch predicates can select
// on the given "<os>-<arch>" platform. Non-platform predicates (accelerator
// facts) are host-dependent and deliberately ignored here.
func targetCoversPlatform(target binaryfetch.AcquisitionTarget, platform string) bool {
	os, arch, found := strings.Cut(platform, "-")
	if !found {
		return false
	}
	if declared, ok := target.When["os"]; ok && !strings.EqualFold(strings.TrimSpace(declared), os) {
		return false
	}
	if declared, ok := target.When["arch"]; ok && !strings.EqualFold(strings.TrimSpace(declared), arch) {
		return false
	}
	return true
}
