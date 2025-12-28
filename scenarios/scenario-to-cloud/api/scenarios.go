package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ScenarioInfo represents a scenario with its configuration
type ScenarioInfo struct {
	ID          string                  `json:"id"`
	DisplayName string                  `json:"displayName,omitempty"`
	Description string                  `json:"description,omitempty"`
	Ports       map[string]PortConfig   `json:"ports,omitempty"`
}

// PortConfig represents a port configuration from service.json
type PortConfig struct {
	EnvVar      string `json:"env_var,omitempty"`
	Description string `json:"description,omitempty"`
	Range       string `json:"range,omitempty"`
	Port        int    `json:"port,omitempty"`
}

// ServiceJSON represents the structure of .vrooli/service.json
type ServiceJSON struct {
	Service struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
	} `json:"service"`
	Ports map[string]PortConfig `json:"ports"`
}

// ScenariosResponse is the response for GET /scenarios
type ScenariosResponse struct {
	Scenarios []ScenarioInfo `json:"scenarios"`
	Timestamp string         `json:"timestamp"`
}

// ScenarioPortsResponse is the response for GET /scenarios/:id/ports
type ScenarioPortsResponse struct {
	ScenarioID string                `json:"scenario_id"`
	Ports      map[string]PortConfig `json:"ports"`
	Timestamp  string                `json:"timestamp"`
}

// ReachabilityRequest is the request body for POST /validate/reachability
type ReachabilityRequest struct {
	Host   string `json:"host,omitempty"`
	Domain string `json:"domain,omitempty"`
}

// ReachabilityResult represents the result of a single reachability check
type ReachabilityResult struct {
	Target    string `json:"target"`
	Type      string `json:"type"` // "host" or "domain"
	Reachable bool   `json:"reachable"`
	Message   string `json:"message,omitempty"`
	Hint      string `json:"hint,omitempty"`
}

// ReachabilityResponse is the response for POST /validate/reachability
type ReachabilityResponse struct {
	Results   []ReachabilityResult `json:"results"`
	Timestamp string               `json:"timestamp"`
}

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "repo_root_not_found",
			Message: "Unable to locate Vrooli repo root",
			Hint:    err.Error(),
		})
		return
	}

	scenariosDir := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "scenarios_dir_read_failed",
			Message: "Unable to read scenarios directory",
			Hint:    err.Error(),
		})
		return
	}

	var scenarios []ScenarioInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioID := entry.Name()
		serviceJSONPath := filepath.Join(scenariosDir, scenarioID, ".vrooli", "service.json")

		info := ScenarioInfo{
			ID: scenarioID,
		}

		// Try to read service.json for additional metadata
		if data, err := os.ReadFile(serviceJSONPath); err == nil {
			var svc ServiceJSON
			if json.Unmarshal(data, &svc) == nil {
				if svc.Service.DisplayName != "" {
					info.DisplayName = svc.Service.DisplayName
				}
				if svc.Service.Description != "" {
					info.Description = svc.Service.Description
				}
				if len(svc.Ports) > 0 {
					info.Ports = svc.Ports
				}
			}
		}

		scenarios = append(scenarios, info)
	}

	writeJSON(w, http.StatusOK, ScenariosResponse{
		Scenarios: scenarios,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleScenarioPorts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["id"]

	if scenarioID == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_scenario_id",
			Message: "Scenario ID is required",
		})
		return
	}

	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "repo_root_not_found",
			Message: "Unable to locate Vrooli repo root",
			Hint:    err.Error(),
		})
		return
	}

	serviceJSONPath := filepath.Join(repoRoot, "scenarios", scenarioID, ".vrooli", "service.json")
	data, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeAPIError(w, http.StatusNotFound, APIError{
				Code:    "scenario_not_found",
				Message: "Scenario not found or missing service.json",
				Hint:    "Ensure the scenario exists and has a .vrooli/service.json file.",
			})
		} else {
			writeAPIError(w, http.StatusInternalServerError, APIError{
				Code:    "service_json_read_failed",
				Message: "Unable to read service.json",
				Hint:    err.Error(),
			})
		}
		return
	}

	var svc ServiceJSON
	if err := json.Unmarshal(data, &svc); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "service_json_parse_failed",
			Message: "Unable to parse service.json",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, ScenarioPortsResponse{
		ScenarioID: scenarioID,
		Ports:      svc.Ports,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleReachabilityCheck(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[ReachabilityRequest](r.Body, 1<<16)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var results []ReachabilityResult

	// Check host reachability
	if req.Host != "" {
		result := checkHostReachability(ctx, req.Host)
		results = append(results, result)
	}

	// Check domain reachability (DNS resolution)
	if req.Domain != "" {
		result := checkDomainReachability(ctx, req.Domain)
		results = append(results, result)
	}

	writeJSON(w, http.StatusOK, ReachabilityResponse{
		Results:   results,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func checkHostReachability(ctx context.Context, host string) ReachabilityResult {
	result := ReachabilityResult{
		Target: host,
		Type:   "host",
	}

	// Try TCP connection to SSH port (22) as a basic reachability check
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, "22"))
	if err != nil {
		errStr := err.Error()

		// Check for IPv6-specific errors
		if isIPv6(host) && (strings.Contains(errStr, "no route to host") ||
			strings.Contains(errStr, "network is unreachable")) {
			result.Reachable = false
			result.Message = "IPv6 not available"
			result.Hint = ipv6ConnectivityHint
			return result
		}

		// Check if it's a timeout or connection refused
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.Reachable = false
			if isIPv6(host) {
				result.Message = "Connection timed out (IPv6)"
				result.Hint = ipv6ConnectivityHint
			} else {
				result.Message = "Connection timed out"
				result.Hint = "The host may be unreachable, or SSH port 22 may be blocked. You can proceed if the server is not yet configured."
			}
		} else if strings.Contains(errStr, "connection refused") {
			// Connection refused means the host is reachable but SSH isn't running
			result.Reachable = true
			result.Message = "Host reachable, but SSH port 22 is closed"
			result.Hint = "The server is reachable but SSH may not be running yet. Ensure SSH is enabled before deployment."
		} else {
			result.Reachable = false
			result.Message = "Unable to connect"
			result.Hint = "Check that the IP address is correct and the server is online. You can proceed if the server is not yet provisioned."
		}
		return result
	}
	conn.Close()

	result.Reachable = true
	result.Message = "Host is reachable via SSH"
	return result
}

func checkDomainReachability(ctx context.Context, domain string) ReachabilityResult {
	result := ReachabilityResult{
		Target: domain,
		Type:   "domain",
	}

	// Check DNS resolution
	resolver := &net.Resolver{}
	addrs, err := resolver.LookupHost(ctx, domain)
	if err != nil {
		if dnsErr, ok := err.(*net.DNSError); ok {
			if dnsErr.IsNotFound {
				result.Reachable = false
				result.Message = "Domain does not resolve (NXDOMAIN)"
				result.Hint = "DNS is not configured for this domain yet. You can proceed if you plan to configure DNS before deployment."
			} else if dnsErr.IsTimeout {
				result.Reachable = false
				result.Message = "DNS lookup timed out"
				result.Hint = "Unable to verify DNS. You can proceed if you're confident the domain is configured correctly."
			} else {
				result.Reachable = false
				result.Message = "DNS lookup failed"
				result.Hint = "Check that the domain is valid. You can proceed if DNS will be configured later."
			}
		} else {
			result.Reachable = false
			result.Message = "Unable to resolve domain"
			result.Hint = "You can proceed if DNS will be configured before deployment."
		}
		return result
	}

	result.Reachable = true
	result.Message = "Domain resolves to " + strings.Join(addrs, ", ")
	return result
}
