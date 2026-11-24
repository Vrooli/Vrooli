// Package dependencyrationale exposes declaration-backed reasons for
// cross-scenario dependency edges. It reads the canonical service manifest,
// keeps each source field distinct, and never infers a rationale from graph
// connectivity alone.
package dependencyrationale

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

const (
	defaultLimit   = 10
	maximumLimit   = 25
	maximumBodyLen = 32 << 10
)

type SearchRequest struct {
	Query    string `json:"query"`
	Scenario string `json:"scenario,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type SearchResponse struct {
	Query        string        `json:"query"`
	Scenario     string        `json:"scenario,omitempty"`
	Results      []Result      `json:"results"`
	Scan         ScanEvidence  `json:"scan"`
	Degradations []Degradation `json:"degradations"`
}

type Result struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Score       float64        `json:"score"`
	Snippet     string         `json:"snippet"`
	Path        string         `json:"path"`
	Consumer    string         `json:"consumer"`
	Dependency  string         `json:"dependency"`
	Type        string         `json:"type,omitempty"`
	Required    bool           `json:"required"`
	Enabled     bool           `json:"enabled"`
	RuntimeOnly bool           `json:"runtimeOnly"`
	Evidence    EvidenceFields `json:"evidence"`
	Source      SourceEvidence `json:"source"`
}

type EvidenceFields struct {
	Description          string `json:"description,omitempty"`
	Purpose              string `json:"purpose,omitempty"`
	RuntimeOnlyRationale string `json:"runtimeOnlyRationale,omitempty"`
	DegradedBehavior     string `json:"degradedBehavior,omitempty"`
}

type SourceEvidence struct {
	Owner       string    `json:"owner"`
	Basis       string    `json:"basis"`
	Path        string    `json:"path"`
	Fingerprint string    `json:"fingerprint"`
	ObservedAt  time.Time `json:"observedAt"`
}

type ScanEvidence struct {
	Basis              string `json:"basis"`
	ScenariosScanned   int    `json:"scenariosScanned"`
	ScenariosFailed    int    `json:"scenariosFailed"`
	Complete           bool   `json:"complete"`
	RuntimeStatusKnown bool   `json:"runtimeStatusKnown"`
}

type Degradation struct {
	Scenario string `json:"scenario,omitempty"`
	Code     string `json:"code"`
	Reason   string `json:"reason"`
}

// RegisterHTTPRoutes mounts a read-only Search Hub adapter under the caller's
// versioned API group. The input is an owner-controlled scenarios directory.
func RegisterHTTPRoutes(api gin.IRoutes, scenariosDir func() string) {
	api.POST("/dependencies/rationale/search", func(c *gin.Context) {
		searchHandler(c, scenariosDir)
	})
}

func searchHandler(c *gin.Context, scenariosDir func() string) {
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, maximumBodyLen+1))
	decoder.DisallowUnknownFields()
	var req SearchRequest
	if err := decoder.Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid search request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request must contain one JSON object"})
		return
	}
	if len(req.Query) > maximumBodyLen {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search request exceeds size limit"})
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	req.Scenario = strings.TrimSpace(req.Scenario)
	limit := req.Limit
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be positive"})
		return
	}
	if limit > maximumLimit {
		limit = maximumLimit
	}
	response := SearchResponse{
		Query: req.Query, Scenario: req.Scenario,
		Results: []Result{}, Degradations: []Degradation{},
		Scan: ScanEvidence{Basis: "canonical_service_manifest", Complete: true, RuntimeStatusKnown: false},
	}
	if req.Query == "" {
		response.Degradations = append(response.Degradations, Degradation{Code: "query_required", Reason: "provide a query for declared dependency rationale"})
		c.JSON(http.StatusOK, response)
		return
	}
	root := ""
	if scenariosDir != nil {
		root = strings.TrimSpace(scenariosDir())
	}
	if root == "" {
		response.Scan.Complete = false
		response.Degradations = append(response.Degradations, Degradation{Code: "scenario_root_unavailable", Reason: "the owner scenarios directory is not configured"})
		c.JSON(http.StatusOK, response)
		return
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		response.Scan.Complete = false
		response.Degradations = append(response.Degradations, Degradation{Code: "scenario_root_unavailable", Reason: "the owner scenarios directory could not be read"})
		c.JSON(http.StatusOK, response)
		return
	}
	now := time.Now().UTC()
	queryTerms := terms(req.Query)
	scenarioFound := false
	for _, entry := range entries {
		if !entry.IsDir() || (req.Scenario != "" && entry.Name() != req.Scenario) {
			continue
		}
		scenarioFound = true
		response.Scan.ScenariosScanned++
		manifestPath := filepath.Join(root, entry.Name(), ".vrooli", "service.json")
		info, statErr := os.Lstat(manifestPath)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				if req.Scenario != "" {
					appendScanFailure(&response, entry.Name(), errors.New("scoped service manifest is missing"))
				}
				continue
			}
			appendScanFailure(&response, entry.Name(), statErr)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			appendScanFailure(&response, entry.Name(), errors.New("service manifest is not a regular file"))
			continue
		}
		manifest, err := scenariomodel.LoadServiceManifest(manifestPath)
		if err != nil {
			appendScanFailure(&response, entry.Name(), err)
			continue
		}
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			appendScanFailure(&response, entry.Name(), err)
			continue
		}
		fingerprint := sha256.Sum256(raw)
		relativePath, err := filepath.Rel(root, manifestPath)
		if err != nil || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
			appendScanFailure(&response, entry.Name(), errors.New("manifest path is outside owner scenarios root"))
			continue
		}
		for dependencyName, dependency := range manifest.Dependencies.Scenarios {
			evidence := EvidenceFields{
				Description: dependency.Description, Purpose: dependency.Purpose,
				RuntimeOnlyRationale: dependency.RuntimeOnlyRationale,
				DegradedBehavior:     dependency.DegradedBehavior,
			}
			if !hasEvidence(evidence) {
				continue
			}
			score := relevance(queryTerms, terms(strings.Join([]string{
				entry.Name(), dependencyName, evidence.Description, evidence.Purpose,
				evidence.RuntimeOnlyRationale, evidence.DegradedBehavior,
			}, " ")))
			if score == 0 {
				continue
			}
			reason := evidence.RuntimeOnlyRationale
			if reason == "" {
				reason = evidence.Purpose
			}
			if reason == "" {
				reason = evidence.Description
			}
			if reason == "" {
				reason = evidence.DegradedBehavior
			}
			response.Results = append(response.Results, Result{
				ID:       "dependency-rationale:" + entry.Name() + ":" + dependencyName,
				Title:    entry.Name() + " → " + dependencyName,
				Score:    score,
				Snippet:  strings.TrimSpace(reason),
				Path:     filepath.ToSlash(relativePath),
				Consumer: entry.Name(), Dependency: dependencyName,
				Type: dependency.Type, Required: dependency.Required,
				Enabled: dependency.Enabled, RuntimeOnly: dependency.RuntimeOnly,
				Evidence: evidence,
				Source:   SourceEvidence{Owner: entry.Name(), Basis: "service_manifest.dependency_declaration", Path: filepath.ToSlash(relativePath), Fingerprint: "sha256:" + hex.EncodeToString(fingerprint[:]), ObservedAt: now},
			})
		}
	}
	if req.Scenario != "" && !scenarioFound {
		response.Scan.Complete = false
		response.Degradations = append(response.Degradations, Degradation{Scenario: req.Scenario, Code: "scenario_not_found", Reason: "the requested scenario does not exist in the owner scenarios directory"})
	}
	sort.SliceStable(response.Results, func(i, j int) bool {
		if response.Results[i].Score != response.Results[j].Score {
			return response.Results[i].Score > response.Results[j].Score
		}
		if response.Results[i].Consumer != response.Results[j].Consumer {
			return response.Results[i].Consumer < response.Results[j].Consumer
		}
		return response.Results[i].Dependency < response.Results[j].Dependency
	})
	if len(response.Results) > limit {
		response.Results = response.Results[:limit]
	}
	c.JSON(http.StatusOK, response)
}

func appendScanFailure(response *SearchResponse, scenario string, err error) {
	response.Scan.ScenariosFailed++
	response.Scan.Complete = false
	response.Degradations = append(response.Degradations, Degradation{
		Scenario: scenario, Code: "manifest_unavailable",
		Reason: "the scenario manifest could not be read as authoritative dependency evidence",
	})
}

func hasEvidence(e EvidenceFields) bool {
	return strings.TrimSpace(e.Description) != "" || strings.TrimSpace(e.Purpose) != "" ||
		strings.TrimSpace(e.RuntimeOnlyRationale) != "" || strings.TrimSpace(e.DegradedBehavior) != ""
}

func terms(value string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, term := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		if len(term) >= 2 {
			out[term] = struct{}{}
		}
	}
	return out
}

func relevance(query, evidence map[string]struct{}) float64 {
	if len(query) == 0 {
		return 0
	}
	matched := 0
	for term := range query {
		if _, ok := evidence[term]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(query))
}
