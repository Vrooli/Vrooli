package installgateway

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PythonBootstrap is a first-lock plan, not permission to skip package approval.
// Every authored direct requirement is exposed to the existing governance owner.
type PythonBootstrap struct {
	Indexes                                pythonIndexSet
	InputPath, InputSHA256, ManifestSHA256 string
	Requirements                           []PythonRequirement
}

type PythonRequirement struct {
	Name, Version, WheelURL, SHA256 string
}

var exactPythonVersion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.+!-]*$`)
var pythonHash = regexp.MustCompile(`^[a-f0-9]{64}$`)
var pythonPackageName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
var pythonNameSeparators = regexp.MustCompile(`[-_.]+`)

func pythonName(name string) string {
	return pythonNameSeparators.ReplaceAllString(strings.ToLower(name), "-")
}

func pythonInputBytes(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return nil, fmt.Errorf("Python input must be a regular file at most 1 MiB")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if len(data) > 1<<20 {
		return nil, fmt.Errorf("Python input exceeds 1 MiB")
	}
	return data, err
}

func pythonDigest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func safePythonURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Opaque != "" {
		return nil, fmt.Errorf("Python sources require HTTPS URLs without credentials or query strings")
	}
	return u, nil
}

func parsePythonDirect(line string, requireHash bool) (PythonRequirement, error) {
	name, raw, ok := strings.Cut(line, " @ ")
	if !ok || !pythonPackageName.MatchString(name) {
		return PythonRequirement{}, fmt.Errorf("expected named direct wheel requirement")
	}
	u, err := safePythonURL(raw)
	if err != nil {
		return PythonRequirement{}, err
	}
	filename := filepath.Base(u.Path)
	if !strings.HasSuffix(filename, ".whl") {
		return PythonRequirement{}, fmt.Errorf("direct Python sources must be wheels; source builds are not permitted")
	}
	parts := strings.Split(strings.TrimSuffix(filename, ".whl"), "-")
	if len(parts) < 5 || pythonName(parts[0]) != pythonName(name) || !exactPythonVersion.MatchString(parts[1]) {
		return PythonRequirement{}, fmt.Errorf("direct wheel name/version does not match requirement")
	}
	hash := strings.TrimPrefix(u.Fragment, "sha256=")
	if (requireHash || u.Fragment != "") && (!strings.HasPrefix(u.Fragment, "sha256=") || !pythonHash.MatchString(hash)) {
		return PythonRequirement{}, fmt.Errorf("direct wheel requires one exact SHA256 URL fragment")
	}
	if !requireHash && u.Fragment == "" {
		hash = ""
	}
	return PythonRequirement{Name: name, Version: parts[1], WheelURL: raw, SHA256: hash}, nil
}

func parsePythonBootstrapInput(data []byte) ([]PythonRequirement, error) {
	var requirements []PythonRequirement
	seen := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var req PythonRequirement
		if strings.Contains(line, " @ ") {
			var err error
			req, err = parsePythonDirect(line, true)
			if err != nil {
				return nil, err
			}
		} else {
			match := pinnedRequirement.FindStringSubmatch(line)
			if len(match) != 3 || strings.HasSuffix(line, "\\") || !exactPythonVersion.MatchString(match[2]) {
				return nil, fmt.Errorf("bootstrap input requires exact name==version pins or hash-bound direct wheels; includes/options/ranges are not supported")
			}
			req = PythonRequirement{Name: match[1], Version: match[2]}
		}
		key := pythonName(req.Name)
		if seen[key] {
			return nil, fmt.Errorf("duplicate Python direct requirement %q", req.Name)
		}
		seen[key] = true
		requirements = append(requirements, req)
	}
	if len(requirements) == 0 {
		return nil, fmt.Errorf("bootstrap input is empty")
	}
	return requirements, nil
}

func resolvePythonBootstrap(root, packageName, version string) (*PythonBootstrap, error) {
	manifest, err := pythonInputBytes(filepath.Join(root, "resource.json"))
	if err != nil {
		return nil, err
	}
	indexes := pythonIndexSetsFromJSON(manifest)
	index, found := indexes["requirements.lock"]
	if !found || index.Lockfile != "requirements.lock" {
		return nil, fmt.Errorf("Python bootstrap requires resource.json to declare python-wheels lockfile requirements.lock")
	}
	for _, raw := range append([]string{index.Primary}, index.Extras...) {
		u, err := safePythonURL(raw)
		if err != nil || u.Fragment != "" {
			return nil, fmt.Errorf("invalid resource Python index URL")
		}
	}
	input := filepath.Join(root, "requirements.in")
	data, err := pythonInputBytes(input)
	if err != nil {
		return nil, err
	}
	requirements, err := parsePythonBootstrapInput(data)
	if err != nil {
		return nil, err
	}
	found = false
	for _, requirement := range requirements {
		if pythonName(requirement.Name) == pythonName(packageName) && requirement.Version == version {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("bootstrap trigger must match an authored exact direct requirement")
	}
	return &PythonBootstrap{InputPath: input, InputSHA256: pythonDigest(data), ManifestSHA256: pythonDigest(manifest), Requirements: requirements, Indexes: index}, nil
}

func bootstrapPythonLock(ctx context.Context, r Resolution) (string, error) {
	plan := r.PythonBootstrap
	if _, err := os.Lstat(r.PythonLockfiles[0]); err == nil {
		return "", fmt.Errorf("Python bootstrap output already exists")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	data, err := pythonInputBytes(plan.InputPath)
	if err != nil {
		return "", err
	}
	manifest, err := pythonInputBytes(filepath.Join(r.SurfaceRoot, "resource.json"))
	if err != nil {
		return "", err
	}
	if pythonDigest(data) != plan.InputSHA256 || pythonDigest(manifest) != plan.ManifestSHA256 {
		return "", fmt.Errorf("Python bootstrap input or index manifest changed after approval")
	}
	work, err := os.MkdirTemp(r.SurfaceRoot, ".sda-python-lock-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(work)
	input, output := filepath.Join(work, "requirements.in"), filepath.Join(work, "requirements.lock")
	if err := os.WriteFile(input, data, 0o600); err != nil {
		return "", err
	}
	diagnostic, err := compilePythonLock(ctx, r.SurfaceRoot, input, output, plan.Indexes)
	if err != nil {
		return diagnostic, fmt.Errorf("uv wheel-only bootstrap failed: %w", err)
	}
	generated, err := pythonInputBytes(output)
	if err != nil {
		return diagnostic, err
	}
	if err := validatePythonGeneratedLock(generated, plan.Requirements); err != nil {
		return diagnostic, err
	}
	current, err := pythonInputBytes(plan.InputPath)
	if err != nil || pythonDigest(current) != plan.InputSHA256 {
		return diagnostic, fmt.Errorf("Python bootstrap authored input changed during resolution")
	}
	currentManifest, err := pythonInputBytes(filepath.Join(r.SurfaceRoot, "resource.json"))
	if err != nil || pythonDigest(currentManifest) != plan.ManifestSHA256 {
		return diagnostic, fmt.Errorf("Python bootstrap index manifest changed during resolution")
	}
	relativeInput, err := filepath.Rel(r.RepositoryRoot, plan.InputPath)
	if err != nil {
		return diagnostic, err
	}
	if err := os.WriteFile(output, []byte(canonicalizePythonLockOutput(generated, input, filepath.ToSlash(relativeInput))), 0o644); err != nil {
		return diagnostic, err
	}
	// Link publishes without replacing a lock created by another owner after resolve.
	if err := os.Link(output, r.PythonLockfiles[0]); err != nil {
		return diagnostic, fmt.Errorf("publish first Python lock without replacement: %w", err)
	}
	return diagnostic + "\nHashed wheel-only resource lock bootstrapped through Scenario Dependency Analyzer", nil
}

func validatePythonGeneratedLock(data []byte, authored []PythonRequirement) error {
	seen := map[string]PythonRequirement{}
	var current PythonRequirement
	var hashes []string
	finish := func() error {
		if current.Name == "" {
			return nil
		}
		if len(hashes) == 0 {
			return fmt.Errorf("resolved Python requirement %q has no SHA256", current.Name)
		}
		if current.SHA256 != "" {
			for _, hash := range hashes {
				if hash != current.SHA256 {
					return fmt.Errorf("direct wheel hash set widened beyond exact publisher SHA256")
				}
			}
		}
		key := pythonName(current.Name)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate resolved Python requirement")
		}
		seen[key] = current
		return nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "\\"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "--hash=sha256:") {
			hash := strings.TrimPrefix(line, "--hash=sha256:")
			if current.Name == "" || !pythonHash.MatchString(hash) {
				return fmt.Errorf("invalid resolved Python hash")
			}
			hashes = append(hashes, hash)
			continue
		}
		if err := finish(); err != nil {
			return err
		}
		hashes = nil
		if strings.Contains(line, " @ ") {
			var err error
			current, err = parsePythonDirect(line, false)
			if err != nil {
				return err
			}
			for _, want := range authored {
				if pythonName(want.Name) == pythonName(current.Name) && want.WheelURL != "" {
					if current.SHA256 != "" && current.SHA256 != want.SHA256 {
						return fmt.Errorf("resolved direct wheel URL hash changed")
					}
					current.SHA256 = want.SHA256
				}
			}
		} else {
			match := pinnedRequirement.FindStringSubmatch(line)
			if len(match) != 3 || !exactPythonVersion.MatchString(match[2]) {
				return fmt.Errorf("resolved Python lock contains non-pinned or unsupported requirement")
			}
			current = PythonRequirement{Name: match[1], Version: match[2]}
		}
	}
	if err := finish(); err != nil {
		return err
	}
	for _, want := range authored {
		got, ok := seen[pythonName(want.Name)]
		if !ok || got.Version != want.Version {
			return fmt.Errorf("resolved lock changed or omitted authored requirement %q", want.Name)
		}
		if want.WheelURL != "" {
			gotURL, _ := url.Parse(got.WheelURL)
			wantURL, _ := url.Parse(want.WheelURL)
			if gotURL == nil || wantURL == nil {
				return fmt.Errorf("resolved direct wheel URL missing")
			}
			gotURL.Fragment, gotURL.RawPath = "", ""
			wantURL.Fragment, wantURL.RawPath = "", ""
			if gotURL.String() != wantURL.String() {
				return fmt.Errorf("resolved direct wheel URL changed")
			}
		}
	}
	return nil
}
