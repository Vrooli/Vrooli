// Package resourceinventory exposes declaration-backed resource anatomy from
// the canonical Vrooli resource catalog. It never probes runtime health.
package resourceinventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/vrooli/vrooli/internal/resources/catalog"
)

const (
	defaultLimit = 10
	maximumLimit = 25
	maximumBody  = 32 << 10
)

type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type SearchResponse struct {
	Query        string        `json:"query"`
	Results      []Result      `json:"results"`
	Scan         ScanEvidence  `json:"scan"`
	Degradations []Degradation `json:"degradations"`
}

type Result struct {
	ID                 string                  `json:"id"`
	Title              string                  `json:"title"`
	Score              float64                 `json:"score"`
	Snippet            string                  `json:"snippet"`
	Path               string                  `json:"path"`
	Resource           catalog.ManifestAnatomy `json:"resource"`
	EvidenceBasis      string                  `json:"evidenceBasis"`
	RuntimeStatusKnown bool                    `json:"runtimeStatusKnown"`
}

type ScanEvidence struct {
	Basis              string `json:"basis"`
	ResourcesScanned   int    `json:"resourcesScanned"`
	ResourcesFailed    int    `json:"resourcesFailed"`
	Complete           bool   `json:"complete"`
	RuntimeStatusKnown bool   `json:"runtimeStatusKnown"`
}

type Degradation struct {
	Resource string `json:"resource,omitempty"`
	Code     string `json:"code"`
	Reason   string `json:"reason"`
}

func RegisterHTTPRoutes(api gin.IRoutes, repoRoot func() string) {
	api.POST("/resources/inventory/search", func(c *gin.Context) {
		searchHandler(c, repoRoot)
	})
}

func searchHandler(c *gin.Context, repoRoot func() string) {
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, maximumBody+1))
	decoder.DisallowUnknownFields()
	var req SearchRequest
	if err := decoder.Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inventory search request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request must contain one JSON object"})
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}
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
		Query: req.Query, Results: []Result{}, Degradations: []Degradation{},
		Scan: ScanEvidence{Basis: "resource_catalog_and_resource_manifest", Complete: true, RuntimeStatusKnown: false},
	}
	root := ""
	if repoRoot != nil {
		root = strings.TrimSpace(repoRoot())
	}
	if root == "" {
		response.Scan.Complete = false
		response.Degradations = append(response.Degradations, Degradation{Code: "repository_root_unavailable", Reason: "resource catalog root is not configured"})
		c.JSON(http.StatusOK, response)
		return
	}
	report, err := catalog.New(root).DiscoverManifestAnatomy(catalog.DiscoverOptions{})
	if err != nil {
		response.Scan.Complete = false
		response.Degradations = append(response.Degradations, Degradation{Code: "resource_catalog_unavailable", Reason: "canonical resource catalog could not be read"})
		c.JSON(http.StatusOK, response)
		return
	}
	response.Scan.ResourcesScanned = len(report.Items) + len(report.Failures)
	response.Scan.ResourcesFailed = len(report.Failures)
	response.Scan.Complete = len(report.Failures) == 0
	for _, failure := range report.Failures {
		response.Degradations = append(response.Degradations, Degradation{
			Resource: failure.Name, Code: "manifest_unavailable",
			Reason: "resource declaration could not be loaded as authoritative anatomy",
		})
	}
	query := terms(req.Query)
	for _, resource := range report.Items {
		text := resourceSearchText(resource)
		score := overlap(query, terms(text))
		if score == 0 {
			continue
		}
		response.Results = append(response.Results, Result{
			ID:                 "resource-inventory:" + resource.ResourceID,
			Title:              resource.ResourceID,
			Score:              score,
			Snippet:            resourceSnippet(resource),
			Path:               resource.SourcePath,
			Resource:           resource,
			EvidenceBasis:      "declared_resource_manifest_and_catalog",
			RuntimeStatusKnown: false,
		})
	}
	sort.SliceStable(response.Results, func(i, j int) bool {
		if response.Results[i].Score != response.Results[j].Score {
			return response.Results[i].Score > response.Results[j].Score
		}
		return response.Results[i].Resource.ResourceID < response.Results[j].Resource.ResourceID
	})
	if len(response.Results) > limit {
		response.Results = response.Results[:limit]
	}
	c.JSON(http.StatusOK, response)
}

func resourceSearchText(resource catalog.ManifestAnatomy) string {
	var b strings.Builder
	write := func(value string) { b.WriteString(value); b.WriteByte(' ') }
	write(resource.ResourceID)
	write(resource.ManifestName)
	write(resource.DisplayName)
	write(resource.Description)
	write(resource.Driver)
	write(resource.ConfigDescription)
	write(strings.Join(resource.Dependencies, " "))
	for _, port := range resource.Ports {
		write(port.Name)
		write(port.Protocol)
		write(strings.Join([]string{itoa(port.Host), itoa(port.Container)}, " "))
	}
	for _, check := range resource.HealthChecks {
		write(check.Type)
		write(check.Kind)
	}
	return b.String()
}

func resourceSnippet(resource catalog.ManifestAnatomy) string {
	parts := []string{"Declared resource " + resource.ResourceID + " uses driver " + resource.Driver + "."}
	for _, port := range resource.Ports {
		label := port.Name
		if label == "" {
			label = "unnamed"
		}
		parts = append(parts, "Port "+label+" declares host "+itoa(port.Host)+" and container "+itoa(port.Container)+" ("+port.Protocol+").")
	}
	if len(resource.HealthChecks) > 0 {
		parts = append(parts, itoa(len(resource.HealthChecks))+" health-check declaration(s).")
	}
	if len(resource.Dependencies) > 0 {
		parts = append(parts, "Declared resource dependencies: "+strings.Join(resource.Dependencies, ", ")+".")
	}
	if !resource.IdentityConsistent {
		parts = append(parts, "Manifest name differs from the canonical catalog slug; the result ID uses the catalog slug.")
	}
	parts = append(parts, "Runtime condition is not observed by this manifest query.")
	return strings.Join(parts, " ")
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

func overlap(query, evidence map[string]struct{}) float64 {
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

func itoa(value int) string {
	return fmt.Sprintf("%d", value)
}
