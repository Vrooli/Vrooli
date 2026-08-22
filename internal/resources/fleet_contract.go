package resources

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

var digestPattern = regexp.MustCompile(`@sha256:[0-9a-fA-F]{64}$`)

// CheckFleetContract verifies the repository-wide resource migration boundary.
// It is intentionally independent of runtime state so CI cannot regress to a
// shell-backed resource between live scenario runs.
func CheckFleetContract(root string) error {
	resourceRoot := filepath.Join(root, "resources")
	entries, err := os.ReadDir(resourceRoot)
	if err != nil {
		return fmt.Errorf("read resources: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		manifestPath := filepath.Join(resourceRoot, name, "resource.json")
		if _, statErr := os.Stat(manifestPath); os.IsNotExist(statErr) {
			// Active resource discovery is manifest-backed. Resource-local
			// prototypes may live beside published resources without becoming
			// fleet-contract subjects before they have an honest manifest.
			continue
		} else if statErr != nil {
			return fmt.Errorf("inspect resource %s manifest: %w", name, statErr)
		}
		names = append(names, name)
	}
	// The shell policy runs across the whole fleet before any per-resource
	// check so one run reports every violating file. Reporting only the first
	// hides the rest behind it, which is how the second violation in this
	// repository stayed invisible.
	if err := checkResourceShellPolicy(resourceRoot, names); err != nil {
		return err
	}
	for _, name := range names {
		manifestPath := filepath.Join(resourceRoot, name, "resource.json")
		manifest, err := manifestpkg.Load(manifestPath)
		if err != nil {
			return fmt.Errorf("load resource %s: %w", name, err)
		}
		if err := checkHealthKinds(name, manifest.HealthChecks); err != nil {
			return err
		}
		if err := checkManifestImage(manifestPath, manifest.Runtime.Image); err != nil {
			return err
		}
		if err := checkManifestCommands(name, manifest.Install.Command, manifest.Install.Platforms); err != nil {
			return err
		}
		if err := checkManagedArtifact(name, manifest); err != nil {
			return err
		}
		if err := checkLegacyAcceleratorSurfaces(name, manifestPath); err != nil {
			return err
		}
		if err := checkCapabilityContract(root, name); err != nil {
			return err
		}
	}
	if err := checkPlatformClaims(resourceRoot, names); err != nil {
		return err
	}
	if err := checkResourceImages(resourceRoot); err != nil {
		return err
	}
	return nil
}

// checkPlatformClaims holds every resource's platforms map against its declared
// acquisition, in both directions: a resource may not claim a platform it has
// no route to, and may not deny a platform it demonstrably serves.
//
// This is the rule that keeps the fix done. Postgres and Redis drifted into
// claiming one thing in the manifest and another in the platform documentation
// precisely because nothing compared a claim against its evidence. Like the
// shell policy, it collects every violation so one run shows the whole picture.
func checkPlatformClaims(resourceRoot string, names []string) error {
	violations := make([]string, 0)
	for _, name := range names {
		manifest, err := manifestpkg.Load(filepath.Join(resourceRoot, name, "resource.json"))
		if err != nil {
			return fmt.Errorf("load resource %s: %w", name, err)
		}
		service := manifest.ManagedService
		if service == nil || service.Acquisition == nil {
			// Without a declared acquisition there is no evidence to compare a
			// claim against; other checks own that case.
			continue
		}
		for _, osName := range []string{"linux", "macos", "windows"} {
			claim := strings.ToLower(strings.TrimSpace(platformClaim(manifest.Platforms, osName)))
			if claim == "" {
				continue
			}
			reachable := acquisitionReaches(service.Acquisition, osName)
			switch {
			case claim != "unsupported" && !reachable:
				violations = append(violations, fmt.Sprintf(
					"%s declares %s %q but no acquisition target serves that platform", name, osName, claim))
			case claim == "unsupported" && reachable:
				violations = append(violations, fmt.Sprintf(
					"%s declares %s unsupported but an acquisition target serves that platform", name, osName))
			}
		}
	}
	if len(violations) == 0 {
		return nil
	}
	sort.Strings(violations)
	return fmt.Errorf("resource platform claims are unsupported by acquisition: %s", strings.Join(violations, "; "))
}

// acquisitionReaches reports whether the acquisition declares any usable target
// for the operating system.
//
// This reads the declaration rather than resolving against this host's facts,
// because the question is whether a route exists at all, not whether the
// machine running the check happens to satisfy it. Reranker is the case that
// makes the difference: its usable Linux target is gated on
// gpu.cuda_compute >= 8.9, so a bare os/arch resolution finds only the
// explicitly unsupported CPU fallback and would report a real route as missing.
func acquisitionReaches(acquisition *binaryfetch.Acquisition, osName string) bool {
	factsOS := osName
	if factsOS == "macos" {
		factsOS = "darwin"
	}
	for _, target := range acquisition.Targets {
		if strings.TrimSpace(target.Unsupported) != "" {
			continue
		}
		declared := strings.ToLower(strings.TrimSpace(target.When["os"]))
		if declared == "" || declared == factsOS || declared == osName {
			return true
		}
	}
	return false
}

func platformClaim(platforms manifestpkg.ResourcePlatforms, osName string) string {
	switch osName {
	case "linux":
		return platforms.Linux
	case "macos":
		return platforms.MacOS
	case "windows":
		return platforms.Windows
	}
	return ""
}

func checkHealthKinds(name string, checks []manifestpkg.ResourceHealthCheck) error {
	if len(checks) == 0 {
		return fmt.Errorf("%s declares no health checks", name)
	}
	for index, check := range checks {
		if check.Kind != "readiness" && check.Kind != "liveness" {
			return fmt.Errorf("%s health_checks[%d] has invalid kind %q", name, index, check.Kind)
		}
	}
	return nil
}

func checkManagedArtifact(name string, manifest manifestpkg.ResourceManifest) error {
	service := manifest.ManagedService
	if service == nil {
		return nil
	}
	for _, osName := range []string{"linux", "macos", "windows"} {
		target, found := manifest.Deployment.Target("desktop", osName, "")
		if !found {
			return fmt.Errorf("resource %s deployment profile does not declare %s", name, osName)
		}
		if target.Support == "unsupported" {
			if strings.TrimSpace(target.Reason) == "" {
				return fmt.Errorf("resource %s unsupported %s target has no reason", name, osName)
			}
			continue
		}
		if manifest.Bundling == "host-required" && strings.HasPrefix(target.Mode, "bundled-") {
			return fmt.Errorf("resource %s claims host-required bundling but desktop mode %s ships an artifact", name, target.Mode)
		}
		if service.Acquisition == nil {
			return fmt.Errorf("resource %s managed artifact has no acquisition contract for claimed %s", name, osName)
		}
		if manifest.Bundling == "vendorable" {
			for index, candidate := range service.Acquisition.Targets {
				if !binaryfetch.UsesOnlyBuildTimeFacts(candidate) {
					return fmt.Errorf("resource %s vendorable acquisition target %d for %s uses runtime facts", name, index, osName)
				}
			}
		}
		for _, arch := range target.Architectures {
			platform, err := resourcedeployment.ParsePlatform(osName + "-" + arch)
			if err != nil {
				return err
			}
			factsOS := platform.OS
			if factsOS == "macos" {
				factsOS = "darwin"
			}
			acquired, err := service.Acquisition.Resolve(binaryfetch.Facts{"os": factsOS, "arch": arch})
			if err != nil {
				return fmt.Errorf("resource %s acquisition does not cover %s: %w", name, platform, err)
			}
			if acquired.Unsupported != "" {
				return fmt.Errorf("resource %s claims %s but acquisition says unsupported: %s", name, platform, acquired.Unsupported)
			}
			if strings.EqualFold(strings.TrimSpace(service.Artifact.Verification), "host-tool") {
				if service.Acquisition.Kind != "none" || strings.TrimSpace(acquired.Executable) == "" || acquired.Executable != service.Artifact.Path {
					return fmt.Errorf("resource %s host-tool artifact for %s must adopt artifact path %q through acquisition kind none", name, platform, service.Artifact.Path)
				}
				continue
			}
			artifact, err := service.Artifact.ForPlatform(osName, arch)
			if err != nil {
				return fmt.Errorf("resource %s managed artifact %s: %w", name, platform, err)
			}
			if acquired.Layout != "dir" && acquired.Archive == "none" && !strings.EqualFold(strings.TrimSpace(acquired.SHA256), strings.TrimSpace(artifact.SHA256)) {
				return fmt.Errorf("resource %s download/artifact digest mismatch for %s", name, platform)
			}
			if !binaryfetch.UsesOnlyBuildTimeFacts(acquired) && manifest.Bundling == "vendorable" {
				return fmt.Errorf("resource %s vendorable acquisition for %s uses runtime facts", name, platform)
			}
		}
	}
	return nil
}

// legacyAcceleratorSurfaces are the three declarations the acceleration block
// replaced, with where each one's value now lives.
var legacyAcceleratorSurfaces = []struct {
	description string
	present     func(map[string]json.RawMessage) bool
	moveTo      string
}{
	{
		description: "gpu block",
		present:     func(raw map[string]json.RawMessage) bool { return hasJSONValue(raw["gpu"]) },
		moveTo:      "acceleration.<backend>",
	},
	{
		description: "top-level capacity block",
		present:     func(raw map[string]json.RawMessage) bool { return hasJSONValue(raw["capacity"]) },
		moveTo:      "acceleration.claim",
	},
	{
		description: "requirements.gpu block",
		present: func(raw map[string]json.RawMessage) bool {
			var requirements map[string]json.RawMessage
			if json.Unmarshal(raw["requirements"], &requirements) != nil {
				return false
			}
			return hasJSONValue(requirements["gpu"])
		},
		moveTo: "acceleration.cuda.min_compute",
	},
}

// checkLegacyAcceleratorSurfaces rejects a manifest still carrying any of the
// three surfaces the acceleration block replaced.
//
// It reads the manifest's raw JSON rather than the parsed struct, because the
// parsed struct no longer has fields for them: a manifest that still declares
// one would load cleanly and be silently ignored, which is worse than failing.
//
// The rejection used to fire only for `gpu` on a managed-service resource,
// while internal/capacity used that same block as its only test for "this
// resource uses the GPU". The two disagreed, so the resources that could not
// declare it were invisible to reconciliation and the ones that could raised
// findings forever. One declaration, one rejection, every driver.
func checkLegacyAcceleratorSurfaces(name, manifestPath string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read resource %s manifest: %w", name, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse resource %s manifest: %w", name, err)
	}
	for _, surface := range legacyAcceleratorSurfaces {
		if surface.present(raw) {
			return fmt.Errorf("resource %s declares the deprecated %s; move it to %s", name, surface.description, surface.moveTo)
		}
	}
	return nil
}

// hasJSONValue reports whether a raw JSON field is present and not null.
func hasJSONValue(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

func checkManifestCommands(name string, command []string, platformCommands map[string][]string) error {
	check := func(surface string, args []string) error {
		for _, arg := range args {
			lower := strings.ToLower(strings.TrimSpace(arg))
			if lower == "bash" || lower == "sh" || lower == "-lc" || strings.Contains(lower, ".sh") || strings.HasPrefix(lower, "source ") {
				return fmt.Errorf("resource %s %s invokes a shell: %q", name, surface, strings.Join(args, " "))
			}
		}
		return nil
	}
	if err := check("install", command); err != nil {
		return err
	}
	for platform, args := range platformCommands {
		if err := check("install."+platform, args); err != nil {
			return err
		}
	}
	return nil
}

// checkResourceShellPolicy reports every forbidden shell file across the named
// resources. It collects rather than short-circuits: a resource fleet that is
// meant to run on hosts without bash needs the full list in one run, not the
// alphabetically first offender.
func checkResourceShellPolicy(resourceRoot string, names []string) error {
	violations := make([]string, 0)
	for _, name := range names {
		base := filepath.Join(resourceRoot, name)
		walkErr := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			extension := strings.ToLower(filepath.Ext(path))
			if extension != ".sh" && extension != ".bash" {
				return nil
			}
			rel, relErr := filepath.Rel(filepath.Dir(resourceRoot), path)
			if relErr != nil {
				return relErr
			}
			violations = append(violations, filepath.ToSlash(rel))
			return nil
		})
		if walkErr != nil {
			return fmt.Errorf("scan resource %s for shell files: %w", name, walkErr)
		}
	}
	if len(violations) == 0 {
		return nil
	}
	sort.Strings(violations)
	return fmt.Errorf("resource shell files are forbidden: %s", strings.Join(violations, ", "))
}

func checkManifestImage(path, image string) error {
	if strings.TrimSpace(image) == "" {
		return nil
	}
	if !digestPattern.MatchString(strings.TrimSpace(image)) {
		return fmt.Errorf("%s runtime.image is not pinned by sha256 digest: %q", path, image)
	}
	return nil
}

func checkResourceImages(resourceRoot string) error {
	return filepath.WalkDir(resourceRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == "data" || name == "node_modules" || name == "coverage" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "Dockerfile" && !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			var image string
			switch {
			case strings.HasPrefix(line, "image:"):
				image = strings.TrimSpace(strings.TrimPrefix(line, "image:"))
			case strings.HasPrefix(line, "FROM "):
				image = strings.Fields(strings.TrimPrefix(line, "FROM "))[0]
			default:
				continue
			}
			image = imageReference(image)
			if image == "" || digestPattern.MatchString(image) {
				continue
			}
			return fmt.Errorf("%s:%d image is not pinned by sha256 digest: %q", path, lineNumber, image)
		}
		return scanner.Err()
	})
}

func imageReference(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "'\""))
	if strings.HasPrefix(value, "${") {
		if separator := strings.Index(value, ":-"); separator >= 0 {
			value = value[separator+2:]
			value = strings.TrimSuffix(value, "}")
		}
	}
	return strings.TrimSpace(strings.Trim(value, "'\""))
}

func checkCapabilityContract(root, name string) error {
	path := filepath.Join(root, "resources", name, "config", "capabilities.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	content := string(data)
	if strings.Contains(content, "test_file:") || strings.Contains(content, "endpoint: \"./") {
		return fmt.Errorf("%s retains an unstructured or shell-owned capability entry", path)
	}
	implementationCount := strings.Count(content, "implementation:")
	testCount := strings.Count(content, "test:")
	if implementationCount == 0 || testCount < implementationCount {
		return fmt.Errorf("%s capability commands must declare implementation and test evidence", path)
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "test:") {
			continue
		}
		value := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "test:")), "\"'")
		if strings.HasPrefix(value, "resource.json") {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "resources", name, value)); err != nil {
			return fmt.Errorf("%s capability test %q is missing: %w", path, value, err)
		}
	}
	return nil
}
