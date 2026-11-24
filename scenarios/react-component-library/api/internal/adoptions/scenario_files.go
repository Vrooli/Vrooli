package adoptions

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/librarywalk"

	"github.com/vrooli/api-core/storage"
)

// FSScenarioFileReader is the production ScenarioFileReader. Maps
// adopted_path under root/<scenario>/ with a path-traversal guard.
type FSScenarioFileReader struct {
	root string
}

// NewFSScenarioFileReader constructs an FSScenarioFileReader rooted at
// root. Root must be absolute; callers resolve it via api-core/storage.
func NewFSScenarioFileReader(root string) *FSScenarioFileReader {
	return &FSScenarioFileReader{root: root}
}

var (
	_ ScenarioFileReader                   = (*FSScenarioFileReader)(nil)
	_ ScenarioProvenanceScanner            = (*FSScenarioFileReader)(nil)
	_ ScenarioCandidateScanner             = (*FSScenarioFileReader)(nil)
	_ ScenarioTokenNamespaceReader         = (*FSScenarioFileReader)(nil)
	_ ScenarioTokenInventoryReader         = (*FSScenarioFileReader)(nil)
	_ ScenarioRuntimeTokenInventoryReader  = (*FSScenarioFileReader)(nil)
	_ ScenarioLibraryImportReader          = (*FSScenarioFileReader)(nil)
	_ ScenarioReferenceTokenReader         = (*FSScenarioFileReader)(nil)
	_ ScenarioTokenFallbackExemptionReader = (*FSScenarioFileReader)(nil)
)

// scannedSource is one .ts/.tsx file yielded by walkScenarioSources: its
// scenario-relative path plus raw bytes. Both ScanProvenance and ScanUntagged
// classify these; neither re-walks the tree.
type scannedSource struct {
	scenario    string
	adoptedPath string
	content     []byte
}

// walkScenarioSources is the single filesystem walk shared by ScanProvenance
// (header-tagged files) and ScanUntagged (header-less candidates). It reads
// every non-symlink .ts/.tsx file under each scenario's ui/src tree, skipping
// node_modules. It never writes.
func (r *FSScenarioFileReader) walkScenarioSources(ctx context.Context) ([]scannedSource, error) {
	entries, err := os.ReadDir(r.root)
	if err != nil {
		return nil, fmt.Errorf("list scenarios for scan: %w", err)
	}
	var out []scannedSource
	for _, scenario := range entries {
		if !scenario.IsDir() {
			continue
		}
		base := filepath.Join(r.root, scenario.Name(), "ui", "src")
		if _, err := os.Stat(base); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		err := librarywalk.WalkContext(ctx, base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || (!strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx")) {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(filepath.Join(r.root, scenario.Name()), path)
			if err != nil {
				return err
			}
			out = append(out, scannedSource{scenario: scenario.Name(), adoptedPath: filepath.ToSlash(rel), content: raw})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan sources for %s: %w", scenario.Name(), err)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].scenario != out[j].scenario {
			return out[i].scenario < out[j].scenario
		}
		return out[i].adoptedPath < out[j].adoptedPath
	})
	return out, nil
}

// ScanProvenance reads only scenario UI source files. It never writes target
// scenarios; reconciliation owns the sole database mutation after this scan.
func (r *FSScenarioFileReader) ScanProvenance(ctx context.Context) ([]ProvenanceFile, error) {
	sources, err := r.walkScenarioSources(ctx)
	if err != nil {
		return nil, err
	}
	var out []ProvenanceFile
	for _, src := range sources {
		libraryID := provenanceField(string(src.content), "@vrooliComponentSource")
		version := provenanceField(string(src.content), "@vrooliComponentVersion")
		if libraryID == "" || version == "" {
			continue
		}
		out = append(out, ProvenanceFile{Scenario: src.scenario, AdoptedPath: src.adoptedPath, LibraryID: libraryID, Version: version, AdoptionID: provenanceField(string(src.content), "@vrooliComponentAdoption"), Content: src.content})
	}
	return out, nil
}

// ScanUntagged returns the inverse set: every .ts/.tsx file that has NO
// @vrooliComponentSource/@vrooliComponentVersion header. These are the files
// discovery scores for similarity; a real vendored copy that lost its header
// is exactly the drift-blind case this closes.
func (r *FSScenarioFileReader) ScanUntagged(ctx context.Context) ([]CandidateFile, error) {
	sources, err := r.walkScenarioSources(ctx)
	if err != nil {
		return nil, err
	}
	var out []CandidateFile
	for _, src := range sources {
		libraryID := provenanceField(string(src.content), "@vrooliComponentSource")
		version := provenanceField(string(src.content), "@vrooliComponentVersion")
		if libraryID != "" && version != "" {
			continue // already tagged; ScanProvenance/Reconcile owns these
		}
		out = append(out, CandidateFile{Scenario: src.scenario, AdoptedPath: src.adoptedPath, Content: src.content})
	}
	return out, nil
}

func (r *FSScenarioFileReader) Read(_ context.Context, scenario, adoptedPath string) ([]byte, error) {
	cleaned, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return nil, err
	}
	bytes, err := os.ReadFile(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrAdoptedFileMissing{Scenario: scenario, AdoptedPath: adoptedPath}
		}
		return nil, fmt.Errorf("read adopted file %q: %w", cleaned, err)
	}
	return bytes, nil
}

func (r *FSScenarioFileReader) Exists(_ context.Context, scenario, adoptedPath string) (bool, error) {
	cleaned, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(cleaned)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (r *FSScenarioFileReader) Write(ctx context.Context, scenario, adoptedPath string, content []byte) (string, error) {
	cleaned, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return "", err
	}
	root, relative, err := r.openScenarioFileRoot(scenario, adoptedPath)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := storage.WriteFileAtomicInRoot(root, relative, content, 0o600); err != nil {
		return "", fmt.Errorf("write adopted file %q: %w", adoptedPath, err)
	}
	if err := r.formatWrittenSource(ctx, scenario, adoptedPath, cleaned); err != nil {
		return "", err
	}
	return cleaned, nil
}

// WriteRaw restores a batch snapshot without invoking the scenario formatter.
// Rollback must recover the exact bytes that existed before the batch began.
func (r *FSScenarioFileReader) WriteRaw(_ context.Context, scenario, adoptedPath string, content []byte) error {
	root, relative, err := r.openScenarioFileRoot(scenario, adoptedPath)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := storage.WriteFileAtomicInRoot(root, relative, content, 0o600); err != nil {
		return fmt.Errorf("restore adopted file %q: %w", adoptedPath, err)
	}
	return nil
}

func (r *FSScenarioFileReader) Remove(_ context.Context, scenario, adoptedPath string) error {
	root, relative, err := r.openScenarioFileRoot(scenario, adoptedPath)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.Remove(relative); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove adopted file %q: %w", adoptedPath, err)
	}
	return nil
}

// formatWrittenSource keeps copied source readable without imposing a
// library-global formatter on consumers. Each adopting scenario owns its UI
// toolchain; when that toolchain includes Prettier, use that exact binary and
// its config. Scenarios without a formatter remain adoptable, while the
// component-library scenario (and the standard UI template) get formatting
// automatically at the write boundary.
func (r *FSScenarioFileReader) formatWrittenSource(ctx context.Context, scenario, adoptedPath, absolutePath string) error {
	ext := strings.ToLower(filepath.Ext(adoptedPath))
	if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" && ext != ".css" && ext != ".json" {
		return nil
	}
	uiRoot := filepath.Join(r.root, scenario, "ui")
	candidates := []string{
		filepath.Join(uiRoot, "node_modules", ".bin", "prettier"),
		filepath.Join(uiRoot, "node_modules", ".bin", "prettier.cmd"),
		filepath.Join(uiRoot, "node_modules", "prettier", "bin", "prettier.cjs"),
	}
	formatter := ""
	for _, candidate := range candidates {
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			formatter = candidate
			break
		}
	}
	if formatter == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, formatter, "--write", absolutePath)
	cmd.Dir = uiRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("format adopted file %q with target Prettier: %w: %s", adoptedPath, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// openScenarioFileRoot anchors a managed file operation at the selected
// scenario directory. resolve validates the caller path before it is converted
// back to a root-relative path; os.Root then keeps symlinks from redirecting
// the operation outside that directory.
func (r *FSScenarioFileReader) openScenarioFileRoot(scenario, adoptedPath string) (*os.Root, string, error) {
	cleaned, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return nil, "", err
	}
	base := filepath.Join(r.root, scenario)
	relative, err := filepath.Rel(base, cleaned)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, "", fmt.Errorf("adopted_path %q escapes scenario root", adoptedPath)
	}
	root, err := openScenarioRoot(base)
	if err != nil {
		return nil, "", fmt.Errorf("open scenario root %q: %w", scenario, err)
	}
	return root, relative, nil
}

func openScenarioRoot(base string) (*os.Root, error) {
	if err := storage.EnsureDirectory(base, 0o755); err != nil {
		return nil, fmt.Errorf("ensure scenario root: %w", err)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, fmt.Errorf("open scenario root %q: %w", base, err)
	}
	return root, nil
}

// resolve maps (scenario, adoptedPath) onto disk. Scenario is either a plain
// scenario name under root, or a template key ("../templates/scenarios/<id>")
// pointing at a vendored template copy; only adoptedPath is traversal-guarded
// against escaping the resolved scenario dir.
func (r *FSScenarioFileReader) resolve(scenario, adoptedPath string) (string, error) {
	base := filepath.Join(r.root, scenario)
	cleaned := filepath.Clean(filepath.Join(base, adoptedPath))
	rel, err := filepath.Rel(base, cleaned)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("adopted_path %q escapes scenario root", adoptedPath)
	}
	return cleaned, nil
}
