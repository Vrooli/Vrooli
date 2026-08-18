package resources

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
		manifest, err := manifestpkg.Load(manifestPath)
		if err != nil {
			return fmt.Errorf("load resource %s: %w", name, err)
		}
		if err := checkHealthKinds(name, manifest.HealthChecks); err != nil {
			return err
		}
		if err := checkResourceShellPolicy(resourceRoot, name); err != nil {
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
		if err := checkCapabilityContract(root, name); err != nil {
			return err
		}
	}
	if err := checkResourceImages(resourceRoot); err != nil {
		return err
	}
	return nil
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
	if manifest.GPU != nil {
		return fmt.Errorf("resource %s managed-service retains obsolete gpu block", name)
	}
	return nil
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

func checkResourceShellPolicy(resourceRoot, name string) error {
	base := filepath.Join(resourceRoot, name)
	return filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
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
		rel, err := filepath.Rel(filepath.Dir(resourceRoot), path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		return fmt.Errorf("resource shell file is forbidden: %s", rel)
	})
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
