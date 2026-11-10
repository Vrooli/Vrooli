// Package lifecycleinference ranks persisted lifecycle candidates from source
// evidence. A candidate is a review target, not a claim that the field is a
// state machine; callers must verify its transition owner and allowed edges.
package lifecycleinference

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type Inventory struct {
	Root                       string            `json:"root"`
	ScenarioCount              int               `json:"scenarioCount"`
	CandidateCount             int               `json:"candidateCount"`
	HuntCoverageComplete       bool              `json:"huntCoverageComplete"`
	HuntCoverageScope          string            `json:"huntCoverageScope,omitempty"`
	PreselectionCandidateCount int               `json:"preselectionCandidateCount"`
	PreselectionCandidates     []Candidate       `json:"preselectionCandidates,omitempty"`
	UnrankedCandidates         []Candidate       `json:"unrankedCandidates,omitempty"`
	Scenarios                  []ScenarioSummary `json:"scenarios"`
	Candidates                 []Candidate       `json:"candidates"`
}

type ScenarioSummary struct {
	ID              string `json:"id"`
	SourceFiles     int    `json:"sourceFiles"`
	ParseErrors     int    `json:"parseErrors"`
	FlowParseErrors int    `json:"flowContractParseErrors"`
	CandidateCount  int    `json:"candidateCount"`
	ExplicitFlows   int    `json:"explicitFlowContracts"`
}

type Candidate struct {
	ID                 string                    `json:"id"`
	ScenarioID         string                    `json:"scenarioId"`
	Package            string                    `json:"package"`
	TypeName           string                    `json:"typeName"`
	FieldName          string                    `json:"fieldName"`
	FieldType          string                    `json:"fieldType"`
	Declaration        Evidence                  `json:"declaration"`
	States             []StateEvidence           `json:"states,omitempty"`
	RelatedFlags       []Evidence                `json:"relatedFlags,omitempty"`
	TimeFields         []Evidence                `json:"timeFields,omitempty"`
	WriteSites         []Evidence                `json:"writeSites,omitempty"`
	GuardSites         []Evidence                `json:"guardSites,omitempty"`
	TransitionOwners   []Evidence                `json:"transitionOwners,omitempty"`
	TestFiles          []string                  `json:"testFiles,omitempty"`
	HuntRecords        []HuntReference           `json:"huntRecords,omitempty"`
	Persistence        PersistenceClassification `json:"persistence"`
	PreselectionStatus string                    `json:"preselectionStatus"`
	ExplicitFlows      []FlowReference           `json:"explicitFlows,omitempty"`
	ExplicitlyModeled  bool                      `json:"explicitlyModeled,omitempty"`
	BurdenScore        int                       `json:"burdenScore"`
	ScoreBreakdown     []string                  `json:"scoreBreakdown"`
	Confidence         string                    `json:"confidence"`
}

type Evidence struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

type StateEvidence struct {
	Name  string   `json:"name"`
	Value string   `json:"value"`
	At    Evidence `json:"at"`
}

type HuntReference struct {
	EntryID string `json:"entryId"`
	Round   string `json:"round"`
	Status  string `json:"status"`
}

// FlowReference links one candidate to a flow contract only when the contract
// names the candidate's status type. ExactLifecycleMatch additionally requires
// the full state-value sets to agree; it is still a lead until callers and the
// transition owner are audited.
type FlowReference struct {
	FlowID              string   `json:"flowId"`
	Path                string   `json:"path"`
	StatusType          string   `json:"statusType"`
	MatchedStates       []string `json:"matchedStates,omitempty"`
	CandidateStateCount int      `json:"candidateStateCount"`
	ContractStateCount  int      `json:"contractStateCount"`
	ExactLifecycleMatch bool     `json:"exactLifecycleMatch"`
}

type Options struct {
	// HuntIndex is an optional Search Hub recall JSON file whose `hits` rows
	// contain entry_id and text. A reviewed `candidate_ids` binding may connect
	// a source identity to a hunt record when prose uses generic domain terms.
	// Matches are leads only; status is always "review" because record prose
	// cannot prove a lifecycle closure.
	HuntIndex string
	// HuntIndexJSON carries an already-read recall envelope for API callers.
	// The server never reads a caller-supplied filesystem path.
	HuntIndexJSON []byte
}

type packageFacts struct {
	path           string
	files          []parsedFile
	consts         map[string][]StateEvidence
	structs        map[string]map[string]fieldFact
	structFields   map[string]map[string]string
	writeSites     map[string][]Evidence
	guardSites     map[string][]Evidence
	tests          map[string][]string
	owners         map[string][]Evidence
	ownerTestTypes map[string][]string
}

type parsedFile struct {
	path                string
	source              string
	file                *ast.File
	fset                *token.FileSet
	test                bool
	localPackageAliases map[string]bool
}

type fieldFact struct {
	typeName string
	decl     Evidence
	flags    []Evidence
	times    []Evidence
}

// Analyze walks service-backed scenarios and ranks Go lifecycle candidates.
// It deliberately scans source rather than generated binaries and reports
// parse failures per scenario so one damaged owner cannot hide the fleet.
func Analyze(root string, opts Options) (Inventory, error) {
	return AnalyzeContext(context.Background(), root, opts)
}

// AnalyzeContext stops source parsing and classification when its request is
// canceled or its bounded operation expires.
func AnalyzeContext(ctx context.Context, root string, opts Options) (Inventory, error) {
	if err := ctx.Err(); err != nil {
		return Inventory{}, err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, err
	}
	entries, err := os.ReadDir(filepath.Join(abs, "scenarios"))
	if err != nil {
		return Inventory{}, fmt.Errorf("lifecycle inventory: read scenarios: %w", err)
	}
	index, err := readHuntIndex(opts)
	if err != nil {
		return Inventory{}, err
	}
	inventory := Inventory{Root: abs, Scenarios: []ScenarioSummary{}, Candidates: []Candidate{}}
	inventory.HuntCoverageComplete = index.coverageComplete()
	inventory.HuntCoverageScope = index.Scope
	persistenceClassifier := NewPersistenceClassifier(abs)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return Inventory{}, err
		}
		if !entry.IsDir() {
			continue
		}
		scenarioRoot := filepath.Join(abs, "scenarios", entry.Name())
		if _, err := os.Stat(filepath.Join(scenarioRoot, ".vrooli", "service.json")); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Inventory{}, fmt.Errorf("lifecycle inventory: scenario %s: %w", entry.Name(), err)
		}
		summary, candidates, err := analyzeScenario(ctx, abs, scenarioRoot, entry.Name())
		if err != nil {
			return Inventory{}, err
		}
		inventory.ScenarioCount++
		inventory.CandidateCount += len(candidates)
		inventory.Scenarios = append(inventory.Scenarios, summary)
		for i := range candidates {
			if err := ctx.Err(); err != nil {
				return Inventory{}, err
			}
			candidates[i].HuntRecords = matchHunt(candidates[i], index)
			candidates[i].Persistence = persistenceClassifier.ClassifyContext(ctx, candidates[i])
			if err := ctx.Err(); err != nil {
				return Inventory{}, err
			}
			inventory.Candidates = append(inventory.Candidates, candidates[i])
		}
	}
	rawCandidates := inventory.Candidates
	inventory.Candidates = make([]Candidate, 0, len(rawCandidates))
	for i := range rawCandidates {
		candidate := rawCandidates[i]
		candidate.PreselectionStatus = preselectionStatus(candidate, inventory.HuntCoverageComplete)
		if candidate.PreselectionStatus == "owner_adjudication_candidate" {
			inventory.PreselectionCandidates = append(inventory.PreselectionCandidates, candidate)
			inventory.Candidates = append(inventory.Candidates, candidate)
		} else {
			inventory.UnrankedCandidates = append(inventory.UnrankedCandidates, candidate)
		}
	}
	inventory.PreselectionCandidateCount = len(inventory.PreselectionCandidates)
	sort.Slice(inventory.PreselectionCandidates, func(i, j int) bool {
		if inventory.PreselectionCandidates[i].BurdenScore != inventory.PreselectionCandidates[j].BurdenScore {
			return inventory.PreselectionCandidates[i].BurdenScore > inventory.PreselectionCandidates[j].BurdenScore
		}
		return inventory.PreselectionCandidates[i].ID < inventory.PreselectionCandidates[j].ID
	})
	sort.Slice(inventory.Scenarios, func(i, j int) bool { return inventory.Scenarios[i].ID < inventory.Scenarios[j].ID })
	sortCandidates(inventory.Candidates)
	sortCandidates(inventory.PreselectionCandidates)
	sort.Slice(inventory.UnrankedCandidates, func(i, j int) bool { return inventory.UnrankedCandidates[i].ID < inventory.UnrankedCandidates[j].ID })
	return inventory, nil
}

func sortCandidates(candidates []Candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].BurdenScore != candidates[j].BurdenScore {
			return candidates[i].BurdenScore > candidates[j].BurdenScore
		}
		return candidates[i].ID < candidates[j].ID
	})
}

func preselectionStatus(candidate Candidate, huntCoverageComplete bool) string {
	if !huntCoverageComplete {
		return "unknown_hunt_coverage"
	}
	if len(candidate.HuntRecords) > 0 {
		return "prior_hunt_review_required"
	}
	if !candidate.Persistence.ScanComplete || candidate.Persistence.Class != ClassPersistedLifecycle {
		return "unknown_or_not_source_proven_persisted"
	}
	if len(candidate.States) < 2 || len(candidate.WriteSites) == 0 || len(candidate.TransitionOwners) == 0 {
		return "source_state_machine_unconfirmed"
	}
	return "owner_adjudication_candidate"
}

func analyzeScenario(ctx context.Context, root, scenarioRoot, scenarioID string) (ScenarioSummary, []Candidate, error) {
	summary := ScenarioSummary{ID: scenarioID}
	flowContracts, flowCount, flowParseErrors := readFlowContracts(root, scenarioRoot)
	summary.ExplicitFlows = flowCount
	summary.FlowParseErrors = flowParseErrors
	packages := map[string]*packageFacts{}
	for _, sourceRoot := range []string{"api", "cli", "ui/src"} {
		walkRoot := filepath.Join(scenarioRoot, sourceRoot)
		if _, err := os.Stat(walkRoot); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return summary, nil, err
		}
		err := filepath.WalkDir(walkRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				if path != walkRoot && skipDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_generated.go") || strings.HasSuffix(path, ".pb.go") {
				return nil
			}
			if strings.Contains(path, string(filepath.Separator)+"generated"+string(filepath.Separator)) {
				return nil
			}
			summary.SourceFiles++
			fset := token.NewFileSet()
			source, err := os.ReadFile(path)
			if err != nil {
				summary.ParseErrors++
				return nil
			}
			file, err := parser.ParseFile(fset, path, source, parser.ParseComments|parser.AllErrors)
			if err != nil {
				summary.ParseErrors++
				return nil
			}
			pkgPath := filepath.Dir(path)
			facts := packages[pkgPath]
			if facts == nil {
				facts = &packageFacts{path: pkgPath, consts: map[string][]StateEvidence{}, structs: map[string]map[string]fieldFact{}, structFields: map[string]map[string]string{}, writeSites: map[string][]Evidence{}, guardSites: map[string][]Evidence{}, tests: map[string][]string{}, owners: map[string][]Evidence{}, ownerTestTypes: map[string][]string{}}
				packages[pkgPath] = facts
			}
			relativePath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			aliases := map[string]bool{}
			localImportPath := packageImportPath(root, pkgPath)
			for _, imported := range file.Imports {
				importPath, err := strconv.Unquote(imported.Path.Value)
				if err != nil || localImportPath == "" || importPath != localImportPath {
					continue
				}
				alias := filepath.Base(importPath)
				if imported.Name != nil {
					alias = imported.Name.Name
				}
				if alias != "." && alias != "_" {
					aliases[alias] = true
				}
			}
			facts.files = append(facts.files, parsedFile{path: filepath.ToSlash(relativePath), source: string(source), file: file, fset: fset, test: strings.HasSuffix(path, "_test.go"), localPackageAliases: aliases})
			return nil
		})
		if err != nil {
			return summary, nil, fmt.Errorf("lifecycle inventory: walk %s: %w", walkRoot, err)
		}
	}

	var candidates []Candidate
	for _, facts := range packages {
		collectDeclarations(facts)
		collectUses(facts)
		for typeName, fields := range facts.structs {
			for fieldName, field := range fields {
				if !isLifecycleField(fieldName) {
					continue
				}
				candidate := Candidate{
					ID:               filepath.ToSlash(filepath.Join(scenarioID, strings.TrimPrefix(facts.path, filepath.Join(root, "scenarios", scenarioID)+string(filepath.Separator)))) + "#" + typeName + "." + fieldName,
					ScenarioID:       scenarioID,
					Package:          filepath.ToSlash(strings.TrimPrefix(facts.path, filepath.Join(root, "scenarios", scenarioID)+string(filepath.Separator))),
					TypeName:         typeName,
					FieldName:        fieldName,
					FieldType:        field.typeName,
					Declaration:      field.decl,
					RelatedFlags:     append([]Evidence(nil), field.flags...),
					TimeFields:       append([]Evidence(nil), field.times...),
					States:           append([]StateEvidence(nil), facts.consts[field.typeName]...),
					WriteSites:       append([]Evidence(nil), facts.writeSites[typeName+"."+fieldName]...),
					GuardSites:       append([]Evidence(nil), facts.guardSites[typeName+"."+fieldName]...),
					TransitionOwners: append([]Evidence(nil), facts.owners[typeName+"."+fieldName]...),
					TestFiles:        candidateTestFiles(facts, typeName+"."+fieldName, fieldName),
				}
				candidate.BurdenScore, candidate.ScoreBreakdown = score(candidate)
				candidate.Confidence = confidence(candidate)
				candidate.ExplicitFlows = matchFlowContracts(candidate, flowContracts)
				for _, flow := range candidate.ExplicitFlows {
					candidate.ExplicitlyModeled = candidate.ExplicitlyModeled || flow.ExactLifecycleMatch
				}
				candidates = append(candidates, candidate)
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	summary.CandidateCount = len(candidates)
	return summary, candidates, nil
}

// packageImportPath returns the module-qualified import path for a directory
// by finding its nearest go.mod. This lets external test packages refer back
// to the exact local package without confusing a similarly named dependency.
func packageImportPath(root, packageDir string) string {
	for dir := packageDir; ; dir = filepath.Dir(dir) {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil {
			module := ""
			for _, line := range strings.Split(string(data), "\n") {
				fields := strings.Fields(line)
				if len(fields) == 2 && fields[0] == "module" {
					module = fields[1]
					break
				}
			}
			if module == "" {
				return ""
			}
			relative, err := filepath.Rel(dir, packageDir)
			if err != nil || relative == "." {
				return module
			}
			return strings.TrimRight(module, "/") + "/" + filepath.ToSlash(relative)
		}
		if dir == root {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return ""
}

func collectDeclarations(facts *packageFacts) {
	for _, parsed := range facts.files {
		if parsed.test {
			continue
		}
		for _, decl := range parsed.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gen.Specs {
				switch item := spec.(type) {
				case *ast.TypeSpec:
					st, ok := item.Type.(*ast.StructType)
					if !ok {
						continue
					}
					fields := map[string]fieldFact{}
					allFields := map[string]string{}
					for _, field := range st.Fields.List {
						fieldType := exprName(field.Type)
						for _, name := range field.Names {
							allFields[name.Name] = fieldType
							text := sourceFragment(parsed, field)
							evidence := Evidence{Path: filepath.ToSlash(parsed.path), Line: parsed.fset.Position(name.Pos()).Line, Text: text}
							fact := fieldFact{typeName: fieldType, decl: evidence}
							if isLifecycleField(name.Name) {
								for _, sibling := range st.Fields.List {
									for _, siblingName := range sibling.Names {
										if isLifecycleFlag(siblingName.Name) && exprName(sibling.Type) == "bool" {
											fact.flags = append(fact.flags, Evidence{Path: filepath.ToSlash(parsed.path), Line: parsed.fset.Position(siblingName.Pos()).Line, Text: sourceFragment(parsed, sibling)})
										}
										if isTimeField(siblingName.Name, exprName(sibling.Type)) {
											fact.times = append(fact.times, Evidence{Path: filepath.ToSlash(parsed.path), Line: parsed.fset.Position(siblingName.Pos()).Line, Text: sourceFragment(parsed, sibling)})
										}
									}
								}
								fields[name.Name] = fact
							}
						}
					}
					facts.structs[item.Name.Name] = fields
					facts.structFields[item.Name.Name] = allFields
				case *ast.ValueSpec:
					if gen.Tok != token.CONST || item.Type == nil {
						continue
					}
					typ := exprName(item.Type)
					for i, name := range item.Names {
						if i >= len(item.Values) {
							continue
						}
						value, ok := stringValue(item.Values[i])
						if !ok {
							continue
						}
						facts.consts[typ] = append(facts.consts[typ], StateEvidence{Name: name.Name, Value: value, At: Evidence{Path: filepath.ToSlash(parsed.path), Line: parsed.fset.Position(name.Pos()).Line, Text: sourceFragment(parsed, item)}})
					}
				}
			}
		}
	}
	// Resolve transition owners after every type declaration in the package is
	// known. Receiver methods write the persisted owner directly. Pure transition
	// functions use a typed state value and return that same state type, so map
	// their evidence back to the persisted record by the transition function's
	// domain name and matching lifecycle field type.
	for _, parsed := range facts.files {
		if parsed.test {
			continue
		}
		for _, decl := range parsed.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			var owners []transitionOwnerField
			if isTransitionOwner(fn) {
				owners = transitionOwnerFields(fn)
			}
			owners = append(owners, functionalTransitionOwnerFields(fn, facts.structs, facts.structFields)...)
			for _, owner := range owners {
				key := owner.TypeName + "." + owner.FieldName
				if owner.TestTypeName != "" {
					facts.ownerTestTypes[key] = appendUniqueString(facts.ownerTestTypes[key], owner.TestTypeName)
				}
				facts.owners[key] = appendUniqueEvidence(facts.owners[key], Evidence{
					Path: filepath.ToSlash(parsed.path), Line: parsed.fset.Position(fn.Pos()).Line,
					Text: sourceFragment(parsed, fn),
				})
			}
		}
	}
}

type transitionOwnerField struct {
	TypeName     string
	FieldName    string
	TestTypeName string
}

func isTransitionOwner(fn *ast.FuncDecl) bool {
	if fn == nil || fn.Recv == nil || fn.Body == nil {
		return false
	}
	return strings.HasPrefix(strings.ToLower(fn.Name.Name), "transition")
}

func transitionOwnerFields(fn *ast.FuncDecl) []transitionOwnerField {
	if fn == nil || fn.Recv == nil {
		return nil
	}
	owners := make([]transitionOwnerField, 0)
	for _, receiver := range fn.Recv.List {
		typeName := exprName(receiver.Type)
		for _, receiverName := range receiver.Names {
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				assignment, ok := node.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for _, left := range assignment.Lhs {
					selector, ok := left.(*ast.SelectorExpr)
					if !ok || !isLifecycleField(selector.Sel.Name) {
						continue
					}
					base, ok := selector.X.(*ast.Ident)
					if !ok || base.Name != receiverName.Name {
						continue
					}
					owners = append(owners, transitionOwnerField{TypeName: typeName, FieldName: selector.Sel.Name})
				}
				return true
			})
		}
	}
	return owners
}

// functionalTransitionOwnerFields recognizes a pure, typed transition owner
// such as TransitionDraft(DraftState, DraftEvent) (DraftState, error). It is
// intentionally narrower than a name-only match: the input and output must be
// the same declared state struct, and the function must read one of that
// struct's lifecycle fields. The domain name suffix maps the state value back
// to its persisted owner (DraftState -> Draft).
func functionalTransitionOwnerFields(fn *ast.FuncDecl, structs map[string]map[string]fieldFact, structFields map[string]map[string]string) []transitionOwnerField {
	if fn == nil || fn.Recv != nil || fn.Body == nil || fn.Type.Params == nil || fn.Type.Results == nil || !strings.HasPrefix(fn.Name.Name, "Transition") {
		return nil
	}
	params := map[string]string{}
	for _, field := range fn.Type.Params.List {
		typeName := exprName(field.Type)
		if _, ok := structFields[typeName]; !ok {
			continue
		}
		for _, name := range field.Names {
			params[name.Name] = typeName
		}
	}
	stateTypes := map[string]bool{}
	for _, field := range fn.Type.Results.List {
		if typeName := exprName(field.Type); paramsForType(params, typeName) {
			stateTypes[typeName] = true
		}
	}
	if len(stateTypes) == 0 {
		return nil
	}
	var inputFields []transitionOwnerField
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || !isLifecycleField(selector.Sel.Name) {
			return true
		}
		base, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		stateType := params[base.Name]
		if !stateTypes[stateType] || structFields[stateType][selector.Sel.Name] == "" {
			return true
		}
		inputFields = append(inputFields, transitionOwnerField{TypeName: stateType, FieldName: selector.Sel.Name})
		return true
	})
	if len(inputFields) == 0 {
		return nil
	}
	transitionName := strings.TrimPrefix(fn.Name.Name, "Transition")
	var owners []transitionOwnerField
	for stateType := range structFields {
		if !stateTypes[stateType] {
			continue
		}
		for _, stateField := range inputFields {
			stateFieldType := structFields[stateField.TypeName][stateField.FieldName]
			for ownerType, ownerFields := range structs {
				if !strings.HasSuffix(strings.ToLower(transitionName), strings.ToLower(ownerType)) || ownerFields[stateField.FieldName].typeName != stateFieldType {
					continue
				}
				owners = append(owners, transitionOwnerField{TypeName: ownerType, FieldName: stateField.FieldName, TestTypeName: stateType})
			}
		}
	}
	return owners
}

func candidateTestFiles(facts *packageFacts, key, fieldName string) []string {
	files := append([]string(nil), facts.tests[key]...)
	for _, testType := range facts.ownerTestTypes[key] {
		for _, path := range facts.tests[testType+"."+fieldName] {
			files = appendUniqueString(files, path)
		}
	}
	sort.Strings(files)
	return files
}

func paramsForType(params map[string]string, typeName string) bool {
	for _, paramType := range params {
		if paramType == typeName {
			return true
		}
	}
	return false
}

func collectUses(facts *packageFacts) {
	for _, parsed := range facts.files {
		for _, decl := range parsed.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			types := functionTypes(fn, facts.structFields, parsed.localPackageAliases)
			collectLocalTypes(fn.Body, types, facts.structFields, parsed.localPackageAliases)
			writes := map[token.Pos]struct{}{}
			guards := map[token.Pos]struct{}{}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				switch stmt := node.(type) {
				case *ast.AssignStmt:
					for _, lhs := range stmt.Lhs {
						if selector, ok := lhs.(*ast.SelectorExpr); ok && isLifecycleField(selector.Sel.Name) {
							writes[selector.Pos()] = struct{}{}
						}
					}
				case *ast.IncDecStmt:
					if selector, ok := stmt.X.(*ast.SelectorExpr); ok && isLifecycleField(selector.Sel.Name) {
						writes[selector.Pos()] = struct{}{}
					}
				case *ast.BinaryExpr:
					if stmt.Op == token.EQL || stmt.Op == token.NEQ {
						for _, expr := range []ast.Expr{stmt.X, stmt.Y} {
							if selector, ok := expr.(*ast.SelectorExpr); ok && isLifecycleField(selector.Sel.Name) {
								guards[selector.Pos()] = struct{}{}
							}
						}
					}
				}
				return true
			})
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok || !isLifecycleField(selector.Sel.Name) {
					return true
				}
				typeName := exprType(selector.X, types, facts.structFields)
				typeName = resolveLocalAlias(typeName, parsed.localPackageAliases)
				key := typeName + "." + selector.Sel.Name
				if _, candidate := facts.structs[typeName][selector.Sel.Name]; !candidate {
					return true
				}
				ev := Evidence{Path: filepath.ToSlash(parsed.path), Line: parsed.fset.Position(selector.Pos()).Line, Text: sourceFragment(parsed, selector)}
				if _, ok := writes[selector.Pos()]; ok {
					facts.writeSites[key] = appendUniqueEvidence(facts.writeSites[key], ev)
				} else if _, ok := guards[selector.Pos()]; ok {
					facts.guardSites[key] = appendUniqueEvidence(facts.guardSites[key], ev)
				}
				return true
			})
			if parsed.test {
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					switch expr := node.(type) {
					case *ast.SelectorExpr:
						if !isLifecycleField(expr.Sel.Name) {
							return true
						}
						typeName := resolveLocalAlias(exprType(expr.X, types, facts.structFields), parsed.localPackageAliases)
						key := typeName + "." + expr.Sel.Name
						if _, candidate := facts.structs[typeName][expr.Sel.Name]; candidate {
							facts.tests[key] = appendUniqueString(facts.tests[key], filepath.ToSlash(parsed.path))
						}
					case *ast.CompositeLit:
						typeName := resolveLocalAlias(exprName(expr.Type), parsed.localPackageAliases)
						for _, element := range expr.Elts {
							pair, ok := element.(*ast.KeyValueExpr)
							if !ok {
								continue
							}
							field, ok := pair.Key.(*ast.Ident)
							if !ok || !isLifecycleField(field.Name) {
								continue
							}
							if _, candidate := facts.structs[typeName][field.Name]; candidate {
								key := typeName + "." + field.Name
								facts.tests[key] = appendUniqueString(facts.tests[key], filepath.ToSlash(parsed.path))
							}
						}
					}
					return true
				})
			}
		}
	}
}

func functionTypes(fn *ast.FuncDecl, structFields map[string]map[string]string, aliases map[string]bool) map[string]string {
	types := map[string]string{}
	add := func(fields *ast.FieldList) {
		if fields == nil {
			return
		}
		for _, field := range fields.List {
			name := resolveLocalAlias(exprName(field.Type), aliases)
			for _, ident := range field.Names {
				types[ident.Name] = name
			}
		}
	}
	if fn.Recv != nil {
		add(fn.Recv)
	}
	if fn.Type != nil {
		add(fn.Type.Params)
	}
	collectLocalTypes(fn.Body, types, structFields, aliases)
	return types
}

func collectLocalTypes(node ast.Node, types map[string]string, structFields map[string]map[string]string, aliases map[string]bool) {
	ast.Inspect(node, func(node ast.Node) bool {
		switch stmt := node.(type) {
		case *ast.ValueSpec:
			for i, name := range stmt.Names {
				if stmt.Type != nil {
					types[name.Name] = resolveLocalAlias(exprName(stmt.Type), aliases)
				} else if i < len(stmt.Values) {
					if inferred := resolveLocalAlias(exprType(stmt.Values[i], types, structFields), aliases); inferred != "" {
						types[name.Name] = inferred
					}
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range stmt.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || i >= len(stmt.Rhs) {
					continue
				}
				if inferred := resolveLocalAlias(exprType(stmt.Rhs[i], types, structFields), aliases); inferred != "" {
					types[ident.Name] = inferred
				}
			}
		}
		return true
	})
}

// resolveLocalAlias maps an imported external-test-package alias back to the
// package-local type name when it imports the package being inventoried.
func resolveLocalAlias(typeName string, aliases map[string]bool) string {
	separator := strings.IndexByte(typeName, '.')
	if separator > 0 && aliases[typeName[:separator]] {
		return typeName[separator+1:]
	}
	return typeName
}

func exprType(expr ast.Expr, types map[string]string, structFields map[string]map[string]string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if typ := types[e.Name]; typ != "" {
			return typ
		}
		if _, ok := structFields[e.Name]; ok {
			return e.Name
		}
	case *ast.StarExpr:
		return exprType(e.X, types, structFields)
	case *ast.CompositeLit:
		return exprName(e.Type)
	case *ast.UnaryExpr:
		return exprType(e.X, types, structFields)
	case *ast.IndexExpr:
		return strings.TrimPrefix(exprType(e.X, types, structFields), "[]")
	case *ast.SelectorExpr:
		base := exprType(e.X, types, structFields)
		if fieldType := structFields[base][e.Sel.Name]; fieldType != "" {
			return fieldType
		}
	}
	return ""
}

func score(c Candidate) (int, []string) {
	score := 1
	breakdown := []string{"base candidate +1"}
	if n := len(c.States); n > 0 {
		score += n * 3
		breakdown = append(breakdown, fmt.Sprintf("%d typed state constants × 3", n))
	}
	if n := len(c.RelatedFlags); n > 0 {
		score += n * 4
		breakdown = append(breakdown, fmt.Sprintf("%d lifecycle booleans in owner struct × 4", n))
	}
	if n := len(c.TimeFields); n > 0 {
		score += n
		breakdown = append(breakdown, fmt.Sprintf("%d time fields in owner struct", n))
	}
	if n := len(c.WriteSites); n > 0 {
		score += n * 2
		breakdown = append(breakdown, fmt.Sprintf("%d field writes × 2", n))
	}
	if n := len(c.GuardSites); n > 0 {
		score += n
		breakdown = append(breakdown, fmt.Sprintf("%d equality guards", n))
	}
	if n := len(c.TestFiles); n > 0 {
		breakdown = append(breakdown, fmt.Sprintf("covered by %d test files", n))
	}
	return score, breakdown
}

func confidence(c Candidate) string {
	if len(c.States) >= 2 && len(c.WriteSites) > 0 && len(c.GuardSites) > 0 {
		return "strong-candidate"
	}
	if len(c.States) > 0 || len(c.WriteSites) > 0 || len(c.GuardSites) > 0 {
		return "candidate-needs-review"
	}
	return "field-only"
}

func isLifecycleField(name string) bool {
	lower := strings.ToLower(name)
	return lower == "status" || lower == "state" || lower == "lifecyclestate" || lower == "outcome"
}

func isLifecycleFlag(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range []string{"complete", "done", "failed", "cancel", "pause", "abandon", "active", "ready", "terminal", "retry"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isTimeField(name, typ string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(typ, "time.Time") || strings.HasSuffix(lower, "at") || strings.Contains(lower, "timestamp") || strings.Contains(lower, "deadline")
}

func exprName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprName(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return exprName(e.X)
	case *ast.ArrayType:
		return "[]" + exprName(e.Elt)
	}
	return ""
}

func stringValue(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconvUnquote(lit.Value)
	return value, err == nil
}

func sourceFragment(parsed parsedFile, node ast.Node) string {
	start := parsed.fset.Position(node.Pos()).Line
	end := parsed.fset.Position(node.End()).Line
	lines := strings.Split(parsed.source, "\n")
	if start < 1 || start > len(lines) {
		return ""
	}
	if end > len(lines) {
		end = len(lines)
	}
	if end-start > 2 {
		end = start + 2
	}
	return strings.TrimSpace(strings.Join(lines[start-1:end], " "))
}

func appendUniqueEvidence(rows []Evidence, item Evidence) []Evidence {
	for _, row := range rows {
		if row.Path == item.Path && row.Line == item.Line {
			return rows
		}
	}
	return append(rows, item)
}

func appendUniqueString(rows []string, item string) []string {
	for _, row := range rows {
		if row == item {
			return rows
		}
	}
	return append(rows, item)
}

func skipDir(name string) bool {
	return name == "vendor" || name == "node_modules" || name == "generated" || strings.HasPrefix(name, ".") || name == "coverage"
}

type flowContract struct {
	ID      string `json:"flowId"`
	Runtime struct {
		Go struct {
			StatusType string `json:"statusType"`
		} `json:"go"`
	} `json:"runtime"`
	States []struct {
		ID string `json:"id"`
	} `json:"states"`
	Path string `json:"-"`
}

func readFlowContracts(root, scenarioRoot string) ([]flowContract, int, int) {
	var contracts []flowContract
	files, parseErrors := 0, 0
	_ = filepath.WalkDir(scenarioRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() && path != scenarioRoot && skipDir(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Base(path) != "flow.json" {
			return nil
		}
		files++
		data, err := os.ReadFile(path)
		if err != nil {
			parseErrors++
			return nil
		}
		var contract flowContract
		if err := json.Unmarshal(data, &contract); err != nil {
			parseErrors++
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			parseErrors++
			return nil
		}
		contract.Path = filepath.ToSlash(rel)
		contracts = append(contracts, contract)
		return nil
	})
	return contracts, files, parseErrors
}

func matchFlowContracts(candidate Candidate, contracts []flowContract) []FlowReference {
	var matches []FlowReference
	for _, contract := range contracts {
		if strings.TrimSpace(contract.Runtime.Go.StatusType) == "" || contract.Runtime.Go.StatusType != candidate.FieldType {
			continue
		}
		stateValues := make([]string, 0, len(candidate.States))
		for _, state := range candidate.States {
			stateValues = appendUniqueString(stateValues, state.Value)
		}
		contractStates := make([]string, 0, len(contract.States))
		for _, state := range contract.States {
			contractStates = appendUniqueString(contractStates, state.ID)
		}
		matched := make([]string, 0)
		for _, state := range stateValues {
			if slices.Contains(contractStates, state) {
				matched = append(matched, state)
			}
		}
		matches = append(matches, FlowReference{
			FlowID:              contract.ID,
			Path:                contract.Path,
			StatusType:          contract.Runtime.Go.StatusType,
			MatchedStates:       matched,
			CandidateStateCount: len(stateValues),
			ContractStateCount:  len(contractStates),
			ExactLifecycleMatch: len(stateValues) > 0 && len(stateValues) == len(contractStates) && len(matched) == len(stateValues),
		})
	}
	return matches
}

type huntIndex struct {
	Scope      string   `json:"scope"`
	Complete   *bool    `json:"complete,omitempty"`
	Degraded   []string `json:"degraded,omitempty"`
	EntryCount int      `json:"entry_count,omitempty"`
	NextCursor *string  `json:"next_cursor,omitempty"`
	Coverage   struct {
		Complete   bool     `json:"complete"`
		Degraded   []string `json:"degraded"`
		EntryCount int      `json:"entry_count"`
		NextCursor *string  `json:"next_cursor"`
	} `json:"coverage,omitempty"`
	Hits []struct {
		EntryID      string   `json:"entry_id"`
		Text         string   `json:"text"`
		CandidateIDs []string `json:"candidate_ids,omitempty"`
	} `json:"hits"`
	Entries []struct {
		ID           string   `json:"id"`
		Body         string   `json:"body"`
		CandidateIDs []string `json:"candidate_ids,omitempty"`
	} `json:"entries,omitempty"`
	CandidateBindings []struct {
		EntryID      string   `json:"entry_id"`
		CandidateIDs []string `json:"candidate_ids"`
	} `json:"candidate_bindings,omitempty"`
}

func (index huntIndex) coverageComplete() bool {
	if index.Scope != "ecosystem-hunt" {
		return false
	}
	complete := index.Coverage.Complete
	entryCount := index.Coverage.EntryCount
	degraded := index.Coverage.Degraded
	nextCursor := index.Coverage.NextCursor
	if index.Complete != nil {
		complete = *index.Complete
		entryCount = index.EntryCount
		degraded = index.Degraded
		nextCursor = index.NextCursor
	}
	return complete && entryCount > 0 && len(degraded) == 0 && nextCursor == nil
}

func readHuntIndex(opts Options) (huntIndex, error) {
	data := opts.HuntIndexJSON
	if len(data) == 0 && opts.HuntIndex == "" {
		return huntIndex{}, nil
	}
	if len(data) == 0 {
		var err error
		data, err = os.ReadFile(opts.HuntIndex)
		if err != nil {
			return huntIndex{}, fmt.Errorf("lifecycle inventory: read hunt index: %w", err)
		}
	}
	var index huntIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return huntIndex{}, fmt.Errorf("lifecycle inventory: parse hunt index: %w", err)
	}
	bindings := make(map[string][]string, len(index.CandidateBindings))
	for _, binding := range index.CandidateBindings {
		bindings[binding.EntryID] = append(bindings[binding.EntryID], binding.CandidateIDs...)
	}
	for _, entry := range index.Entries {
		index.Hits = append(index.Hits, struct {
			EntryID      string   `json:"entry_id"`
			Text         string   `json:"text"`
			CandidateIDs []string `json:"candidate_ids,omitempty"`
		}{EntryID: entry.ID, Text: entry.Body, CandidateIDs: append(entry.CandidateIDs, bindings[entry.ID]...)})
	}
	for i := range index.Hits {
		index.Hits[i].CandidateIDs = append(index.Hits[i].CandidateIDs, bindings[index.Hits[i].EntryID]...)
	}
	return index, nil
}

func matchHunt(candidate Candidate, index huntIndex) []HuntReference {
	scenarioPhrase := strings.ReplaceAll(strings.ToLower(candidate.ScenarioID), "-", " ")
	typePhrase := humanize(candidate.TypeName)
	if typePhrase == "" {
		return nil
	}
	var matches []HuntReference
	for _, hit := range index.Hits {
		text := strings.ToLower(hit.Text)
		exactBinding := slices.Contains(hit.CandidateIDs, candidate.ID)
		textualMatch := len(hit.CandidateIDs) == 0 && (strings.Contains(text, scenarioPhrase) || strings.Contains(text, candidate.ScenarioID)) && hasSpecificTypeTokens(text, typePhrase)
		if !exactBinding && !textualMatch {
			continue
		}
		round := ""
		for _, line := range strings.Split(hit.Text, "\n") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "round") && strings.Contains(lower, "#") {
				round = strings.TrimSpace(strings.TrimLeft(line, "# "))
				break
			}
		}
		matches = append(matches, HuntReference{EntryID: hit.EntryID, Round: round, Status: "review"})
	}
	return matches
}

func humanize(identifier string) string {
	var out strings.Builder
	for i, r := range identifier {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out.WriteByte(' ')
		}
		out.WriteRune(r)
	}
	return strings.ToLower(out.String())
}

func hasSpecificTypeTokens(text, typePhrase string) bool {
	ignored := map[string]struct{}{"state": {}, "status": {}, "record": {}, "model": {}, "config": {}, "request": {}, "response": {}, "summary": {}, "result": {}, "data": {}, "info": {}, "execution": {}, "workflow": {}, "run": {}, "phase": {}, "item": {}, "entry": {}, "event": {}, "job": {}, "task": {}, "operation": {}}
	tokens := strings.Fields(typePhrase)
	specific := 0
	matched := 0
	for _, token := range tokens {
		if _, ok := ignored[token]; ok || len(token) < 4 {
			continue
		}
		specific++
		if strings.Contains(text, token) {
			matched++
		}
	}
	return specific > 0 && matched == specific
}

// strconvUnquote is an aliasable seam kept local to avoid leaking parser
// details through the public inventory surface.
var strconvUnquote = func(value string) (string, error) {
	return strconv.Unquote(value)
}
