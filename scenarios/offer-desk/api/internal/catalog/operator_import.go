package catalog

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
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

type importedNode struct {
	id     string
	kind   offerspb.NodeKind
	name   string
	status offerspb.Status
}
type importedEdge struct {
	from, to, kind, currency string
	price                    int64
}

var (
	statusMarker = regexp.MustCompile(`(?im)^\s*(?:-\s+)?\*\*Status:`)
	tierHeading  = regexp.MustCompile("(?m)^###\\s+Tier\\s+([1-4])\\s+—\\s+([^\\n(]+)\\s+\\(`?(active|candidate|north-star|retired)`?\\)")
	skuID        = regexp.MustCompile(`(?m)^\*\*SKU ID:\*\*\s*` + "`?" + `([^` + "`" + `\s]+)`)
)

func loadImportManifest() (importManifest, error) {
	data, err := importManifestFS.ReadFile("import-manifest.json")
	if err != nil {
		return importManifest{}, err
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
	default:
		return offerspb.Status_STATUS_UNSPECIFIED, false
	}
}

func nodeKindForManifest(kind string) offerspb.NodeKind {
	switch kind {
	case "channel":
		return offerspb.NodeKind_CHANNEL
	case "revenue-line":
		return offerspb.NodeKind_REVENUE_LINE
	case "deliverable":
		return offerspb.NodeKind_DELIVERABLE
	case "variant":
		return offerspb.NodeKind_VARIANT
	case "membership":
		return offerspb.NodeKind_DELIVERABLE
	default:
		return offerspb.NodeKind_OFFER
	}
}

func (s *Store) ImportCatalog(ctx context.Context, sourcePath string, mode offerspb.SourceMode, apply bool, actor string) (*CatalogImportReport, error) {
	root, err := resolveImportRoot(sourcePath, mode)
	if err != nil {
		return nil, err
	}
	manifest, err := loadImportManifest()
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
	type membership struct {
		scenario string
		skus     []string
	}
	memberships := make([]membership, 0)
	for _, entry := range manifest.Paths {
		path := filepath.ToSlash(filepath.Clean(entry.Path))
		if _, ok := actual[path]; !ok {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		file := CatalogImportFile{Path: path, Cardinality: entry.Cardinality, NodeKind: nodeKindForManifest(entry.Kind)}
		switch entry.Cardinality {
		case "none":
			file.Read, file.Written = 0, 0
		case "one":
			file.Read = 1
			status, found, line := parseImportedStatus(string(body))
			if !found {
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
			n := importedNode{id: uuid.NewString(), kind: file.NodeKind, name: strings.TrimSuffix(name, filepath.Ext(name)), status: status}
			nodes = append(nodes, n)
			byName[n.name] = n
			file.Written = 1
			report.StatusMap = append(report.StatusMap, CatalogImportStatus{Path: path, Status: status, Recognized: true, Line: line})
		case "many":
			if entry.Kind == "variant" {
				for _, match := range tierHeading.FindAllStringSubmatch(string(body), -1) {
					status, ok := statusFromWord(match[3])
					if !ok {
						file.Findings++
						report.Findings = append(report.Findings, CatalogImportFinding{Path: path, Reason: "tier status is unrecognizable", Blocking: true})
						continue
					}
					name := "tier-" + match[1]
					n := importedNode{id: uuid.NewString(), kind: offerspb.NodeKind_VARIANT, name: name, status: status}
					nodes = append(nodes, n)
					byName[n.name] = n
					report.StatusMap = append(report.StatusMap, CatalogImportStatus{Path: path + "#" + name, Status: status, Recognized: true})
				}
				file.Read, file.Written = len(tierHeading.FindAllStringSubmatch(string(body), -1)), len(tierHeading.FindAllStringSubmatch(string(body), -1))
			} else if entry.Kind == "membership" {
				var mapping struct {
					Mappings []struct {
						ScenarioID string `json:"scenarioId"`
						SKUs       []struct {
							SKUID string `json:"skuId"`
						} `json:"skus"`
					} `json:"mappings"`
				}
				if err := json.Unmarshal(body, &mapping); err != nil {
					return nil, fmt.Errorf("parse %s: %w", path, err)
				}
				file.Read = len(mapping.Mappings)
				for _, item := range mapping.Mappings {
					itemSKUs := make([]string, 0, len(item.SKUs))
					for _, sku := range item.SKUs {
						itemSKUs = append(itemSKUs, sku.SKUID)
					}
					memberships = append(memberships, membership{scenario: item.ScenarioID, skus: itemSKUs})
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
	for _, item := range memberships {
		deliverable := importedNode{id: uuid.NewString(), kind: offerspb.NodeKind_DELIVERABLE, name: item.scenario, status: offerspb.Status_ACTIVE}
		nodes = append(nodes, deliverable)
		byName[deliverable.name] = deliverable
		for _, sku := range item.skus {
			fileIndex := -1
			for i := range report.Files {
				if report.Files[i].Path == "catalogs/scenario-sku-map.json" {
					fileIndex = i
					break
				}
			}
			if offer, ok := byName[sku]; ok {
				edges = append(edges, importedEdge{from: deliverable.id, to: offer.id, kind: "belongs_to"})
				if fileIndex >= 0 {
					report.Files[fileIndex].Written++
				}
				report.StatusMap = append(report.StatusMap, CatalogImportStatus{Path: "catalogs/scenario-sku-map.json#" + item.scenario + "/" + sku, Status: offerspb.Status_ACTIVE, Recognized: true})
			} else {
				if fileIndex >= 0 {
					report.Files[fileIndex].Findings++
				}
				report.Findings = append(report.Findings, CatalogImportFinding{Path: "catalogs/scenario-sku-map.json", Reason: "membership names an unresolved SKU " + sku, Blocking: false})
			}
		}
	}
	if business, ok := byName["business"]; ok {
		for _, addon := range []string{"elder-care", "family-with-kids", "property-services"} {
			if n, exists := byName[addon]; exists {
				edges = append(edges, importedEdge{from: n.id, to: business.id, kind: "requires"})
			}
		}
	}
	for _, offerName := range []string{"business", "lifestyle"} {
		if offer, ok := byName[offerName]; ok {
			for _, variantName := range []string{"tier-1", "tier-2", "tier-3", "tier-4"} {
				if variant, exists := byName[variantName]; exists {
					edges = append(edges, importedEdge{from: offer.id, to: variant.id, kind: "sells_at", currency: "USD"})
				}
			}
		}
	}
	if apply {
		for _, file := range report.Files {
			if file.Read != file.Written {
				report.Findings = append(report.Findings, CatalogImportFinding{Path: file.Path, Reason: "source record count does not reconcile", Blocking: true})
			}
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
		for _, n := range nodes {
			if _, err := tx.ExecContext(ctx, `INSERT INTO nodes(id,kind,name,status,trigger_id,created_at,actual_account_id) VALUES(?,?,?,?,?,?,?)`, n.id, int32(n.kind), n.name, int32(n.status), "", now, ""); err != nil {
				return nil, err
			}
		}
		for _, edge := range edges {
			if _, err := tx.ExecContext(ctx, `INSERT INTO edges(id,from_id,to_id,kind,intended_price_minor,currency) VALUES(?,?,?,?,?,?)`, uuid.NewString(), edge.from, edge.to, edge.kind, edge.price, edge.currency); err != nil {
				return nil, err
			}
		}
		for _, finding := range report.Findings {
			if _, err := tx.ExecContext(ctx, `INSERT INTO migration_findings(id,node_id,source_file,reference,reason,created_at) VALUES(?,?,?,?,?,?)`, uuid.NewString(), "", finding.Path, "", finding.Reason, now); err != nil {
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
