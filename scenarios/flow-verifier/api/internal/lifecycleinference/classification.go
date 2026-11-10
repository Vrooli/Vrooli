package lifecycleinference

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// PersistenceClass is the source-backed semantic category used to decide
// whether an inferred state field is eligible for lifecycle conversion.
type PersistenceClass string

const (
	ClassPersistedLifecycle PersistenceClass = "persisted_lifecycle"
	ClassDerivedReadModel   PersistenceClass = "derived_read_model"
	ClassTransientResult    PersistenceClass = "transient_result"
	ClassEnumOnly           PersistenceClass = "enum_only"
	ClassUnknown            PersistenceClass = "unknown"
)

// PersistenceClassification explains the evidence used to classify a
// candidate. Unknown is an intentional result: absent or unparsable source is
// never treated as evidence that a candidate is transient or enum-only.
type PersistenceClassification struct {
	Class        PersistenceClass `json:"class"`
	Reason       string           `json:"reason"`
	Scope        string           `json:"scope"`
	FilesScanned int              `json:"filesScanned"`
	ScanComplete bool             `json:"scanComplete"`
	Evidence     []Evidence       `json:"evidence,omitempty"`
}

type parsedSource struct {
	path       string
	packageDir string
	fset       *token.FileSet
	file       *ast.File
	imports    map[string]string
}

type scenarioSource struct {
	files       []parsedSource
	parseErrors []string
	loadError   string
	scope       string
}

// PersistenceClassifier parses each scenario's API source once and reuses the
// syntax trees for every candidate in that scenario. Create one per inventory
// refresh and call Classify for each candidate.
type PersistenceClassifier struct {
	root      string
	mu        sync.Mutex
	scenarios map[string]scenarioSource
}

func NewPersistenceClassifier(root string) *PersistenceClassifier {
	// The cache is refresh-scoped: construct one classifier for one Analyze
	// call. Do not reuse it after source files change.
	return &PersistenceClassifier{root: root, scenarios: map[string]scenarioSource{}}
}

// ClassifyPersistence inspects current Go source for explicit persistence,
// projection, or transient-result evidence associated with candidate. It is
// conservative: a name or enum alone cannot prove persistence, and ambiguous
// source remains unknown. Persistence calls are recognized when an argument
// contains the candidate type or its lifecycle field.
func ClassifyPersistence(root string, candidate Candidate) PersistenceClassification {
	return NewPersistenceClassifier(root).Classify(candidate)
}

// Classify applies the five-way source-backed classification to candidate.
func (classifier *PersistenceClassifier) Classify(candidate Candidate) PersistenceClassification {
	return classifier.ClassifyContext(context.Background(), candidate)
}

func (classifier *PersistenceClassifier) ClassifyContext(ctx context.Context, candidate Candidate) PersistenceClassification {
	unknown := PersistenceClassification{Class: ClassUnknown, Reason: "insufficient source evidence", Scope: "scenario production Go source"}
	if err := ctx.Err(); err != nil {
		unknown.Reason = err.Error()
		return unknown
	}
	if classifier == nil || strings.TrimSpace(classifier.root) == "" || candidate.ScenarioID == "" || candidate.TypeName == "" || candidate.FieldName == "" {
		unknown.Reason = "candidate identity or source root is incomplete"
		return unknown
	}
	packagePath := filepath.ToSlash(candidate.Package)
	if packagePath != "api" && !strings.HasPrefix(packagePath, "api/") {
		unknown.Reason = "candidate is outside scanned API source; persistence remains unresolved"
		return unknown
	}
	source := classifier.sourceFor(ctx, candidate.ScenarioID)
	if err := ctx.Err(); err != nil {
		unknown.Reason = err.Error()
		return unknown
	}
	unknown.Scope = source.scope
	unknown.FilesScanned = len(source.files)
	unknown.ScanComplete = source.loadError == "" && len(source.parseErrors) == 0
	if source.loadError != "" {
		unknown.Reason = source.loadError
		return unknown
	}
	if len(source.parseErrors) > 0 {
		unknown.Reason = "source scan is incomplete; parse/read failures prevent a class decision"
		for _, message := range source.parseErrors {
			unknown.Evidence = append(unknown.Evidence, Evidence{Path: message, Text: "source evidence unavailable"})
		}
		return unknown
	}

	candidateImportPath := packageImportPath(classifier.root, filepath.Join(classifier.root, "scenarios", candidate.ScenarioID, filepath.FromSlash(candidate.Package)))
	typeGraph := scenarioTypeGraph(source.files, candidate, candidateImportPath)
	var persistence []Evidence
	var readModel []Evidence
	var transient []Evidence
	var writes []Evidence
	var otherUses []Evidence
	for _, parsed := range source.files {
		if err := ctx.Err(); err != nil {
			unknown.Reason = err.Error()
			return unknown
		}
		file := parsed.file
		for _, fn := range file.Decls {
			decl, ok := fn.(*ast.FuncDecl)
			if !ok || decl.Body == nil {
				continue
			}
			flow := analyzeFunctionFlow(ctx, decl, parsed, source.files, candidate, candidateImportPath, typeGraph)
			if err := ctx.Err(); err != nil {
				unknown.Reason = err.Error()
				return unknown
			}
			writes = append(writes, flow.writes...)
			persistence = append(persistence, flow.persistence...)
			readModel = append(readModel, flow.readModels...)
			transient = append(transient, flow.transient...)
			otherUses = append(otherUses, flow.otherUses...)
		}
	}

	// Never claim a semantic class from name suffixes or generic helper names.
	// Persistence requires candidate-specific flow to a typed database/sql
	// write; read-model and transient classes require explicit read/response sinks.
	if len(persistence) > 0 {
		if len(readModel) > 0 {
			return unknownWithEvidence(unknown, "candidate has both durable-write and read-model serving evidence; classify by owner review", append(persistence, readModel...))
		}
		return classificationWithScope(ClassPersistedLifecycle, "candidate-containing value reaches a typed database/sql write; database durability is still owner-verified", source, persistence)
	}
	if len(readModel) > 0 {
		if len(transient) > 0 {
			return classificationWithScope(ClassDerivedReadModel, "candidate value is populated from a typed database/sql read and reaches an explicit response sink; no database/sql write was found", source, append(readModel, transient...))
		}
		return unknownWithEvidence(unknown, "database/sql reads populate the candidate, but response/read-model ownership is not established", readModel)
	}
	if len(transient) > 0 {
		return classificationWithScope(ClassTransientResult, "candidate-containing value reaches an explicit HTTP/Connect response sink and no database/sql write/read sink was found", source, transient)
	}
	if len(writes) == 0 && len(otherUses) == 0 && len(candidate.States) > 0 && candidateDeclarationOnly(ctx, source.files, candidate, candidateImportPath, typeGraph) && !candidateHasStorageTag(source.files, candidate) {
		if err := ctx.Err(); err != nil {
			unknown.Reason = err.Error()
			return unknown
		}
		return classificationWithScope(ClassEnumOnly, "closed vocabulary has no production Go caller or write in the complete scanned scenario source; other-language paths are outside this boundary", source, []Evidence{candidate.Declaration})
	}
	unknown.Reason = "source does not prove candidate-specific flow to a typed durable sink, read-model source, or response sink"
	unknown.Evidence = append(writes, otherUses...)
	return unknown
}

func (classifier *PersistenceClassifier) sourceFor(ctx context.Context, scenarioID string) scenarioSource {
	classifier.mu.Lock()
	defer classifier.mu.Unlock()
	if source, ok := classifier.scenarios[scenarioID]; ok {
		return source
	}
	loaded := scenarioSource{}
	scenarioRoot := filepath.Join(classifier.root, "scenarios", scenarioID)
	if _, err := os.Stat(scenarioRoot); err != nil {
		loaded.loadError = "scenario source is unavailable: " + err.Error()
		classifier.scenarios[scenarioID] = loaded
		return loaded
	}
	loaded.scope = "all production Go files under the scenario; tests/generated/vendor and non-Go sources are outside this boundary"
	files, err := goSourceFiles(scenarioRoot)
	if err != nil {
		loaded.loadError = "cannot enumerate scenario API source: " + err.Error()
		classifier.scenarios[scenarioID] = loaded
		return loaded
	}
	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return scenarioSource{loadError: err.Error(), scope: loaded.scope}
		}
		relative, relErr := filepath.Rel(classifier.root, path)
		if relErr != nil {
			loaded.parseErrors = append(loaded.parseErrors, path+": "+relErr.Error())
			continue
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			loaded.parseErrors = append(loaded.parseErrors, filepath.ToSlash(relative)+": "+readErr.Error())
			continue
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, body, parser.AllErrors)
		if parseErr != nil {
			loaded.parseErrors = append(loaded.parseErrors, filepath.ToSlash(relative)+": "+parseErr.Error())
			continue
		}
		imports := map[string]string{}
		for _, imported := range file.Imports {
			importPath, importErr := strconv.Unquote(imported.Path.Value)
			if importErr != nil {
				continue
			}
			alias := filepath.Base(importPath)
			if imported.Name != nil {
				alias = imported.Name.Name
			}
			if alias != "_" && alias != "." {
				imports[alias] = importPath
			}
		}
		packageDir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(relative)))
		packageDir = strings.TrimPrefix(packageDir, "scenarios/"+scenarioID+"/")
		loaded.files = append(loaded.files, parsedSource{path: filepath.ToSlash(relative), packageDir: packageDir, fset: fset, file: file, imports: imports})
	}
	classifier.scenarios[scenarioID] = loaded
	return loaded
}

func classification(class PersistenceClass, reason string, evidence []Evidence) PersistenceClassification {
	sort.Slice(evidence, func(i, j int) bool {
		if evidence[i].Path != evidence[j].Path {
			return evidence[i].Path < evidence[j].Path
		}
		return evidence[i].Line < evidence[j].Line
	})
	return PersistenceClassification{Class: class, Reason: reason, Evidence: evidence}
}

func goSourceFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == "vendor" || entry.Name() == "generated" || entry.Name() == "docs" || entry.Name() == "testdata" || entry.Name() == "examples" || entry.Name() == "evidence" || strings.HasPrefix(entry.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go") && !strings.HasSuffix(path, ".pb.go") && !strings.HasSuffix(path, "_generated.go") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func typeName(expr ast.Expr) string {
	switch n := expr.(type) {
	case *ast.Ident:
		return n.Name
	case *ast.StarExpr:
		return typeName(n.X)
	case *ast.SelectorExpr:
		return n.Sel.Name
	case *ast.ParenExpr:
		return typeName(n.X)
	default:
		return ""
	}
}
