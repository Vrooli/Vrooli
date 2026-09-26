package catalog

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:embed import-manifest.json
var importManifestFS embed.FS

type importManifest struct {
	Paths []manifestPath `json:"paths"`
}
type manifestPath struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Cardinality string `json:"cardinality"`
}

type CatalogImportFile struct {
	Path        string
	Read        int
	Written     int
	Findings    int
	Cardinality string
	NodeKind    offerspb.NodeKind
}
type CatalogImportStatus struct {
	Path       string
	Status     offerspb.Status
	Recognized bool
	Line       int
}
type CatalogImportFinding struct {
	Path     string
	Reason   string
	Blocking bool
	Line     int
}
type CatalogImportReport struct {
	Files     []CatalogImportFile
	StatusMap []CatalogImportStatus
	Findings  []CatalogImportFinding
	Applied   bool
}

type CatalogVerifyFile struct {
	Path     string
	Expected int
	Live     int
}

type CatalogVerifyReport struct {
	Files               []CatalogVerifyFile
	DuplicateIdentities []string
	OrphanEdgeIds       []string
	ExtraNodeIds        []string
	TotalDrift          int
	// Reconciled reports that every check which could run agreed. It is not a
	// claim that the source-versus-live count comparison ran — read Comparable
	// for that.
	Reconciled bool
	// Comparable reports whether the source-versus-live node-count comparison
	// actually ran. It is false when no declared source file yielded a countable
	// record, which is the expected steady state after sources have been
	// compressed to judgment-only prose and their state moved into this
	// scenario. Before this field existed, that state and "the import never
	// ran" were both reported as a bare reconciled=true and could not be told
	// apart.
	Comparable bool
	// NotComparableReason states why the count comparison was skipped. Empty
	// when Comparable is true.
	NotComparableReason string
	ScenarioGaps        []string
}

type importedNode struct {
	id        string
	kind      offerspb.NodeKind
	name      string
	status    offerspb.Status
	fileIndex int
}
type importedEdge struct {
	from, to, kind, currency string
	price                    int64
	priceDeclared            bool
	fileIndex                int
}

type pricingCell struct {
	offer, variant, currency string
	priceMinor               int64
	declared                 bool
}

// operatorReferenceFindings keeps the operator import honest about links in
// the source tree. The catalog deliberately does not copy prose, but dropping
// a broken reference would make a source migration look complete when a
// consumer-facing relationship was actually lost. References are findings,
// rather than lifecycle records, and therefore remain visible without making
// an otherwise valid catalog impossible to apply.
func operatorReferenceFindings(root, sourcePath, body string) []CatalogImportFinding {
	findings := make([]CatalogImportFinding, 0)
	for _, match := range markdownReference.FindAllStringSubmatch(body, -1) {
		target := strings.TrimSpace(match[1])
		if target == "" || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:") {
			continue
		}
		candidate := filepath.Clean(filepath.Join(filepath.Dir(filepath.Join(root, filepath.FromSlash(sourcePath))), filepath.FromSlash(target)))
		if filepath.Ext(candidate) == "" {
			candidate += ".md"
		}
		if _, err := os.Stat(candidate); err != nil {
			findings = append(findings, CatalogImportFinding{Path: sourcePath, Reason: "unresolvable internal reference: " + target, Blocking: false, Line: referenceLine(body, match[0])})
		}
	}
	return findings
}

func referenceLine(body, match string) int {
	index := strings.Index(body, match)
	if index < 0 {
		return 0
	}
	return 1 + strings.Count(body[:index], "\n")
}

var (
	pricingCurrencyAmount = regexp.MustCompile(`(?i)^\s*([A-Z]{3})\s*\$?\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)`)
	pricingAmountCurrency = regexp.MustCompile(`(?i)^\s*\$?\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)\s*([A-Z]{3})`)
	pricingAmount         = regexp.MustCompile(`(?i)[0-9][0-9,]*(?:\.[0-9]{1,2})?`)
	tierNumber            = regexp.MustCompile(`(?i)\btier\s*([1-4])\b`)
)

var (
	statusMarker = regexp.MustCompile(`(?im)^\s*(?:-\s+)?\*\*Status:`)
	tierHeading  = regexp.MustCompile("(?m)^###\\s+Tier\\s+([1-4])\\s+—\\s+([^\\n(]+)\\s+\\(`?(active|candidate|north-star|retired)`?\\)")
	skuID        = regexp.MustCompile(`(?m)^\*\*SKU ID:\*\*\s*` + "`?" + `([^` + "`" + `\s]+)`)
)

// hasAuthorityHandoff recognizes a mixed source after its live records have
// moved into the instrument. The source remains valuable for judgment, but
// the absence of a prose status marker or pricing table is intentional rather
// than a malformed catalog. This is deliberately explicit so an unrelated
// source that silently drops its lifecycle fields still fails loudly.
func hasAuthorityHandoff(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "offer desk is authoritative") || strings.Contains(lower, "money ledger is authoritative")
}

func loadImportManifest(sourceRoot string) (importManifest, error) {
	data, err := os.ReadFile(filepath.Join(sourceRoot, "import-manifest.json"))
	if err != nil {
		// Fixture imports retain the embedded compatibility roster; operator
		// supplied canon imports must carry their own adjacent roster.
		data, err = importManifestFS.ReadFile("import-manifest.json")
		if err != nil {
			return importManifest{}, err
		}
	}
	var manifest importManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return importManifest{}, err
	}
	return manifest, nil
}

func resolveImportRoot(root string, mode offerspb.SourceMode) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.New("catalog source_path is required")
	}
	switch mode {
	case offerspb.SourceMode_SOURCE_MODE_FIXTURE:
		return fixtureRoot(root)
	case offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED:
		abs, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(abs)
		if err != nil && !filepath.IsAbs(root) {
			for parent := filepath.Dir(abs); parent != abs; parent = filepath.Dir(parent) {
				candidate := filepath.Join(parent, root)
				if candidateInfo, candidateErr := os.Stat(candidate); candidateErr == nil && candidateInfo.IsDir() {
					abs, info, err = candidate, candidateInfo, nil
					break
				}
			}
		}
		if err != nil {
			return "", fmt.Errorf("catalog operator source: %w", err)
		}
		if !info.IsDir() {
			return "", errors.New("catalog operator source must be a directory")
		}
		return abs, nil
	default:
		return "", errors.New("catalog source_mode must be fixture or operator-supplied")
	}
}

func statusFromWord(word string) (offerspb.Status, bool) {
	switch strings.ToLower(strings.TrimSpace(word)) {
	case "active":
		return offerspb.Status_ACTIVE, true
	case "candidate":
		return offerspb.Status_CANDIDATE, true
	case "north-star":
		return offerspb.Status_IDEA, true
	case "shipped":
		return offerspb.Status_SHIPPED, true
	case "retired":
		return offerspb.Status_RETIRED, true
	case "trigger-met":
		return offerspb.Status_TRIGGER_MET, true
	case "proposed":
		return offerspb.Status_PROPOSED, true
	default:
		return offerspb.Status_STATUS_UNSPECIFIED, false
	}
}

func nodeKindForManifest(kind string) (offerspb.NodeKind, error) {
	switch kind {
	case "none":
		return offerspb.NodeKind_NODE_KIND_UNSPECIFIED, nil
	case "channel":
		return offerspb.NodeKind_CHANNEL, nil
	case "revenue-line":
		return offerspb.NodeKind_REVENUE_LINE, nil
	case "deliverable":
		return offerspb.NodeKind_DELIVERABLE, nil
	case "variant":
		return offerspb.NodeKind_VARIANT, nil
	case "offer":
		return offerspb.NodeKind_OFFER, nil
	case "benchmark", "pricing":
		return offerspb.NodeKind_NODE_KIND_UNSPECIFIED, nil
	default:
		return offerspb.NodeKind_NODE_KIND_UNSPECIFIED, fmt.Errorf("unrecognized import manifest kind %q", kind)
	}
}

func markdownRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	parts := strings.Split(strings.Trim(line, "|"), "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func markdownSeparator(row []string) bool {
	if len(row) == 0 {
		return false
	}
	for _, cell := range row {
		if strings.Trim(cell, "-: ") != "" {
			return false
		}
	}
	return true
}

func normalizeCatalogName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimSuffix(value, "(add-on)")
	value = strings.TrimSpace(strings.TrimSuffix(value, "bundle"))
	value = strings.TrimSpace(strings.TrimSuffix(value, "(base)"))
	value = strings.NewReplacer(" ", "-", "_", "-", "—", "-", "–", "-").Replace(value)
	value = strings.Trim(value, "- ")
	return value
}

func parseMinorUnits(value string) (int64, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	parts := strings.SplitN(value, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > 2 {
		return 0, fmt.Errorf("amount has more than two decimal places")
	}
	frac += strings.Repeat("0", 2-len(frac))
	minorFrac := int64(0)
	if frac != "" {
		minorFrac, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	if whole > (int64(^uint64(0)>>1)-minorFrac)/100 || whole < 0 {
		return 0, errors.New("amount is outside int64 minor-unit range")
	}
	return whole*100 + minorFrac, nil
}

func parsePricingCell(offer, variant, raw string) (pricingCell, string) {
	cell := strings.TrimSpace(strings.Trim(raw, "`"))
	result := pricingCell{offer: offer, variant: variant}
	if cell == "" || cell == "—" || cell == "-" || strings.Contains(strings.ToLower(cell), "tbd") || strings.Contains(strings.ToLower(cell), "north-star") {
		return result, ""
	}
	match := pricingCurrencyAmount.FindStringSubmatch(cell)
	if len(match) != 3 {
		match = pricingAmountCurrency.FindStringSubmatch(cell)
		if len(match) == 3 {
			match = []string{match[0], match[2], match[1]}
		}
	}
	if len(match) != 3 {
		if pricingAmount.MatchString(cell) {
			return result, "declared amount has no ISO currency"
		}
		return result, ""
	}
	minor, err := parseMinorUnits(match[2])
	if err != nil {
		return result, "invalid declared amount: " + err.Error()
	}
	result.currency = strings.ToUpper(match[1])
	result.priceMinor = minor
	result.declared = true
	return result, ""
}

func parsePricingRows(body string) ([]pricingCell, []CatalogImportFinding) {
	lines := strings.Split(body, "\n")
	headerIndex := -1
	var header []string
	for i, line := range lines {
		row := markdownRow(line)
		if len(row) < 2 || !strings.Contains(strings.ToLower(row[0]), "sku") || !strings.Contains(strings.ToLower(row[0]), "tier") {
			continue
		}
		headerIndex, header = i, row
		break
	}
	if headerIndex < 0 {
		return nil, nil
	}
	tiers := make(map[int]string)
	for i, cell := range header[1:] {
		match := tierNumber.FindStringSubmatch(cell)
		if len(match) == 2 {
			tiers[i+1] = "tier-" + match[1]
		}
	}
	rows := make([]pricingCell, 0)
	findings := make([]CatalogImportFinding, 0)
	for _, line := range lines[headerIndex+1:] {
		row := markdownRow(line)
		if row == nil {
			break
		}
		if markdownSeparator(row) || len(row) < 2 {
			continue
		}
		offer := normalizeCatalogName(row[0])
		if offer == "" {
			continue
		}
		for column, variant := range tiers {
			if column >= len(row) {
				continue
			}
			cell, reason := parsePricingCell(offer, variant, row[column])
			if reason != "" {
				findings = append(findings, CatalogImportFinding{Path: "strategy/PRICING.md", Reason: fmt.Sprintf("%s/%s: %s", offer, variant, reason), Blocking: true})
				continue
			}
			rows = append(rows, cell)
		}
	}
	return rows, findings
}

func slugifyFactPart(value string) string {
	value = normalizeCatalogName(value)
	value = strings.Trim(value, "-")
	return value
}

func parseBenchmarkRows(body string) ([]*offerspb.Fact, []CatalogImportFinding) {
	lines := strings.Split(body, "\n")
	headerIndex := -1
	var header []string
	for i, line := range lines {
		row := markdownRow(line)
		if len(row) < 2 {
			continue
		}
		joined := strings.ToLower(strings.Join(row, "|"))
		if strings.Contains(joined, "date captured") && strings.Contains(joined, "relevant dimension") {
			headerIndex, header = i, row
			break
		}
	}
	if headerIndex < 0 {
		return nil, nil
	}
	columns := make(map[string]int, len(header))
	for i, cell := range header {
		columns[strings.ToLower(strings.TrimSpace(cell))] = i
	}
	required := []string{"comp", "relevant dimension", "value", "date captured"}
	for _, name := range required {
		if _, ok := columns[name]; !ok {
			return nil, []CatalogImportFinding{{Path: "evidence/BENCHMARKS.md", Reason: "benchmark table is missing the " + name + " column", Blocking: true}}
		}
	}
	facts := make([]*offerspb.Fact, 0)
	findings := make([]CatalogImportFinding, 0)
	for _, line := range lines[headerIndex+1:] {
		row := markdownRow(line)
		if row == nil {
			break
		}
		if markdownSeparator(row) || len(row) < len(header) {
			continue
		}
		comp := strings.TrimSpace(row[columns["comp"]])
		dimension := strings.ToLower(strings.TrimSpace(row[columns["relevant dimension"]]))
		value, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(row[columns["value"]], "%")), 64)
		if err != nil {
			findings = append(findings, CatalogImportFinding{Path: "evidence/BENCHMARKS.md", Reason: "benchmark " + comp + " has an invalid numeric value", Blocking: true})
			continue
		}
		observed, err := time.Parse("2006-01-02", strings.TrimSpace(row[columns["date captured"]]))
		if err != nil {
			findings = append(findings, CatalogImportFinding{Path: "evidence/BENCHMARKS.md", Reason: "benchmark " + comp + " has an invalid date captured", Blocking: true})
			continue
		}
		facts = append(facts, &offerspb.Fact{
			Name:           "benchmark:" + slugifyFactPart(comp) + ":" + slugifyFactPart(dimension),
			Value:          value,
			ObservedAt:     timestamppb.New(observed.UTC()),
			StaleAfterDays: staleWindow(dimension),
			Dimension:      dimension,
		})
	}
	return facts, findings
}

func (s *Store) ImportCatalog(ctx context.Context, sourcePath string, mode offerspb.SourceMode, apply bool, actor string) (*CatalogImportReport, error) {
	root, err := resolveImportRoot(sourcePath, mode)
	if err != nil {
		return nil, err
	}
	manifest, err := loadImportManifest(root)
	if err != nil {
		return nil, err
	}
	declared := make(map[string]manifestPath, len(manifest.Paths))
	for _, entry := range manifest.Paths {
		declared[filepath.ToSlash(filepath.Clean(entry.Path))] = entry
	}
	actual := map[string]struct{}{}
	if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Mode().IsRegular() {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			if filepath.ToSlash(rel) == "import-manifest.json" {
				return nil
			}
			actual[filepath.ToSlash(rel)] = struct{}{}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	report := &CatalogImportReport{}
	for path := range actual {
		if _, ok := declared[path]; !ok {
			report.Findings = append(report.Findings, CatalogImportFinding{Path: path, Reason: "undeclared source file", Blocking: true})
		}
	}
	for path := range declared {
		if _, ok := actual[path]; !ok {
			report.Findings = append(report.Findings, CatalogImportFinding{Path: path, Reason: "declared source file is missing", Blocking: true})
		}
	}

	nodes := make([]importedNode, 0)
	byName := make(map[string]importedNode)
	pricing := make(map[string]pricingCell)
	pricingOffers := make(map[string]struct{})
	benchmarkFacts := make([]*offerspb.Fact, 0)
	benchmarkFileIndex := -1
	pricingFileIndex := -1
	pricingStateRetired := false
	for _, entry := range manifest.Paths {
		path := filepath.ToSlash(filepath.Clean(entry.Path))
		if _, ok := actual[path]; !ok {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		nodeKind, kindErr := nodeKindForManifest(entry.Kind)
		if kindErr != nil {
			return nil, kindErr
		}
		file := CatalogImportFile{Path: path, Cardinality: entry.Cardinality, NodeKind: nodeKind}
		if entry.Kind == "offer" && strings.HasPrefix(path, "catalogs/skus/base/") {
			pricingOffers[strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))] = struct{}{}
		}
		fileIndex := len(report.Files)
		for _, finding := range operatorReferenceFindings(root, path, string(body)) {
			file.Findings++
			report.Findings = append(report.Findings, finding)
		}
		switch entry.Cardinality {
		case "none":
			file.Read, file.Written = 0, 0
		case "one":
			file.Read = 1
			status, found, line := parseImportedStatus(string(body))
			if !found {
				if hasAuthorityHandoff(string(body)) {
					file.Read = 0
					break
				}
				file.Findings++
				reason := "lifecycle status marker is absent"
				if statusMarker.FindStringIndex(string(body)) != nil {
					reason = "lifecycle status marker is unrecognizable"
				}
				report.Findings = append(report.Findings, CatalogImportFinding{Path: path, Reason: reason, Blocking: true, Line: line})
				report.StatusMap = append(report.StatusMap, CatalogImportStatus{Path: path, Status: offerspb.Status_STATUS_UNSPECIFIED, Recognized: false, Line: line})
				break
			}
			name := filepath.Base(path)
			if match := skuID.FindStringSubmatch(string(body)); len(match) == 2 {
				name = match[1]
			}
			n := importedNode{id: uuid.NewString(), kind: file.NodeKind, name: strings.TrimSuffix(name, filepath.Ext(name)), status: status, fileIndex: fileIndex}
			nodes = append(nodes, n)
			byName[n.name] = n
			file.Written = 1
			report.StatusMap = append(report.StatusMap, CatalogImportStatus{Path: path, Status: status, Recognized: true, Line: line})
		case "many":
			if entry.Kind == "variant" {
				matches := tierHeading.FindAllStringSubmatch(string(body), -1)
				if len(matches) == 0 && hasAuthorityHandoff(string(body)) {
					file.Read, file.Written = 0, 0
					break
				}
				for _, match := range matches {
					status, ok := statusFromWord(match[3])
					if !ok {
						file.Findings++
						report.Findings = append(report.Findings, CatalogImportFinding{Path: path, Reason: "tier status is unrecognizable", Blocking: true})
						continue
					}
					name := "tier-" + match[1]
					n := importedNode{id: uuid.NewString(), kind: offerspb.NodeKind_VARIANT, name: name, status: status, fileIndex: fileIndex}
					nodes = append(nodes, n)
					byName[n.name] = n
					report.StatusMap = append(report.StatusMap, CatalogImportStatus{Path: path + "#" + name, Status: status, Recognized: true})
				}
				file.Read, file.Written = len(matches), len(matches)
			} else if entry.Kind == "pricing" {
				pricingFileIndex = len(report.Files)
				rows, findings := parsePricingRows(string(body))
				if len(rows) == 0 && hasAuthorityHandoff(string(body)) {
					pricingStateRetired = true
					file.Read, file.Written = 0, 0
					report.Files = append(report.Files, file)
					continue
				}
				for _, finding := range findings {
					file.Findings++
					report.Findings = append(report.Findings, finding)
				}
				for _, row := range rows {
					key := row.offer + "/" + row.variant
					pricing[key] = row
				}
				file.Read = 0
				for _, row := range pricing {
					if _, ok := pricingOffers[row.offer]; ok {
						file.Read++
					}
				}
			} else if entry.Kind == "benchmark" {
				benchmarkFileIndex = fileIndex
				facts, findings := parseBenchmarkRows(string(body))
				file.Read = len(facts)
				file.Written = len(facts)
				benchmarkFacts = append(benchmarkFacts, facts...)
				for _, finding := range findings {
					file.Findings++
					report.Findings = append(report.Findings, finding)
				}
			}
		default:
			return nil, fmt.Errorf("manifest path %s has unsupported cardinality %q", path, entry.Cardinality)
		}
		report.Files = append(report.Files, file)
	}

	// The relationship declarations are part of the import, not inferred by a
	// later read. The edge payload remains typed and auditable in the catalog.
	edges := make([]importedEdge, 0)
	if business, ok := byName["business"]; ok {
		for _, addon := range []string{"elder-care", "family-with-kids", "property-services"} {
			if n, exists := byName[addon]; exists {
				edges = append(edges, importedEdge{from: n.id, to: business.id, kind: "requires", fileIndex: -1})
			}
		}
	}
	if pricingFileIndex >= 0 && !pricingStateRetired {
		keys := make([]string, 0, len(pricing))
		for key := range pricing {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			row := pricing[key]
			if _, ok := pricingOffers[row.offer]; !ok {
				continue
			}
			offer, offerOK := byName[row.offer]
			variant, variantOK := byName[row.variant]
			if !offerOK || !variantOK {
				if pricingFileIndex >= 0 {
					report.Files[pricingFileIndex].Findings++
				}
				report.Findings = append(report.Findings, CatalogImportFinding{Path: "strategy/PRICING.md", Reason: "pricing row names an unresolved offer or variant: " + key, Blocking: true})
				continue
			}
			edges = append(edges, importedEdge{from: offer.id, to: variant.id, kind: "sells_at", currency: row.currency, price: row.priceMinor, priceDeclared: row.declared, fileIndex: pricingFileIndex})
			if pricingFileIndex >= 0 {
				report.Files[pricingFileIndex].Written++
			}
		}
	}
	if apply {
		for _, file := range report.Files {
			if file.Read != file.Written {
				report.Findings = append(report.Findings, CatalogImportFinding{Path: file.Path, Reason: "source record count does not reconcile", Blocking: true})
			}
		}
		for i := range report.Files {
			report.Files[i].Written = 0
		}
		for _, finding := range report.Findings {
			if finding.Blocking {
				return report, fmt.Errorf("catalog import refused: blocking finding at %s: %s", finding.Path, finding.Reason)
			}
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer func() { _ = tx.Rollback() }()
		now := s.now().UTC().Format(time.RFC3339Nano)
		if actor == "" {
			actor = "operator"
		}
		idMap := make(map[string]string, len(nodes))
		for i := range nodes {
			n := &nodes[i]
			originalID := n.id
			var existingID string
			var existingStatus int32
			lookupErr := tx.QueryRowContext(ctx, `SELECT id,status FROM nodes WHERE kind=? AND name=?`, int32(n.kind), n.name).Scan(&existingID, &existingStatus)
			wrote := false
			switch lookupErr {
			case nil:
				n.id = existingID
				if existingStatus != int32(n.status) {
					if _, err := tx.ExecContext(ctx, `UPDATE nodes SET status=? WHERE id=?`, int32(n.status), existingID); err != nil {
						return nil, err
					}
					if _, err := tx.ExecContext(ctx, `INSERT INTO catalog_audit(id,node_id,actor,prior_status,next_status,reason,created_at) VALUES(?,?,?,?,?,?,?)`, uuid.NewString(), existingID, actor, existingStatus, int32(n.status), "catalog import status upsert", now); err != nil {
						return nil, err
					}
					wrote = true
				}
			case sql.ErrNoRows:
				if _, err := tx.ExecContext(ctx, `INSERT INTO nodes(id,kind,name,status,trigger_id,created_at,actual_account_id) VALUES(?,?,?,?,?,?,?)`, n.id, int32(n.kind), n.name, int32(n.status), "", now, ""); err != nil {
					return nil, err
				}
				wrote = true
			default:
				return nil, lookupErr
			}
			idMap[originalID] = n.id
			if wrote && n.fileIndex >= 0 && n.fileIndex < len(report.Files) {
				report.Files[n.fileIndex].Written++
			}
		}
		for _, edge := range edges {
			fromID, fromOK := idMap[edge.from]
			toID, toOK := idMap[edge.to]
			if !fromOK || !toOK {
				return nil, fmt.Errorf("catalog import edge references an unresolved node: %s -> %s", edge.from, edge.to)
			}
			var existingID, existingCurrency string
			var existingPrice int64
			var existingDeclared int
			lookupErr := tx.QueryRowContext(ctx, `SELECT id,intended_price_minor,currency,intended_price_declared FROM edges WHERE from_id=? AND to_id=? AND kind=?`, fromID, toID, edge.kind).Scan(&existingID, &existingPrice, &existingCurrency, &existingDeclared)
			wrote := false
			nextPrice, nextCurrency, nextDeclared := edge.price, edge.currency, edge.priceDeclared
			if lookupErr == nil {
				if existingDeclared != 0 && !edge.priceDeclared {
					nextPrice, nextCurrency, nextDeclared = existingPrice, existingCurrency, true
				}
				if existingPrice != nextPrice || existingCurrency != nextCurrency || (existingDeclared != 0) != nextDeclared {
					if _, err := tx.ExecContext(ctx, `UPDATE edges SET intended_price_minor=?,currency=?,intended_price_declared=? WHERE id=?`, nextPrice, nextCurrency, boolInt(nextDeclared), existingID); err != nil {
						return nil, err
					}
					wrote = true
				}
			} else if lookupErr == sql.ErrNoRows {
				if _, err := tx.ExecContext(ctx, `INSERT INTO edges(id,from_id,to_id,kind,intended_price_minor,currency,intended_price_declared) VALUES(?,?,?,?,?,?,?)`, uuid.NewString(), fromID, toID, edge.kind, nextPrice, nextCurrency, boolInt(nextDeclared)); err != nil {
					return nil, err
				}
				wrote = true
			} else {
				return nil, lookupErr
			}
			if wrote && edge.fileIndex >= 0 && edge.fileIndex < len(report.Files) {
				report.Files[edge.fileIndex].Written++
			}
		}
		for _, fact := range benchmarkFacts {
			observed := fact.ObservedAt.AsTime().UTC().Format(time.RFC3339Nano)
			var oldValue float64
			var oldObserved, oldDimension string
			var oldStale int32
			lookupErr := tx.QueryRowContext(ctx, `SELECT value,observed_at,stale_after_days,dimension FROM facts WHERE name=?`, fact.Name).Scan(&oldValue, &oldObserved, &oldStale, &oldDimension)
			wrote := lookupErr == sql.ErrNoRows || oldValue != fact.Value || oldObserved != observed || oldStale != fact.StaleAfterDays || oldDimension != fact.Dimension
			if wrote {
				if _, err := tx.ExecContext(ctx, `INSERT INTO facts(name,value,observed_at,stale_after_days,dimension) VALUES(?,?,?,?,?) ON CONFLICT(name) DO UPDATE SET value=excluded.value,observed_at=excluded.observed_at,stale_after_days=excluded.stale_after_days,dimension=excluded.dimension`, fact.Name, fact.Value, observed, fact.StaleAfterDays, fact.Dimension); err != nil {
					return nil, err
				}
			}
			if lookupErr != nil && lookupErr != sql.ErrNoRows {
				return nil, lookupErr
			}
			if wrote && benchmarkFileIndex >= 0 && benchmarkFileIndex < len(report.Files) {
				report.Files[benchmarkFileIndex].Written++
			}
		}
		for _, finding := range report.Findings {
			if _, err := tx.ExecContext(ctx, `INSERT INTO migration_findings(id,node_id,source_file,reference,reason,created_at) SELECT ?,?,?,?,?,? WHERE NOT EXISTS (SELECT 1 FROM migration_findings WHERE source_file=? AND reason=?)`, uuid.NewString(), "", finding.Path, "", finding.Reason, now, finding.Path, finding.Reason); err != nil {
				return nil, err
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		report.Applied = true
	}
	sort.Slice(report.Files, func(i, j int) bool { return report.Files[i].Path < report.Files[j].Path })
	sort.Slice(report.Findings, func(i, j int) bool { return report.Findings[i].Path < report.Findings[j].Path })
	_ = actor
	return report, nil
}

// VerifyCatalog reuses the import parser as the source-of-truth for the
// declared record denominator, then checks the live graph for identity and
// reference drift. Authority-handoff prose has no declared records, so it is
// represented by a zero expected/live row; retired drill nodes are likewise
// outside the source denominator. The verifier never writes findings.
func (s *Store) VerifyCatalog(ctx context.Context, sourcePath string, mode offerspb.SourceMode) (*CatalogVerifyReport, error) {
	importReport, err := s.ImportCatalog(ctx, sourcePath, mode, false, "verify")
	if err != nil {
		return nil, err
	}
	report := &CatalogVerifyReport{Files: make([]CatalogVerifyFile, 0, len(importReport.Files))}
	if root, rootErr := findRepositoryRoot(); rootErr == nil {
		nodes, nodeErr := s.ListNodes(ctx, offerspb.NodeKind_DELIVERABLE, offerspb.Status_STATUS_UNSPECIFIED)
		if nodeErr != nil {
			return nil, nodeErr
		}
		for _, node := range nodes {
			if node.Status == offerspb.Status_RETIRED {
				continue
			}
			if _, statErr := os.Stat(filepath.Join(root, "scenarios", node.Name)); os.IsNotExist(statErr) {
				report.ScenarioGaps = append(report.ScenarioGaps, fmt.Sprintf("%s (rank %d)", node.Name, node.ReleaseRank))
			}
		}
		sort.Strings(report.ScenarioGaps)
	}
	comparableFiles := make([]int, 0)
	expectedNodes := 0
	for i, file := range importReport.Files {
		report.Files = append(report.Files, CatalogVerifyFile{Path: file.Path, Expected: file.Written})
		if file.Written > 0 && file.NodeKind != offerspb.NodeKind_NODE_KIND_UNSPECIFIED {
			comparableFiles = append(comparableFiles, i)
			expectedNodes += file.Written
		}
	}

	missingDeclared := 0
	for _, finding := range importReport.Findings {
		if finding.Blocking && strings.Contains(finding.Reason, "declared source file is missing") {
			missingDeclared++
		}
	}

	// Every blocking finding is drift, whatever path it names.
	//
	// This loop used to sit inside the per-file loop above and count a blocking
	// finding only when its path matched a file already in report.Files. That
	// silently dropped the findings that matter most: point the verifier at the
	// wrong root and the import raises "declared source file is missing" for
	// every manifest path plus "undeclared source file" for everything it does
	// find — 63 blocking findings against docs/monetization/catalogs — none of
	// which named a path in report.Files, so drift stayed 0 and the wrong root
	// reported reconciled=true. A verification that passes when it was aimed at
	// the wrong directory is worse than no verification.
	for _, finding := range importReport.Findings {
		if finding.Blocking {
			report.TotalDrift++
		}
	}

	var liveNodes int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE status != ?`, int32(offerspb.Status_RETIRED)).Scan(&liveNodes); err != nil {
		return nil, err
	}
	if expectedNodes > 0 && len(comparableFiles) > 0 {
		delta := liveNodes - expectedNodes
		report.Files[comparableFiles[0]].Live = report.Files[comparableFiles[0]].Expected + delta
		if delta < 0 {
			delta = -delta
		}
		report.TotalDrift += delta
		if liveNodes > expectedNodes {
			rows, queryErr := s.db.QueryContext(ctx, `SELECT id FROM nodes WHERE status != ? ORDER BY created_at DESC, id DESC LIMIT ?`, int32(offerspb.Status_RETIRED), liveNodes-expectedNodes)
			if queryErr != nil {
				return nil, queryErr
			}
			defer rows.Close()
			for rows.Next() {
				var id string
				if scanErr := rows.Scan(&id); scanErr != nil {
					_ = rows.Close()
					return nil, scanErr
				}
				report.ExtraNodeIds = append(report.ExtraNodeIds, id)
			}
			if closeErr := rows.Close(); closeErr != nil {
				return nil, closeErr
			}
			if rowsErr := rows.Err(); rowsErr != nil {
				return nil, rowsErr
			}
		}
	}

	rows, err := s.db.QueryContext(ctx, `SELECT kind,name,COUNT(*) FROM nodes GROUP BY kind,name HAVING COUNT(*) > 1 ORDER BY kind,name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, name string
		var count int
		if err := rows.Scan(&kind, &name, &count); err != nil {
			_ = rows.Close()
			return nil, err
		}
		report.DuplicateIdentities = append(report.DuplicateIdentities, fmt.Sprintf("(%s,%s) (%d rows)", kind, name, count))
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}
	report.TotalDrift += len(report.DuplicateIdentities)

	rows, err = s.db.QueryContext(ctx, `SELECT e.id FROM edges e LEFT JOIN nodes nf ON nf.id=e.from_id LEFT JOIN nodes nt ON nt.id=e.to_id WHERE nf.id IS NULL OR nt.id IS NULL ORDER BY e.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		report.OrphanEdgeIds = append(report.OrphanEdgeIds, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}
	report.TotalDrift += len(report.OrphanEdgeIds)
	report.Reconciled = report.TotalDrift == 0
	report.Comparable = expectedNodes > 0 && len(comparableFiles) > 0
	if !report.Comparable {
		// The two ways a comparison can be skipped need opposite responses, so
		// they must not share a message. A wrong root is a mistake to correct; a
		// compressed source is the intended end state. Reassuring text on the
		// first case is how a mis-aimed verification gets read as a pass.
		if missingDeclared > 0 {
			report.NotComparableReason = fmt.Sprintf(
				"%d declared source file(s) were not found under this root, so it is not the declared source tree; check --source-path before reading any other field",
				missingDeclared)
		} else {
			report.NotComparableReason = fmt.Sprintf(
				"no declared source file yielded a countable record (%d file(s) inspected), so the source-versus-live node count was not compared; this is expected once sources are compressed to judgment-only prose and their state is read from this scenario",
				len(report.Files))
		}
	}
	return report, nil
}

func findRepositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "scenarios")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "packages")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found from %s", dir)
		}
		dir = parent
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
