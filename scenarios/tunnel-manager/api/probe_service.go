package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ProbeResult holds the outcome of a single liveness probe.
type ProbeResult struct {
	RouteID    int    `json:"route_id"`
	Subdomain  string `json:"subdomain"`
	ProbeType  string `json:"probe_type"` // "internal" or "external"
	Status     string `json:"status"`     // "up", "down", "timeout", "error"
	LatencyMs  int    `json:"latency_ms"`
	StatusCode int    `json:"status_code,omitempty"`
	ErrorMsg   string `json:"error_msg,omitempty"`
}

// ProbeService runs liveness probes against published routes.
type ProbeService struct {
	db         *sql.DB
	routeSvc   *RouteService
	httpClient *http.Client
}

func NewProbeService(db *sql.DB, routeSvc *RouteService) *ProbeService {
	return &ProbeService{
		db:       db,
		routeSvc: routeSvc,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// RunAll executes internal and external probes for all enabled routes concurrently.
func (ps *ProbeService) RunAll(ctx context.Context) ([]ProbeResult, error) {
	routes, err := ps.routeSvc.List()
	if err != nil {
		return nil, fmt.Errorf("probes: %w", err)
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []ProbeResult
	)

	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		wg.Add(1)
		go func(r Route) {
			defer wg.Done()
			// Internal probe (local port)
			internal := ps.probeInternal(ctx, r)
			// External probe (public URL)
			external := ps.probeExternal(ctx, r)

			mu.Lock()
			results = append(results, internal, external)
			mu.Unlock()

			// Persist results
			ps.persistResult(internal)
			ps.persistResult(external)
		}(route)
	}

	wg.Wait()
	return results, nil
}

func (ps *ProbeService) probeInternal(ctx context.Context, route Route) ProbeResult {
	url := fmt.Sprintf("http://localhost:%d%s", route.LocalPort, route.HealthPath)
	return ps.doProbe(ctx, route.ID, route.Subdomain, "internal", url)
}

func (ps *ProbeService) probeExternal(ctx context.Context, route Route) ProbeResult {
	if route.PublicURL == "" {
		return ProbeResult{
			RouteID:   route.ID,
			Subdomain: route.Subdomain,
			ProbeType: "external",
			Status:    "error",
			ErrorMsg:  "no public_url configured",
		}
	}
	url := route.PublicURL + route.HealthPath
	return ps.doProbe(ctx, route.ID, route.Subdomain, "external", url)
}

func (ps *ProbeService) doProbe(ctx context.Context, routeID int, subdomain, probeType, url string) ProbeResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ProbeResult{
			RouteID:   routeID,
			Subdomain: subdomain,
			ProbeType: probeType,
			Status:    "error",
			ErrorMsg:  err.Error(),
		}
	}

	resp, err := ps.httpClient.Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		status := "down"
		if ctx.Err() != nil {
			status = "timeout"
		}
		return ProbeResult{
			RouteID:   routeID,
			Subdomain: subdomain,
			ProbeType: probeType,
			Status:    status,
			LatencyMs: latency,
			ErrorMsg:  err.Error(),
		}
	}
	defer resp.Body.Close()

	probeStatus := "up"
	if resp.StatusCode >= 400 {
		probeStatus = "down"
	}

	return ProbeResult{
		RouteID:    routeID,
		Subdomain:  subdomain,
		ProbeType:  probeType,
		Status:     probeStatus,
		LatencyMs:  latency,
		StatusCode: resp.StatusCode,
	}
}

func (ps *ProbeService) persistResult(pr ProbeResult) {
	var statusCode *int
	if pr.StatusCode != 0 {
		statusCode = &pr.StatusCode
	}
	var errMsg *string
	if pr.ErrorMsg != "" {
		errMsg = &pr.ErrorMsg
	}
	_, _ = ps.db.Exec(
		`INSERT INTO probe_results (route_id, probe_type, status, latency_ms, status_code, error_msg) VALUES ($1, $2, $3, $4, $5, $6)`,
		pr.RouteID, pr.ProbeType, pr.Status, pr.LatencyMs, statusCode, errMsg,
	)
}

// --- HTTP Handler ---

func handleRunProbes(svc *ProbeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		results, err := svc.RunAll(ctx)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []ProbeResult{}
		}

		// Compute summary
		up, down := 0, 0
		for _, pr := range results {
			if pr.Status == "up" {
				up++
			} else {
				down++
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"results": results,
			"summary": map[string]int{
				"total": len(results),
				"up":    up,
				"down":  down,
			},
		})
	}
}
