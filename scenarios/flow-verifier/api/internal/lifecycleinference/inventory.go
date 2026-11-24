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
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
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

type CandidateKind string

const CandidateKindImplicitSignal CandidateKind = "implicit_signal"

type Candidate struct {
	Kind                    CandidateKind             `json:"kind,omitempty"`
	ID                      string                    `json:"id"`
	ScenarioID              string                    `json:"scenarioId"`
	Package                 string                    `json:"package"`
	Language                string                    `json:"language,omitempty"`
	TypeName                string                    `json:"typeName"`
	FieldName               string                    `json:"fieldName"`
	FieldType               string                    `json:"fieldType"`
	Declaration             Evidence                  `json:"declaration"`
	States                  []StateEvidence           `json:"states,omitempty"`
	RelatedFlags            []Evidence                `json:"relatedFlags,omitempty"`
	TimeFields              []Evidence                `json:"timeFields,omitempty"`
	WriteSites              []Evidence                `json:"writeSites,omitempty"`
	GuardSites              []Evidence                `json:"guardSites,omitempty"`
	TransitionOwners        []Evidence                `json:"transitionOwners,omitempty"`
	TestFiles               []string                  `json:"testFiles,omitempty"`
	HuntRecords             []HuntReference           `json:"huntRecords,omitempty"`
	Persistence             PersistenceClassification `json:"persistence"`
	PreselectionStatus      string                    `json:"preselectionStatus"`
	ExplicitFlows           []FlowReference           `json:"explicitFlows,omitempty"`
	ExplicitlyModeled       bool                      `json:"explicitlyModeled,omitempty"`
	TransitionGraphVerified bool                      `json:"transitionGraphVerified,omitempty"`
	InvalidTransitionTested bool                      `json:"invalidTransitionTested,omitempty"`
	BurdenScore             int                       `json:"burdenScore"`
	ScoreBreakdown          []string                  `json:"scoreBreakdown"`
	Confidence              string                    `json:"confidence"`
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
	// ScenarioID optionally limits source parsing and classification to one
	// service-backed scenario. It is intended for focused owner review; an
	// empty value retains the fleet-wide inventory contract.
	ScenarioID string
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
		if opts.ScenarioID != "" && entry.Name() != opts.ScenarioID {
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
		if err := classifyCandidates(ctx, candidates, index, persistenceClassifier); err != nil {
			return Inventory{}, err
		}
		inventory.Candidates = append(inventory.Candidates, candidates...)
	}
	rawCandidates := inventory.Candidates
	inventory.Candidates = make([]Candidate, 0, len(rawCandidates))
	for i := range rawCandidates {
		candidate := rawCandidates[i]
		if candidate.Kind != "implicit_signal" {
			candidate.PreselectionStatus = preselectionStatus(candidate, inventory.HuntCoverageComplete)
		}
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

// classifyCandidates shares the scenario source snapshot while distributing
// candidate-specific AST flow analysis across available CPUs. Large scenarios
// commonly have hundreds of candidate fields; serially rescanning their ASTs
// consumed most of the two-minute API operation budget.
func classifyCandidates(ctx context.Context, candidates []Candidate, index huntIndex, classifier *PersistenceClassifier) error {
	if len(candidates) == 0 {
		return nil
	}
	workers := min(runtime.GOMAXPROCS(0), len(candidates), 8)
	if workers < 2 {
		for i := range candidates {
			if err := ctx.Err(); err != nil {
				return err
			}
			candidates[i].HuntRecords = matchHunt(candidates[i], index)
			if candidates[i].Kind != "implicit_signal" && (candidates[i].Language == "" || candidates[i].Language == "go") {
				candidates[i].Persistence = classifier.ClassifyContext(ctx, candidates[i])
			}
		}
		return ctx.Err()
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if ctx.Err() != nil {
					continue
				}
				candidates[i].HuntRecords = matchHunt(candidates[i], index)
				if candidates[i].Kind != "implicit_signal" && (candidates[i].Language == "" || candidates[i].Language == "go") {
					candidates[i].Persistence = classifier.ClassifyContext(ctx, candidates[i])
				}
			}
		}()
	}
	for i := range candidates {
		if ctx.Err() != nil {
			break
		}
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return ctx.Err()
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
	if candidate.TransitionGraphVerified && candidate.InvalidTransitionTested {
		return "already_explicitly_modeled"
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
		candidates = append(candidates, implicitSignalCandidates(facts, scenarioID)...)
		for typeName, fields := range facts.structs {
			for fieldName, field := range fields {
				if !isLifecycleField(fieldName) {
					continue
				}
				candidate := Candidate{
					ID:               filepath.ToSlash(filepath.Join(scenarioID, strings.TrimPrefix(facts.path, filepath.Join(root, "scenarios", scenarioID)+string(filepath.Separator)))) + "#" + typeName + "." + fieldName,
					ScenarioID:       scenarioID,
					Language:         "go",
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
				candidate.TransitionGraphVerified, candidate.InvalidTransitionTested = explicitTransitionProof(facts, typeName+"."+fieldName)
				if owner, ok := databaseTransitionOwner(facts, typeName, fieldName); ok {
					candidate.TransitionOwners = appendUniqueEvidence(candidate.TransitionOwners, owner)
					candidate.TransitionGraphVerified = true
					candidate.InvalidTransitionTested = true
				}
				candidates = append(candidates, candidate)
			}
		}
	}
	tsCandidates, tsSourceFiles, tsParseErrors := analyzeTypeScriptScenario(scenarioRoot, scenarioID)
	summary.SourceFiles += tsSourceFiles
	summary.ParseErrors += tsParseErrors
	for i := range tsCandidates {
		tsCandidates[i].ExplicitFlows = matchFlowContracts(tsCandidates[i], flowContracts)
		for _, flow := range tsCandidates[i].ExplicitFlows {
			tsCandidates[i].ExplicitlyModeled = tsCandidates[i].ExplicitlyModeled || flow.ExactLifecycleMatch
		}
		candidates = append(candidates, tsCandidates[i])
	}
	otherCandidates, otherSourceFiles, otherParseErrors := analyzePythonJavaScenario(scenarioRoot, scenarioID)
	summary.SourceFiles += otherSourceFiles
	summary.ParseErrors += otherParseErrors
	for i := range otherCandidates {
		otherCandidates[i].ExplicitFlows = matchFlowContracts(otherCandidates[i], flowContracts)
		for _, flow := range otherCandidates[i].ExplicitFlows {
			otherCandidates[i].ExplicitlyModeled = otherCandidates[i].ExplicitlyModeled || flow.ExactLifecycleMatch
		}
		candidates = append(candidates, otherCandidates[i])
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	summary.CandidateCount = len(candidates)
	return summary, candidates, nil
}

// implicitSignalCandidates reports a separate, non-persistence inference when
// a Go type couples multiple lifecycle-like booleans with temporal/retry
// evidence and has executable receiver methods. It is deliberately never
// eligible for persisted-lifecycle ranking: this lexical signal identifies a
// review target, not a durable state machine.
func implicitSignalCandidates(facts *packageFacts, scenarioID string) []Candidate {
	var out []Candidate
	for typeName := range facts.structFields {
		if _, explicit := facts.structs[typeName]["Status"]; explicit {
			continue
		}
		if _, explicit := facts.structs[typeName]["State"]; explicit {
			continue
		}
		var flags, temporal, retry, events []Evidence
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
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Name.Name != typeName {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range st.Fields.List {
						for _, name := range field.Names {
							typ, lower := exprName(field.Type), strings.ToLower(name.Name)
							ev := Evidence{Path: filepath.ToSlash(parsed.path), Line: parsed.fset.Position(name.Pos()).Line, Text: sourceFragment(parsed, field)}
							if typ == "bool" && isLifecycleFlag(name.Name) {
								flags = append(flags, ev)
							}
							if isTimeField(name.Name, typ) {
								temporal = append(temporal, ev)
							}
							if strings.Contains(lower, "retry") || strings.Contains(lower, "attempt") || strings.Contains(lower, "event") {
								retry = append(retry, ev)
							}
							if strings.HasPrefix(typ, "chan ") || strings.Contains(lower, "done") || strings.Contains(lower, "event") {
								events = append(events, ev)
							}
						}
					}
				}
			}
		}
		hasReceiver := false
		for _, parsed := range facts.files {
			for _, decl := range parsed.file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil && exprName(fn.Recv.List[0].Type) == typeName {
					hasReceiver = true
				}
			}
		}
		// A coupled flag set plus temporal field, retry/event evidence, and
		// behavior is the minimum useful signal. Mere configuration records
		// and single booleans therefore stay out of inventory.
		if len(flags) < 2 || (len(temporal) == 0 && len(events) == 0) || (len(retry) == 0 && len(events) == 0) || !hasReceiver {
			continue
		}
		decl := flags[0]
		reason := "implicit signal only; persistence and lifecycle semantics are unproven"
		if len(events) > 0 {
			reason += "; coordination event evidence: " + events[0].Path + ":" + strconv.Itoa(events[0].Line) + " " + events[0].Text
		}
		c := Candidate{Kind: CandidateKindImplicitSignal, ID: scenarioID + "#" + typeName + ".<implicit-lifecycle>", ScenarioID: scenarioID, Package: filepath.Base(facts.path), TypeName: typeName, FieldName: "<implicit-lifecycle>", FieldType: "coupled flags", Declaration: decl, RelatedFlags: flags, TimeFields: temporal, Persistence: PersistenceClassification{Class: ClassUnknown, Reason: reason, Scope: "not classified"}, PreselectionStatus: "implicit_signal_non_ranked", Confidence: "review-only-implicit-signal", ScoreBreakdown: []string{"candidate kind: implicit_signal", "coupled lifecycle-like flags", "temporal/event/retry evidence", "receiver behavior present", "persistence unproven"}}
		out = append(out, c)
	}
	return out
}

// explicitTransitionProof is deliberately conservative: a named transition
// method alone is only a lead. We recognize a completed explicit lifecycle
// only when the field-owning method visibly has a transition table, rejects
// invalid edges before mutation, and an owner test invokes that method while
// asserting its error. This keeps the inventory useful after conversions
// without hiding weak setters or untested state graphs.
func explicitTransitionProof(facts *packageFacts, key string) (bool, bool) {
	for _, owner := range facts.owners[key] {
		var text string
		for _, parsed := range facts.files {
			if parsed.path != owner.Path {
				continue
			}
			for _, decl := range parsed.file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if ok && fn.Body != nil && parsed.fset.Position(fn.Pos()).Line == owner.Line {
					text = fullSourceFragment(parsed, fn)
					break
				}
			}
		}
		if ownerIsTypedPolicy(text) {
			if !strings.Contains(text, "switch") && !hasFiniteBooleanTransitionExpression(text) {
				continue
			}
			if invalidPolicyBoundaryTest(facts, key) {
				return true, true
			}
			continue
		}
		delegatedGraphCall := calledTransitionHelperHasGraph(facts, text)
		hasReturnedError := strings.Contains(text, "return fmt.Errorf") || strings.Contains(text, ", fmt.Errorf") || strings.Contains(text, "return validationError") || strings.Contains(text, "return ErrInvalid") || strings.Contains(text, "return ErrIllegal") || strings.Contains(text, "return &") || delegatedGraphCall
		if !hasReturnedError {
			continue
		}
		lowerText := strings.ToLower(text)
		if !strings.Contains(lowerText, "transitions[") && !strings.Contains(text, "CanTransition") && !strings.Contains(text, "switch") && !strings.Contains(text, "Valid()") && !calledPredicateHasGraph(facts, text) && !delegatedGraphCall {
			continue
		}
		field := key[strings.Index(key, ".")+1:]
		rejection := strings.Index(text, "return fmt.Errorf")
		if rejection < 0 {
			rejection = strings.Index(text, ", fmt.Errorf")
		}
		if rejection < 0 {
			rejection = strings.Index(text, "return validationError")
		}
		if rejection < 0 {
			rejection = strings.Index(text, "return ErrInvalid")
		}
		if rejection < 0 {
			rejection = strings.Index(text, "return ErrIllegal")
		}
		if rejection < 0 {
			rejection = strings.Index(text, "return &")
		}
		if rejection < 0 && delegatedGraphCall {
			rejection = strings.Index(strings.ToLower(text), "return transition")
		}
		mutation := fieldAssignmentIndex(text, field)
		noMutationWithoutDelegation := mutation < 0 && !delegatedGraphCall && (!strings.Contains(text, ", nil") || !strings.Contains(text, "return "))
		if rejection < 0 || (mutation >= 0 && rejection > mutation) || noMutationWithoutDelegation {
			continue
		}
		method := transitionMethodName(owner.Text)
		if method == "" {
			continue
		}
		for _, parsed := range facts.files {
			if !parsed.test {
				continue
			}
			for _, decl := range parsed.file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil || !strings.Contains(fn.Name.Name, "Test") {
					continue
				}
				source := fullSourceFragment(parsed, fn)
				lowerSource := strings.ToLower(source)
				invalidCase := strings.Contains(lowerSource, "invalid") || strings.Contains(lowerSource, "illegal") || strings.Contains(lowerSource, "unknown") || (strings.Contains(source, "require.Error") && (strings.Contains(source, "isValid") || strings.Contains(source, "explicit") || strings.Contains(source, "allowed")))
				errorAssertion := strings.Contains(source, "require.Error") || strings.Contains(source, "assert.Error") || strings.Contains(source, "ErrorIs(") || strings.Contains(source, "errors.Is(") || strings.Contains(source, "errors.As(") || strings.Contains(source, "err != nil") || strings.Contains(source, "err == nil")
				if strings.Contains(source, method+"(") && invalidCase && errorAssertion {
					return true, true
				}
			}
		}
	}
	return false, false
}

func ownerIsTypedPolicy(source string) bool {
	if !strings.Contains(source, "CanTransitionTo(") || !strings.Contains(source, "bool") {
		return false
	}
	if strings.Contains(source, "switch") {
		return true
	}
	return hasFiniteBooleanTransitionExpression(source)
}

func hasFiniteBooleanTransitionExpression(source string) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "transition.go", "package lifecycleproof\n"+source, 0)
	if err != nil {
		return false
	}
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "CanTransitionTo" || fn.Body == nil {
			continue
		}
		equalities, conjunctions, alternatives := 0, 0, 0
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			binary, ok := node.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			switch binary.Op {
			case token.EQL:
				equalities++
			case token.LAND:
				conjunctions++
			case token.LOR:
				alternatives++
			}
			return true
		})
		return equalities >= 2 && conjunctions >= 1 && alternatives >= 1
	}
	return false
}

func invalidPolicyBoundaryTest(facts *packageFacts, key string) bool {
	typeName := key[:strings.Index(key, ".")]
	for _, parsed := range facts.files {
		if !parsed.test {
			continue
		}
		for _, decl := range parsed.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !strings.Contains(fn.Name.Name, "Test") {
				continue
			}
			source := fullSourceFragment(parsed, fn)
			lower := strings.ToLower(source)
			policyRejectsInvalid := strings.Contains(source, "CanTransitionTo(") &&
				(strings.Contains(source, "t.Fatal") || strings.Contains(source, "t.Errorf") || strings.Contains(source, "t.Fatalf") || strings.Contains(source, "require.False") || strings.Contains(source, "assert.False"))
			if (strings.Contains(lower, "unknown") || strings.Contains(lower, "invalid") || strings.Contains(lower, "stale")) &&
				((strings.Contains(source, "require.Error") || strings.Contains(source, "assert.Error") || strings.Contains(source, "err == nil") || strings.Contains(source, "err != nil")) &&
					(strings.Contains(source, "Update(") || strings.Contains(source, "Create(")) || policyRejectsInvalid) &&
				(strings.Contains(source, typeName) || strings.Contains(source, "State")) {
				return true
			}
		}
	}
	return false
}

// calledPredicateHasGraph accepts a transition owner that delegates its
// vocabulary check to a source-local predicate (for example
// knownPublicationState). The helper must itself contain a finite switch;
// method names alone never establish a graph.
func calledPredicateHasGraph(facts *packageFacts, source string) bool {
	for _, parsed := range facts.files {
		for _, decl := range parsed.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil || !(strings.HasPrefix(strings.ToLower(fn.Name.Name), "known") || strings.HasPrefix(strings.ToLower(fn.Name.Name), "valid")) {
				continue
			}
			if strings.Contains(source, fn.Name.Name+"(") && strings.Contains(fullSourceFragment(parsed, fn), "switch") {
				return true
			}
		}
	}
	return false
}

// calledTransitionHelperHasGraph accepts a field-owning receiver method that
// delegates validation and mutation of an addressed lifecycle field to a
// package-local helper. The helper must check a finite transition map and
// return an error on rejection; a call by name alone is not proof.
func calledTransitionHelperHasGraph(facts *packageFacts, source string) bool {
	for _, parsed := range facts.files {
		if parsed.test {
			continue
		}
		for _, decl := range parsed.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil || !strings.HasPrefix(strings.ToLower(fn.Name.Name), "transition") {
				continue
			}
			if !strings.Contains(source, fn.Name.Name+"(") {
				continue
			}
			helper := fullSourceFragment(parsed, fn)
			if strings.Contains(helper, "allowed[") && strings.Contains(helper, "return fmt.Errorf") && strings.Contains(helper, "*current = next") {
				return true
			}
		}
	}
	return false
}

// databaseTransitionOwner recognizes an explicit lifecycle whose state is
// persisted with compare-and-swap SQL rather than a Go field assignment. It
// requires a typed owner return, a state CAS, a called validator backed by a
// legal-transition graph, and a test that exercises the owner and asserts the
// illegal-transition error.
func databaseTransitionOwner(facts *packageFacts, typeName, fieldName string) (Evidence, bool) {
	column := strings.ToLower(fieldName)
	for _, parsed := range facts.files {
		if parsed.test {
			continue
		}
		for _, decl := range parsed.file.Decls {
			owner, ok := decl.(*ast.FuncDecl)
			if !ok || owner.Body == nil || owner.Name.Name != "Transition" || !functionReturnsNamedType(owner, typeName) {
				continue
			}
			source := fullSourceFragment(parsed, owner)
			lower := strings.ToLower(source)
			if !strings.Contains(lower, "update ") || !strings.Contains(lower, "set "+column+" =") || !strings.Contains(lower, "where") || !strings.Contains(lower, column+" =") {
				continue
			}
			validator := calledFunctionName(source, "validate")
			if validator == "" || !hasLegalTransitionValidator(facts, validator) {
				continue
			}
			if hasIllegalTransitionTest(facts, owner.Name.Name) {
				return Evidence{Path: filepath.ToSlash(parsed.path), Line: parsed.fset.Position(owner.Pos()).Line, Text: "typed transition owner persists state with compare-and-swap SQL and validates legal edges"}, true
			}
		}
	}
	return Evidence{}, false
}

func functionReturnsNamedType(fn *ast.FuncDecl, name string) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, result := range fn.Type.Results.List {
		if ident, ok := result.Type.(*ast.Ident); ok && ident.Name == name {
			return true
		}
	}
	return false
}

func calledFunctionName(source, prefix string) string {
	for _, token := range strings.FieldsFunc(source, func(r rune) bool {
		return !(r == '_' || r == '.' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	}) {
		if strings.HasPrefix(strings.ToLower(token), strings.ToLower(prefix)) {
			return token
		}
	}
	return ""
}

func hasLegalTransitionValidator(facts *packageFacts, validator string) bool {
	for _, parsed := range facts.files {
		if parsed.test {
			continue
		}
		for _, decl := range parsed.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != validator {
				continue
			}
			source := fullSourceFragment(parsed, fn)
			if strings.Contains(source, "legalTransition(") && (strings.Contains(source, "ErrIllegalTransition") || strings.Contains(source, "return fmt.Errorf")) {
				return true
			}
		}
	}
	return false
}

func hasIllegalTransitionTest(facts *packageFacts, owner string) bool {
	for _, parsed := range facts.files {
		if !parsed.test {
			continue
		}
		for _, decl := range parsed.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !strings.Contains(fn.Name.Name, "Test") {
				continue
			}
			source := fullSourceFragment(parsed, fn)
			if strings.Contains(source, owner+"(") && strings.Contains(source, "ErrIllegalTransition") && (strings.Contains(source, "errors.Is(") || strings.Contains(source, "ErrorIs(")) {
				return true
			}
		}
	}
	return false
}

// fieldAssignmentIndex finds an assignment to the lifecycle field, skipping
// equality comparisons such as `s.State == ModelReady` that share the same
// prefix. A comparison before the rejection branch is part of the graph, not
// an early state mutation.
func fieldAssignmentIndex(source, field string) int {
	needle := "." + field
	for from := 0; from < len(source); {
		relative := strings.Index(source[from:], needle)
		if relative < 0 {
			return -1
		}
		index := from + relative
		equalsRelative := strings.Index(source[index+len(needle):], "=")
		if equalsRelative < 0 {
			return -1
		}
		afterEquals := index + len(needle) + equalsRelative
		if afterEquals+1 >= len(source) || source[afterEquals+1] != '=' {
			return afterEquals
		}
		from = afterEquals + 2
	}
	return -1
}

func fullSourceFragment(parsed parsedFile, node ast.Node) string {
	start := parsed.fset.Position(node.Pos()).Offset
	end := parsed.fset.Position(node.End()).Offset
	if start < 0 || end < start || end > len(parsed.source) {
		return ""
	}
	return parsed.source[start:end]
}

func transitionMethodName(source string) string {
	start := strings.Index(source, "func ")
	if start < 0 {
		return ""
	}
	declaration := strings.TrimSpace(source[start+len("func "):])
	if strings.HasPrefix(declaration, "(") {
		if closeReceiver := strings.Index(declaration, ") "); closeReceiver >= 0 {
			declaration = declaration[closeReceiver+2:]
		}
	}
	end := strings.IndexByte(declaration, '(')
	if end <= 0 {
		return ""
	}
	return strings.TrimSpace(declaration[:end])
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
			owners = append(owners, typedStatePolicyOwnerFields(fn, facts.structs)...)
			owners = append(owners, recordTypedStatePolicyOwnerFields(fn, facts.structFields)...)
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

func typedStatePolicyOwnerFields(fn *ast.FuncDecl, structs map[string]map[string]fieldFact) []transitionOwnerField {
	if fn == nil || fn.Recv == nil || fn.Body == nil || fn.Name.Name != "CanTransitionTo" {
		return nil
	}
	var result []transitionOwnerField
	for _, receiver := range fn.Recv.List {
		stateType := exprName(receiver.Type)
		for ownerType, fields := range structs {
			for fieldName, field := range fields {
				if field.typeName == stateType {
					result = append(result, transitionOwnerField{TypeName: ownerType, FieldName: fieldName})
				}
			}
		}
	}
	return result
}

// recordTypedStatePolicyOwnerFields maps a CanTransitionTo method declared on
// a persisted record back to the lifecycle field it reads. This covers
// records whose field stores a string while the record method owns the graph,
// rather than the state enum declaring the policy itself.
func recordTypedStatePolicyOwnerFields(fn *ast.FuncDecl, structFields map[string]map[string]string) []transitionOwnerField {
	if fn == nil || fn.Recv == nil || fn.Body == nil || fn.Name.Name != "CanTransitionTo" || fn.Type.Results == nil {
		return nil
	}
	returnsBool := false
	for _, result := range fn.Type.Results.List {
		if exprName(result.Type) == "bool" {
			returnsBool = true
		}
	}
	if !returnsBool {
		return nil
	}
	hasFinitePolicy := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.SwitchStmt:
			hasFinitePolicy = true
		case *ast.CallExpr:
			if ident, ok := node.Fun.(*ast.Ident); ok && (strings.HasPrefix(strings.ToLower(ident.Name), "known") || strings.HasPrefix(strings.ToLower(ident.Name), "valid")) {
				hasFinitePolicy = true
			}
		}
		return true
	})
	if !hasFinitePolicy {
		return nil
	}
	var owners []transitionOwnerField
	for _, receiver := range fn.Recv.List {
		typeName := exprName(receiver.Type)
		fields := structFields[typeName]
		if len(fields) == 0 {
			continue
		}
		for _, receiverName := range receiver.Names {
			for fieldName := range fields {
				if !isLifecycleField(fieldName) {
					continue
				}
				readsField := false
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					selector, ok := node.(*ast.SelectorExpr)
					if !ok || selector.Sel.Name != fieldName {
						return true
					}
					base, ok := selector.X.(*ast.Ident)
					readsField = readsField || (ok && base.Name == receiverName.Name)
					return true
				})
				if readsField {
					owners = append(owners, transitionOwnerField{TypeName: typeName, FieldName: fieldName})
				}
			}
		}
	}
	return owners
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
	name := strings.ToLower(fn.Name.Name)
	return strings.HasPrefix(name, "transition") || name == "advance"
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
				unary, ok := node.(*ast.UnaryExpr)
				if ok && unary.Op == token.AND {
					selector, selectorOK := unary.X.(*ast.SelectorExpr)
					if selectorOK {
						base, baseOK := selector.X.(*ast.Ident)
						if baseOK && base.Name == receiverName.Name && isLifecycleField(selector.Sel.Name) {
							owners = append(owners, transitionOwnerField{TypeName: typeName, FieldName: selector.Sel.Name})
						}
					}
				}
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
	for _, marker := range []string{"complete", "done", "failed", "cancel", "pause", "abandon", "active", "ready", "terminal", "retry", "start", "stop", "finish", "shutdown"} {
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
	Scope      string          `json:"scope"`
	Complete   *bool           `json:"complete,omitempty"`
	Degraded   json.RawMessage `json:"degraded,omitempty"`
	EntryCount int             `json:"entry_count,omitempty"`
	NextCursor *string         `json:"next_cursor,omitempty"`
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
	degraded := len(index.Coverage.Degraded) != 0
	nextCursor := index.Coverage.NextCursor
	if index.Complete != nil {
		complete = *index.Complete
		entryCount = index.EntryCount
		degraded = topLevelDegraded(index.Degraded)
		nextCursor = index.NextCursor
	}
	return complete && entryCount > 0 && !degraded && nextCursor == nil
}

// source-ledger journal export represents degradation as a boolean, while
// Search Hub recall and earlier nested coverage envelopes use string arrays.
// Unknown encodings remain degraded so incomplete coverage cannot rank owners.
func topLevelDegraded(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var degraded bool
	if err := json.Unmarshal(raw, &degraded); err == nil {
		return degraded
	}
	var reasons []string
	if err := json.Unmarshal(raw, &reasons); err == nil {
		return len(reasons) > 0
	}
	return true
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
