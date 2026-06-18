package genmanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
)

const SchemaVersion = 1

type Manifest struct {
	SchemaVersion int               `json:"schema_version"`
	Scenario      string            `json:"scenario"`
	InputDigest   string            `json:"input_digest"`
	InputFiles    []string          `json:"input_files"`
	Toolchain     Toolchain         `json:"toolchain"`
	Outputs       map[string]string `json:"outputs"`
}

type Toolchain struct {
	Buf              string            `json:"buf"`
	BufGenYAMLDigest string            `json:"buf_gen_yaml_digest"`
	Plugins          map[string]string `json:"plugins"`
}

type Options struct {
	RepoRoot  string
	ProtoRoot string
}

var importRE = regexp.MustCompile(`(?m)^\s*import\s+(?:(?:public|weak)\s+)?"([^"]+)"\s*;`)

func BuildManifest(opts Options, scenario string) (Manifest, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Manifest{}, fmt.Errorf("scenario is required")
	}
	files, digest, err := InputClosure(opts, scenario)
	if err != nil {
		return Manifest{}, err
	}
	toolchain, err := ToolchainFingerprint(opts)
	if err != nil {
		return Manifest{}, err
	}
	outputs, err := OutputDigests(opts, scenario)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{
		SchemaVersion: SchemaVersion,
		Scenario:      scenario,
		InputDigest:   digest,
		InputFiles:    files,
		Toolchain:     toolchain,
		Outputs:       outputs,
	}, nil
}

func ScenarioNames(protoRoot string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(protoRoot, "schemas"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, entry.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func InputClosure(opts Options, scenario string) ([]string, string, error) {
	protoRoot := cleanProtoRoot(opts)
	startRoot := filepath.Join(protoRoot, "schemas", scenario)
	if _, err := os.Stat(startRoot); err != nil {
		return nil, "", fmt.Errorf("stat scenario schema %s: %w", scenario, err)
	}
	seen := map[string]bool{}
	var visit func(string) error
	visit = func(rel string) error {
		rel = filepath.ToSlash(strings.TrimPrefix(rel, "packages/proto/"))
		if seen[rel] {
			return nil
		}
		seen[rel] = true
		raw, err := os.ReadFile(filepath.Join(protoRoot, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("read input proto %s: %w", rel, err)
		}
		for _, imported := range parseImports(string(raw)) {
			resolved, ok := resolveImport(protoRoot, imported)
			if !ok {
				return fmt.Errorf("resolve import %q from %s", imported, rel)
			}
			if resolved == "" {
				continue
			}
			if err := visit(resolved); err != nil {
				return err
			}
		}
		return nil
	}
	err := filepath.WalkDir(startRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".proto" {
			return nil
		}
		rel, err := filepath.Rel(protoRoot, path)
		if err != nil {
			return err
		}
		return visit(filepath.ToSlash(rel))
	})
	if err != nil {
		return nil, "", err
	}
	files := sortedKeys(seen)
	digest, err := digestFiles(protoRoot, files)
	if err != nil {
		return nil, "", err
	}
	return files, digest, nil
}

func ToolchainFingerprint(opts Options) (Toolchain, error) {
	protoRoot := cleanProtoRoot(opts)
	repoRoot := cleanRepoRoot(opts)
	bufGenDigest, err := digestOne(filepath.Join(protoRoot, "buf.gen.yaml"))
	if err != nil {
		return Toolchain{}, err
	}
	versions := map[string]string{}
	for _, name := range []string{"buf", "protoc-gen-go", "protoc-gen-connect-go", "protoc-gen-es", "protoc"} {
		version, err := toolVersion(filepath.Join(repoRoot, "internal", "tools", name, "tool.json"))
		if err != nil {
			return Toolchain{}, err
		}
		versions[name] = version
	}
	return Toolchain{
		Buf:              versions["buf"],
		BufGenYAMLDigest: bufGenDigest,
		Plugins: map[string]string{
			"protoc-gen-go":         versions["protoc-gen-go"],
			"protoc-gen-connect-go": versions["protoc-gen-connect-go"],
			"protoc-gen-es":         versions["protoc-gen-es"],
			"protoc":                versions["protoc"],
		},
	}, nil
}

func OutputDigests(opts Options, scenario string) (map[string]string, error) {
	protoRoot := cleanProtoRoot(opts)
	files, err := OutputFiles(protoRoot, scenario)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(files))
	type result struct {
		rel    string
		digest string
		err    error
	}
	workers := minInt(len(files), runtime.NumCPU())
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan string)
	results := make(chan result, len(files))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range jobs {
				digest, err := digestOne(filepath.Join(protoRoot, filepath.FromSlash(rel)))
				results <- result{rel: rel, digest: digest, err: err}
			}
		}()
	}
	for _, rel := range files {
		jobs <- rel
	}
	close(jobs)
	go func() {
		wg.Wait()
		close(results)
	}()
	for result := range results {
		if result.err != nil {
			return nil, result.err
		}
		out[result.rel] = result.digest
	}
	return out, nil
}

func OutputFiles(protoRoot, scenario string) ([]string, error) {
	var out []string
	for _, dir := range scenarioOutputDirs(scenario) {
		root := filepath.Join(protoRoot, filepath.FromSlash(dir))
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(protoRoot, path)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

func WriteManifest(path string, manifest Manifest) error {
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func ManifestPath(protoRoot, scenario string) string {
	return filepath.Join(protoRoot, "gen", "manifests", scenario+".lock.json")
}

func LoadManifest(protoRoot, scenario string) (Manifest, error) {
	raw, err := os.ReadFile(ManifestPath(protoRoot, scenario))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, err
	}
	if manifest.SchemaVersion != SchemaVersion {
		return Manifest{}, fmt.Errorf("unsupported manifest schema_version %d", manifest.SchemaVersion)
	}
	if manifest.Scenario != scenario {
		return Manifest{}, fmt.Errorf("manifest scenario %q does not match %q", manifest.Scenario, scenario)
	}
	return manifest, nil
}

func parseImports(src string) []string {
	matches := importRE.FindAllStringSubmatch(src, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			out = append(out, match[1])
		}
	}
	sort.Strings(out)
	return out
}

func resolveImport(protoRoot, imported string) (string, bool) {
	if strings.HasPrefix(imported, "google/protobuf/") {
		return "", true
	}
	for _, candidate := range []string{
		filepath.Join("schemas", filepath.FromSlash(imported)),
		filepath.Join("vendor", "googleapis", filepath.FromSlash(imported)),
		filepath.Join("vendor", "protovalidate", filepath.FromSlash(imported)),
		filepath.FromSlash(imported),
	} {
		if _, err := os.Stat(filepath.Join(protoRoot, candidate)); err == nil {
			return filepath.ToSlash(candidate), true
		}
	}
	return "", false
}

func scenarioOutputDirs(scenario string) []string {
	pythonScenario := strings.ReplaceAll(scenario, "-", "_")
	return []string{
		filepath.ToSlash(filepath.Join("gen", "go", scenario)),
		filepath.ToSlash(filepath.Join("gen", "typescript", scenario)),
		filepath.ToSlash(filepath.Join("gen", "typescript", "js", scenario)),
		filepath.ToSlash(filepath.Join("gen", "python", pythonScenario)),
	}
}

func digestFiles(root string, files []string) (string, error) {
	h := sha256.New()
	for _, rel := range files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write(raw)
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func digestOne(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func toolVersion(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var data struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", err
	}
	if strings.TrimSpace(data.Version) == "" {
		return "", fmt.Errorf("%s has no version", path)
	}
	return data.Version, nil
}

func cleanProtoRoot(opts Options) string {
	if opts.ProtoRoot != "" {
		return opts.ProtoRoot
	}
	return filepath.Join(cleanRepoRoot(opts), "packages", "proto")
}

func cleanRepoRoot(opts Options) string {
	if opts.RepoRoot != "" {
		return opts.RepoRoot
	}
	if opts.ProtoRoot != "" {
		return filepath.Clean(filepath.Join(opts.ProtoRoot, "..", ".."))
	}
	return filepath.Clean(filepath.Join("..", ".."))
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
