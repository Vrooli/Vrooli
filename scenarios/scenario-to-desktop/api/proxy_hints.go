package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

type ProxyHint struct {
	URL        string `json:"url"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Message    string `json:"message"`
}

func (s *Server) proxyHintsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	hints := s.collectProxyHints(scenario)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"scenario": scenario,
		"hints":    hints,
	})
}

func (s *Server) collectProxyHints(scenario string) []ProxyHint {
	seen := make(map[string]struct{})
	var hints []ProxyHint

	appendHint := func(h ProxyHint) {
		if h.URL == "" {
			return
		}
		if _, ok := seen[h.URL]; ok {
			return
		}
		seen[h.URL] = struct{}{}
		hints = append(hints, h)
	}

	if cfgHint := s.loadSavedProxyHint(scenario); cfgHint != nil {
		appendHint(*cfgHint)
	}
	if telem := s.loadTelemetryProxyHint(scenario); telem != nil {
		appendHint(*telem)
	}
	if hosts := detectCloudflareHosts(); len(hosts) > 0 {
		for _, host := range hosts {
			candidate := buildProxyURLFromHost(host, scenario)
			if candidate == "" {
				continue
			}
			appendHint(ProxyHint{
				URL:        candidate,
				Source:     "cloudflared",
				Confidence: "medium",
				Message:    fmt.Sprintf("Detected hostname '%s' from ~/.cloudflared/config.yml", host),
			})
		}
	}
	if local := s.buildLocalProxyHint(scenario); local != nil {
		appendHint(*local)
	}

	return hints
}

func (s *Server) loadSavedProxyHint(scenario string) *ProxyHint {
	scenarioRoot := s.resolveScenarioRoot(scenario)
	if scenarioRoot == "" {
		return nil
	}
	cfg, err := loadDesktopConnectionConfig(scenarioRoot)
	if err != nil || cfg == nil {
		return nil
	}
	if cfg.ProxyURL == "" {
		return nil
	}
	return &ProxyHint{
		URL:        cfg.ProxyURL,
		Source:     "saved-config",
		Confidence: "high",
		Message:    "Stored from the last successful desktop configuration",
	}
}

func (s *Server) loadTelemetryProxyHint(scenario string) *ProxyHint {
	telemetryPath := filepath.Join(s.getVrooliRoot(), ".vrooli", "deployment", "telemetry", fmt.Sprintf("%s.jsonl", scenario))
	file, err := os.Open(telemetryPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lastURL string
	for scanner.Scan() {
		line := scanner.Bytes()
		var payload struct {
			Details map[string]any `json:"details"`
		}
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		if payload.Details == nil {
			continue
		}
		if raw, ok := payload.Details["serverUrl"].(string); ok && raw != "" {
			lastURL = raw
		}
	}
	if lastURL == "" {
		return nil
	}
	normalized, err := normalizeProxyURL(lastURL)
	if err != nil {
		normalized = lastURL
	}
	return &ProxyHint{
		URL:        normalized,
		Source:     "telemetry",
		Confidence: "high",
		Message:    "Reported by a previously generated desktop app",
	}
}

func (s *Server) buildLocalProxyHint(scenario string) *ProxyHint {
	analyzer := NewScenarioAnalyzer(s.getVrooliRoot())
	metadata, err := analyzer.AnalyzeScenario(scenario)
	if err != nil {
		return nil
	}
	port := metadata.UIPort
	if port == 0 {
		port = 3000
	}
	url := fmt.Sprintf("http://localhost:%d/", port)
	normalized, err := normalizeProxyURL(url)
	if err != nil {
		normalized = url
	}
	return &ProxyHint{
		URL:        normalized,
		Source:     "local",
		Confidence: "low",
		Message:    "Scenario appears to be running locally on this machine",
	}
}

func detectCloudflareHosts() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	configPath := filepath.Join(homeDir, ".cloudflared", "config.yml")
	file, err := os.Open(configPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hosts []string
	var pendingHost string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasPrefix(line, "hostname:") {
			pendingHost = strings.TrimSpace(strings.TrimPrefix(line, "hostname:"))
			pendingHost = strings.Trim(pendingHost, "\"'")
			continue
		}
		if strings.HasPrefix(line, "service:") {
			if pendingHost != "" {
				hosts = append(hosts, pendingHost)
				pendingHost = ""
			}
		}
	}
	if pendingHost != "" {
		hosts = append(hosts, pendingHost)
	}
	return hosts
}

func buildProxyURLFromHost(host, scenario string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(host), "http://") && !strings.HasPrefix(strings.ToLower(host), "https://") {
		host = "https://" + host
	}
	base := strings.TrimRight(host, "/")
	escaped := url.PathEscape(scenario)
	proxy := fmt.Sprintf("%s/apps/%s/proxy/", base, escaped)
	normalized, err := normalizeProxyURL(proxy)
	if err != nil {
		return proxy
	}
	return normalized
}

func (s *Server) getVrooliRoot() string {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		currentDir, _ := os.Getwd()
		vrooliRoot = filepath.Join(currentDir, "../../..")
	}
	return vrooliRoot
}

func (s *Server) resolveScenarioRoot(scenario string) string {
	if scenario == "" {
		return ""
	}
	root := s.getVrooliRoot()
	path := filepath.Join(root, "scenarios", scenario)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}
