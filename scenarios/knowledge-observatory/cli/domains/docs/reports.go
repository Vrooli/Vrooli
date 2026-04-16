package docs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

type AuditResponse struct {
	ScenarioName        string                  `json:"scenario_name"`
	HealthScore         float64                 `json:"health_score"`
	TotalDocs           int                     `json:"total_docs"`
	Infrastructure      *AuditInfrastructure    `json:"infrastructure"`
	CodeWithoutDocRefs  []AuditUndocumentedFile `json:"code_without_doc_refs"`
	BrokenCodeRefs      []AuditBrokenRef        `json:"broken_code_refs"`
	OrphanedDocs        []string                `json:"orphaned_docs"`
	DuplicateTitles     []AuditDuplicateTitle   `json:"duplicate_titles"`
	UndocumentedTargets []string                `json:"undocumented_targets"`
}

type AuditInfrastructure struct {
	MisplacedDocs []AuditMisplacedDoc `json:"misplaced_docs"`
	MissingDocs   []string            `json:"missing_docs"`
	ExtraDocs     []string            `json:"extra_docs"`
	TemporaryDocs []string            `json:"temporary_docs"`
}

type AuditMisplacedDoc struct {
	ActualPath   string `json:"actual_path"`
	ExpectedPath string `json:"expected_path"`
	DocType      string `json:"doc_type"`
	Severity     string `json:"severity"`
}

type AuditUndocumentedFile struct {
	Path            string `json:"path"`
	ExportedSymbols int    `json:"exported_symbols"`
}

type AuditBrokenRef struct {
	DocPath string `json:"doc_path"`
	Line    int    `json:"line"`
	Target  string `json:"target"`
}

type AuditDuplicateTitle struct {
	Title string   `json:"title"`
	Files []string `json:"files"`
}

type HealthResponse struct {
	ScenarioName  string              `json:"scenario_name"`
	HealthScore   float64             `json:"health_score"`
	TotalDocs     int                 `json:"total_docs"`
	MisplacedDocs []AuditMisplacedDoc `json:"misplaced_docs"`
	MissingDocs   []string            `json:"missing_docs"`
	ExtraDocs     []string            `json:"extra_docs"`
	TemporaryDocs []string            `json:"temporary_docs"`
	CanAutoFix    bool                `json:"can_auto_fix"`
	FixCategory   string              `json:"fix_category"`
}

func (d *AuditMisplacedDoc) UnmarshalJSON(data []byte) error {
	type alias AuditMisplacedDoc
	var tagged alias
	if err := json.Unmarshal(data, &tagged); err != nil {
		return err
	}
	type legacy struct {
		ActualPath   string `json:"ActualPath"`
		ExpectedPath string `json:"ExpectedPath"`
		DocType      string `json:"DocType"`
		Severity     string `json:"Severity"`
	}
	var legacyValue legacy
	if err := json.Unmarshal(data, &legacyValue); err != nil {
		return err
	}
	if strings.TrimSpace(tagged.ActualPath) == "" {
		tagged.ActualPath = legacyValue.ActualPath
	}
	if strings.TrimSpace(tagged.ExpectedPath) == "" {
		tagged.ExpectedPath = legacyValue.ExpectedPath
	}
	if strings.TrimSpace(tagged.DocType) == "" {
		tagged.DocType = legacyValue.DocType
	}
	if strings.TrimSpace(tagged.Severity) == "" {
		tagged.Severity = legacyValue.Severity
	}
	*d = AuditMisplacedDoc(tagged)
	return nil
}

func (d *AuditInfrastructure) UnmarshalJSON(data []byte) error {
	type alias AuditInfrastructure
	var tagged alias
	if err := json.Unmarshal(data, &tagged); err != nil {
		return err
	}
	type legacy struct {
		MisplacedDocs []AuditMisplacedDoc `json:"MisplacedDocs"`
		MissingDocs   []string            `json:"MissingDocs"`
		ExtraDocs     []string            `json:"ExtraDocs"`
		TemporaryDocs []string            `json:"TemporaryDocs"`
	}
	var legacyValue legacy
	if err := json.Unmarshal(data, &legacyValue); err != nil {
		return err
	}
	if len(tagged.MisplacedDocs) == 0 {
		tagged.MisplacedDocs = legacyValue.MisplacedDocs
	}
	if len(tagged.MissingDocs) == 0 {
		tagged.MissingDocs = legacyValue.MissingDocs
	}
	if len(tagged.ExtraDocs) == 0 {
		tagged.ExtraDocs = legacyValue.ExtraDocs
	}
	if len(tagged.TemporaryDocs) == 0 {
		tagged.TemporaryDocs = legacyValue.TemporaryDocs
	}
	*d = AuditInfrastructure(tagged)
	return nil
}

type auditSeverity string

const (
	auditSeverityOK   auditSeverity = "OK"
	auditSeverityWarn auditSeverity = "WARN"
	auditSeverityFail auditSeverity = "FAIL"
)

type triageItem struct {
	priority int
	sortKey  string
	text     string
}

type manualGroup struct {
	name  string
	items []triageItem
}

type nextStep struct {
	description string
	command     string
}

func RenderAuditReport(result AuditResponse, fallbackScenario string) string {
	var out bytes.Buffer
	_ = cliapp.RenderOperationalReport(&out, BuildAuditReport(result, fallbackScenario))
	return out.String()
}

func BuildAuditReport(result AuditResponse, fallbackScenario string) cliapp.OperationalReport {
	scenario := strings.TrimSpace(result.ScenarioName)
	if scenario == "" {
		scenario = strings.TrimSpace(fallbackScenario)
	}
	if scenario == "" {
		scenario = "unknown"
	}

	misplaced, missing, extra, temporary := infraAuditSlices(result.Infrastructure)
	autoItems := buildAutoFixItems(misplaced)
	agentItems := buildAgentItems(missing, extra)
	manualGroups := buildManualGroups(result, temporary)
	autoCount := len(autoItems)
	agentCount := len(agentItems)
	manualCount := countManualGroupItems(manualGroups)
	totalFindings := autoCount + agentCount + manualCount

	status := classifyAuditStatus(totalFindings, result)
	healthPct := int(result.HealthScore*100 + 0.5)
	if healthPct < 0 {
		healthPct = 0
	}
	if healthPct > 100 {
		healthPct = 100
	}

	statusLine := fmt.Sprintf("Documentation Audit: %s", scenario)
	healthLine := fmt.Sprintf("Health: %d%% (%d docs", healthPct, result.TotalDocs)
	drivers := auditStatusDrivers(result)
	if len(drivers) > 0 {
		statusLine += fmt.Sprintf(" | Status: %s (drivers: %s)", status, strings.Join(drivers, ", "))
	} else {
		statusLine += fmt.Sprintf(" | Status: %s", status)
	}
	if len(misplaced)+len(missing)+len(extra)+len(temporary) > 0 {
		healthLine += fmt.Sprintf("; %d misplaced, %d missing, %d extra, %d temporary", len(misplaced), len(missing), len(extra), len(temporary))
	}
	healthLine += ")"

	report := cliapp.OperationalReport{
		Status: []string{
			statusLine,
			healthLine,
			fmt.Sprintf("Findings: %d total", totalFindings),
		},
	}

	if len(autoItems) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: fmt.Sprintf("Auto-fix now (%d)", len(autoItems)),
			Items:   triageTexts(autoItems),
		})
	}
	if len(agentItems) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: fmt.Sprintf("Agent repair (%d)", len(agentItems)),
			Items:   triageTexts(agentItems),
		})
	}
	if manualCount > 0 {
		report.Triage = append(report.Triage, buildManualReviewGroup(manualGroups, manualCount))
	}
	if len(report.Triage) == 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Triage",
			Items:   []string{"No findings"},
		})
	}

	report.NextSteps = make([]string, 0, 4)
	for i, step := range nextStepGuidance(scenario, autoCount, agentCount, manualCount, result.HealthScore) {
		report.NextSteps = append(report.NextSteps, fmt.Sprintf("%d. %s", i+1, step.description), step.command)
	}
	return report
}

func RenderHealthReport(result HealthResponse, fallbackScenario string) string {
	var out bytes.Buffer
	_ = cliapp.RenderOperationalReport(&out, BuildHealthReport(result, fallbackScenario))
	return out.String()
}

func BuildHealthReport(result HealthResponse, fallbackScenario string) cliapp.OperationalReport {
	scenario := strings.TrimSpace(result.ScenarioName)
	if scenario == "" {
		scenario = strings.TrimSpace(fallbackScenario)
	}
	if scenario == "" {
		scenario = "unknown"
	}

	requiredDocs := 1
	requiredPresent := requiredDocs
	missingRequired := 0
	for _, missing := range result.MissingDocs {
		if strings.EqualFold(strings.TrimSpace(missing), "readme") {
			missingRequired++
		}
	}
	requiredPresent -= missingRequired
	if requiredPresent < 0 {
		requiredPresent = 0
	}
	requiredCoverage := 1.0
	if requiredDocs > 0 {
		requiredCoverage = float64(requiredPresent) / float64(requiredDocs)
	}

	misplacedPenalty := 0.05 * float64(len(result.MisplacedDocs))
	temporaryPenalty := 0.01 * float64(len(result.TemporaryDocs))
	healthPct := int(result.HealthScore*100 + 0.5)
	if healthPct < 0 {
		healthPct = 0
	}
	if healthPct > 100 {
		healthPct = 100
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Documentation Health: %s", scenario),
			fmt.Sprintf("Score: %d%% (%d docs)", healthPct, result.TotalDocs),
			fmt.Sprintf("Issues: %d misplaced, %d missing, %d extra, %d temporary", len(result.MisplacedDocs), len(result.MissingDocs), len(result.ExtraDocs), len(result.TemporaryDocs)),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Score breakdown",
				Items: []string{
					fmt.Sprintf("Required docs baseline: %.0f%% (%d/%d present)", requiredCoverage*100, requiredPresent, requiredDocs),
					fmt.Sprintf("Misplaced penalty: -%.0f%% (%d x 5%%)", misplacedPenalty*100, len(result.MisplacedDocs)),
					fmt.Sprintf("Temporary-docs penalty: -%.0f%% (%d x 1%%)", temporaryPenalty*100, len(result.TemporaryDocs)),
					fmt.Sprintf("Extra docs are informational only (%d)", len(result.ExtraDocs)),
					"Final score is clamped to 0-100%",
				},
			},
			{
				Heading: "Fixability",
				Items: []string{
					fmt.Sprintf("Fix category: %s", strings.TrimSpace(result.FixCategory)),
					fmt.Sprintf("Quick-fixable files: %d", len(result.MisplacedDocs)),
				},
			},
		},
	}
	if result.CanAutoFix {
		report.Triage[1].Items = append(report.Triage[1].Items, "Auto-fix available: yes")
	} else {
		report.Triage[1].Items = append(report.Triage[1].Items, "Auto-fix available: no")
	}
	return report
}

func triageTexts(items []triageItem) []string {
	const maxExamples = 10
	limit := len(items)
	if limit > maxExamples {
		limit = maxExamples
	}
	lines := make([]string, 0, limit+1)
	for i := 0; i < limit; i++ {
		item := items[i]
		lines = append(lines, item.text)
	}
	if len(items) > maxExamples {
		lines = append(lines, fmt.Sprintf("... +%d more", len(items)-maxExamples))
	}
	return lines
}

func buildManualReviewGroup(groups []manualGroup, total int) cliapp.TriageGroup {
	const maxExamples = 10
	items := make([]string, 0, total+len(groups))
	for _, group := range groups {
		items = append(items, fmt.Sprintf("%s (%d)", group.name, len(group.items)))
		limit := len(group.items)
		if limit > maxExamples {
			limit = maxExamples
		}
		for i := 0; i < limit; i++ {
			items = append(items, group.items[i].text)
		}
		if len(group.items) > maxExamples {
			items = append(items, fmt.Sprintf("... +%d more", len(group.items)-maxExamples))
		}
	}
	return cliapp.TriageGroup{
		Heading: fmt.Sprintf("Manual review (%d)", total),
		Items:   items,
	}
}

func classifyAuditStatus(totalFindings int, result AuditResponse) auditSeverity {
	if totalFindings == 0 {
		return auditSeverityOK
	}
	if len(result.BrokenCodeRefs) > 0 || len(result.UndocumentedTargets) > 0 {
		return auditSeverityFail
	}
	return auditSeverityWarn
}

func infraAuditSlices(infra *AuditInfrastructure) ([]AuditMisplacedDoc, []string, []string, []string) {
	if infra == nil {
		return nil, nil, nil, nil
	}
	return infra.MisplacedDocs, infra.MissingDocs, infra.ExtraDocs, infra.TemporaryDocs
}

func buildAutoFixItems(misplaced []AuditMisplacedDoc) []triageItem {
	items := make([]triageItem, 0, len(misplaced))
	for _, doc := range misplaced {
		actual := strings.TrimSpace(doc.ActualPath)
		expected := strings.TrimSpace(doc.ExpectedPath)
		if actual == "" && expected == "" {
			continue
		}
		items = append(items, triageItem{priority: 0, sortKey: strings.ToLower(actual + "|" + expected), text: fmt.Sprintf("%s -> %s", actual, expected)})
	}
	sortTriageItems(items)
	return items
}

func buildAgentItems(missing []string, extra []string) []triageItem {
	items := make([]triageItem, 0, len(missing)+len(extra))
	for _, value := range missing {
		docType := strings.TrimSpace(value)
		if docType != "" {
			items = append(items, triageItem{priority: 0, sortKey: strings.ToLower(docType), text: "Missing: " + docType})
		}
	}
	for _, value := range extra {
		path := strings.TrimSpace(value)
		if path != "" {
			items = append(items, triageItem{priority: 1, sortKey: strings.ToLower(path), text: "Extra: " + path})
		}
	}
	sortTriageItems(items)
	return items
}

func sortTriageItems(items []triageItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].priority != items[j].priority {
			return items[i].priority < items[j].priority
		}
		return items[i].sortKey < items[j].sortKey
	})
}

func writeTriageBucket(b *strings.Builder, name string, items []triageItem) {
	fmt.Fprintf(b, "- %s (%d)\n", name, len(items))
	const maxExamples = 10
	limit := len(items)
	if limit > maxExamples {
		limit = maxExamples
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(b, "  - %s\n", items[i].text)
	}
	if len(items) > maxExamples {
		fmt.Fprintf(b, "  ... +%d more\n", len(items)-maxExamples)
	}
}

func writeManualReviewBucket(b *strings.Builder, groups []manualGroup) {
	total := countManualGroupItems(groups)
	fmt.Fprintf(b, "- Manual review (%d)\n", total)
	const maxExamples = 10
	for _, group := range groups {
		if len(group.items) == 0 {
			continue
		}
		fmt.Fprintf(b, "  - %s (%d)\n", group.name, len(group.items))
		limit := len(group.items)
		if limit > maxExamples {
			limit = maxExamples
		}
		for i := 0; i < limit; i++ {
			fmt.Fprintf(b, "    - %s\n", group.items[i].text)
		}
		if len(group.items) > maxExamples {
			fmt.Fprintf(b, "    ... +%d more\n", len(group.items)-maxExamples)
		}
	}
}

func nextStepGuidance(scenario string, autoCount, agentCount, manualCount int, healthScore float64) []nextStep {
	steps := make([]nextStep, 0, 4)
	if autoCount > 0 {
		steps = append(steps, nextStep{"To apply deterministic quick fixes for misplaced docs, run:", fmt.Sprintf("knowledge-observatory docs autofix %s", scenario)})
	}
	if agentCount > 0 {
		steps = append(steps, nextStep{"To run agent-driven repair for missing/extra docs, run:", fmt.Sprintf("knowledge-observatory docs heal %s --wait", scenario)})
	}
	if manualCount > 0 {
		steps = append(steps, nextStep{"To inspect full findings in machine-readable form, run:", fmt.Sprintf("knowledge-observatory docs audit %s --json", scenario)})
	}
	if healthScore < 0.9995 {
		steps = append(steps, nextStep{"To see a detailed documentation-health breakdown and penalties, run:", fmt.Sprintf("knowledge-observatory docs health %s", scenario)})
	}
	if len(steps) == 0 {
		steps = append(steps, nextStep{"No action required. To verify again later, run:", fmt.Sprintf("knowledge-observatory docs audit %s", scenario)})
	}
	return steps
}

func auditStatusDrivers(result AuditResponse) []string {
	drivers := make([]string, 0, 2)
	if len(result.BrokenCodeRefs) > 0 {
		drivers = append(drivers, fmt.Sprintf("%d broken [CODE:] refs", len(result.BrokenCodeRefs)))
	}
	if len(result.UndocumentedTargets) > 0 {
		drivers = append(drivers, fmt.Sprintf("%d undocumented operational targets", len(result.UndocumentedTargets)))
	}
	return drivers
}

func buildManualGroups(result AuditResponse, temporary []string) []manualGroup {
	groups := []manualGroup{
		{name: "No DOC refs", items: make([]triageItem, 0, len(result.CodeWithoutDocRefs))},
		{name: "Broken [CODE:] refs", items: make([]triageItem, 0, len(result.BrokenCodeRefs))},
		{name: "Orphaned docs", items: make([]triageItem, 0, len(result.OrphanedDocs))},
		{name: "Duplicate titles", items: make([]triageItem, 0, len(result.DuplicateTitles))},
		{name: "Undocumented operational targets", items: make([]triageItem, 0, len(result.UndocumentedTargets))},
		{name: "Temporary docs", items: make([]triageItem, 0, len(temporary))},
	}
	for _, file := range result.CodeWithoutDocRefs {
		path := strings.TrimSpace(file.Path)
		groups[0].items = append(groups[0].items, triageItem{sortKey: strings.ToLower(path), text: fmt.Sprintf("%s (%d exported)", path, file.ExportedSymbols)})
	}
	for _, ref := range result.BrokenCodeRefs {
		path := strings.TrimSpace(ref.DocPath)
		target := strings.TrimSpace(ref.Target)
		groups[1].items = append(groups[1].items, triageItem{sortKey: strings.ToLower(path + "|" + strconv.Itoa(ref.Line) + "|" + target), text: fmt.Sprintf("%s:%d -> %s", path, ref.Line, target)})
	}
	for _, doc := range result.OrphanedDocs {
		path := strings.TrimSpace(doc)
		groups[2].items = append(groups[2].items, triageItem{sortKey: strings.ToLower(path), text: path})
	}
	for _, title := range result.DuplicateTitles {
		name := strings.TrimSpace(title.Title)
		groups[3].items = append(groups[3].items, triageItem{sortKey: strings.ToLower(name), text: fmt.Sprintf("%q", name)})
	}
	for _, target := range result.UndocumentedTargets {
		value := strings.TrimSpace(target)
		groups[4].items = append(groups[4].items, triageItem{sortKey: strings.ToLower(value), text: value})
	}
	for _, path := range temporary {
		value := strings.TrimSpace(path)
		groups[5].items = append(groups[5].items, triageItem{sortKey: strings.ToLower(value), text: value})
	}
	out := make([]manualGroup, 0, len(groups))
	for _, group := range groups {
		if len(group.items) == 0 {
			continue
		}
		sort.Slice(group.items, func(i, j int) bool { return group.items[i].sortKey < group.items[j].sortKey })
		out = append(out, group)
	}
	return out
}

func countManualGroupItems(groups []manualGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.items)
	}
	return total
}
