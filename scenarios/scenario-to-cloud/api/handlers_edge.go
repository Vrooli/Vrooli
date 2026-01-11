package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/tlsinfo"
)

// DNSCheckResponse is the response from the DNS check endpoint.
type DNSCheckResponse struct {
	OK        bool             `json:"ok"`
	VPSHost   string           `json:"vps_host"`
	VPSIPs    []string         `json:"vps_ips"`
	Domains   []DNSDomainCheck `json:"domains"`
	Message   string           `json:"message"`
	Timestamp string           `json:"timestamp"`
}

// DNSDomainCheck describes DNS status for a single domain variant.
type DNSDomainCheck struct {
	Domain      string                 `json:"domain"`
	Role        string                 `json:"role"` // apex, www, origin
	OK          bool                   `json:"ok"`
	DomainIPs   []string               `json:"domain_ips,omitempty"`
	PointsToVPS bool                   `json:"points_to_vps"`
	Proxied     bool                   `json:"proxied"`
	Message     string                 `json:"message"`
	Hint        string                 `json:"hint,omitempty"`
	HintData    *domain.DNSARecordHint `json:"hint_data,omitempty"`
}

// DNSRecordsResponse is the response from the DNS records endpoint.
type DNSRecordsResponse struct {
	OK        bool                 `json:"ok"`
	Domains   []DNSRecordSetResult `json:"domains"`
	Message   string               `json:"message"`
	Timestamp string               `json:"timestamp"`
}

// DNSRecordSetResult contains DNS records for a single domain.
type DNSRecordSetResult struct {
	Domain  string               `json:"domain"`
	Records *domain.DNSRecordSet `json:"records,omitempty"`
	Error   string               `json:"error,omitempty"`
}

// CaddyControlRequest is the request body for Caddy control actions.
type CaddyControlRequest struct {
	Action string `json:"action"` // start, stop, restart, reload
}

// CaddyControlResponse is the response from Caddy control actions.
type CaddyControlResponse struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"`
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

// TLSInfoResponse contains detailed TLS certificate information.
type TLSInfoResponse struct {
	OK            bool               `json:"ok"`
	Domain        string             `json:"domain"`
	Valid         bool               `json:"valid"`
	Validation    string             `json:"validation,omitempty"`
	Issuer        string             `json:"issuer,omitempty"`
	Subject       string             `json:"subject,omitempty"`
	NotBefore     string             `json:"not_before,omitempty"`
	NotAfter      string             `json:"not_after,omitempty"`
	DaysRemaining int                `json:"days_remaining"`
	SerialNumber  string             `json:"serial_number,omitempty"`
	SANs          []string           `json:"sans,omitempty"`
	Error         string             `json:"error,omitempty"`
	ALPN          *tlsinfo.ALPNCheck `json:"alpn,omitempty"`
	Timestamp     string             `json:"timestamp"`
}

// TLSRenewResponse is the response from TLS certificate renewal.
type TLSRenewResponse struct {
	OK        bool   `json:"ok"`
	Domain    string `json:"domain"`
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

// handleDNSCheck verifies that the edge domain points to the VPS.
// GET /api/v1/deployments/{id}/edge/dns-check
func (s *Server) handleDNSCheck(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	repo := s.deploymentRepo
	if repo == nil {
		repo = s.repo
	}
	if repo == nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Deployment repository unavailable"})
		return
	}

	// Get deployment
	deployment, err := repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to get deployment"})
		return
	}
	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{Message: "Deployment not found"})
		return
	}

	// Parse manifest
	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to parse manifest"})
		return
	}

	normalized, _ := manifest.ValidateAndNormalize(m)
	if normalized.Target.VPS == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Message: "Deployment does not have a VPS target"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	vpsHost := normalized.Target.VPS.Host
	domainName := normalized.Edge.Domain

	resp := buildDNSCheckResponse(ctx, s.dnsService, domainName, vpsHost)
	resp.Timestamp = time.Now().UTC().Format(time.RFC3339)
	httputil.WriteJSON(w, http.StatusOK, resp)
}

// handleDNSRecords returns common DNS records for the edge domain variants.
// GET /api/v1/deployments/{id}/edge/dns-records
func (s *Server) handleDNSRecords(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	repo := s.deploymentRepo
	if repo == nil {
		repo = s.repo
	}
	if repo == nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Deployment repository unavailable"})
		return
	}

	deployment, err := repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to get deployment"})
		return
	}
	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{Message: "Deployment not found"})
		return
	}

	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to parse manifest"})
		return
	}

	normalized, _ := manifest.ValidateAndNormalize(m)
	if strings.TrimSpace(normalized.Edge.Domain) == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Message: "Deployment does not have an edge domain"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	baseDomain := dns.BaseDomain(normalized.Edge.Domain)
	domains := []string{baseDomain}
	wwwDomain := "www." + baseDomain
	if wwwDomain != baseDomain {
		domains = append(domains, wwwDomain)
	}
	domains = append(domains, "do-origin."+baseDomain)

	resp := DNSRecordsResponse{
		OK:      true,
		Domains: make([]DNSRecordSetResult, 0, len(domains)),
	}

	for _, domainName := range domains {
		records, err := dns.LookupRecordSet(ctx, domainName)
		if err != nil {
			resp.OK = false
			resp.Domains = append(resp.Domains, DNSRecordSetResult{
				Domain: domainName,
				Error:  err.Error(),
			})
			continue
		}
		resp.Domains = append(resp.Domains, DNSRecordSetResult{
			Domain:  domainName,
			Records: &records,
		})
	}

	if resp.OK {
		resp.Message = "DNS records fetched"
	} else {
		resp.Message = "Some DNS records could not be fetched"
	}
	resp.Timestamp = time.Now().UTC().Format(time.RFC3339)
	httputil.WriteJSON(w, http.StatusOK, resp)
}

func buildDNSCheckResponse(ctx context.Context, svc dns.Service, domainName, vpsHost string) DNSCheckResponse {
	eval := dns.Evaluate(ctx, svc, domainName, vpsHost)

	response := DNSCheckResponse{
		VPSHost: eval.VPS.Host,
		VPSIPs:  eval.VPS.IPs,
		Domains: make([]DNSDomainCheck, 0, 4),
	}

	checks := make([]DNSDomainCheck, 0, 4)
	for _, status := range eval.Statuses {
		if status.Role != "apex" && status.Role != "www" && status.Role != "origin" && status.Role != "edge" {
			continue
		}
		checks = append(checks, buildDomainCheck(status, eval.VPS))
	}
	response.Domains = checks

	response.OK = true
	if eval.VPS.Error != nil {
		response.OK = false
		response.Message = fmt.Sprintf("Failed to resolve VPS host: %s", eval.VPS.Error.Message)
		return response
	}
	for _, check := range checks {
		if !check.OK {
			response.OK = false
			break
		}
	}
	if response.OK {
		response.Message = "DNS checks passed"
	} else {
		response.Message = "DNS checks need attention"
	}
	return response
}

func buildDomainCheck(status dns.DomainStatus, vpsLookup domain.DNSLookupResult) DNSDomainCheck {
	check := DNSDomainCheck{
		Domain:      status.Lookup.Host,
		Role:        status.Role,
		DomainIPs:   status.Lookup.IPs,
		PointsToVPS: status.PointsToVPS,
		Proxied:     status.Proxied,
	}
	if status.Lookup.Error != nil {
		check.OK = false
		check.Message = fmt.Sprintf("Failed to resolve %s: %s", status.Host, status.Lookup.Error.Message)
		return check
	}

	if status.AllowProxy && status.Proxied {
		check.OK = true
		check.Message = fmt.Sprintf("%s resolves to Cloudflare proxy", status.Lookup.Host)
		return check
	}

	if vpsLookup.Error != nil {
		check.OK = false
		check.Message = fmt.Sprintf("VPS host unresolved; cannot verify %s", status.Lookup.Host)
		return check
	}

	if status.PointsToVPS {
		check.OK = true
		check.Message = fmt.Sprintf("%s points to the VPS", status.Lookup.Host)
		return check
	}

	check.OK = false
	check.Message = fmt.Sprintf(
		"%s resolves to %s, not your VPS (%s)",
		status.Lookup.Host,
		strings.Join(status.Lookup.IPs, ", "),
		strings.Join(vpsLookup.IPs, ", "),
	)
	if len(vpsLookup.IPs) > 0 {
		hint, hintData := dns.BuildARecordHint(status.Lookup.Host, vpsLookup.IPs[0])
		check.Hint = hint
		check.HintData = &hintData
	}
	return check
}

// handleCaddyControl handles Caddy service control actions.
// POST /api/v1/deployments/{id}/edge/caddy
func (s *Server) handleCaddyControl(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req CaddyControlRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}

	// Validate action
	validActions := map[string]string{
		"start":   "systemctl start caddy",
		"stop":    "systemctl stop caddy",
		"restart": "systemctl restart caddy",
		"reload":  "systemctl reload caddy",
	}

	cmd, ok := validActions[req.Action]
	if !ok {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Message: "Invalid action",
			Hint:    "Valid actions: start, stop, restart, reload",
		})
		return
	}

	repo := s.deploymentRepo
	if repo == nil {
		repo = s.repo
	}
	if repo == nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Deployment repository unavailable"})
		return
	}

	// Get deployment
	deployment, err := repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to get deployment"})
		return
	}
	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{Message: "Deployment not found"})
		return
	}

	// Parse manifest
	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to parse manifest"})
		return
	}

	normalized, _ := manifest.ValidateAndNormalize(m)
	if normalized.Target.VPS == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Message: "Deployment does not have a VPS target"})
		return
	}

	cfg := ssh.ConfigFromManifest(normalized)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := s.sshRunner.Run(ctx, cfg, cmd)

	resp := CaddyControlResponse{
		Action:    req.Action,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil || result.ExitCode != 0 {
		resp.OK = false
		resp.Message = fmt.Sprintf("Failed to %s Caddy", req.Action)
		if result.Stderr != "" {
			resp.Output = result.Stderr
		}
	} else {
		resp.OK = true
		resp.Message = fmt.Sprintf("Caddy %sed successfully", req.Action)
		if result.Stdout != "" {
			resp.Output = result.Stdout
		}
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// handleTLSInfo retrieves detailed TLS certificate information.
// GET /api/v1/deployments/{id}/edge/tls
func (s *Server) handleTLSInfo(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	repo := s.deploymentRepo
	if repo == nil {
		repo = s.repo
	}
	if repo == nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Deployment repository unavailable"})
		return
	}

	// Get deployment
	deployment, err := repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to get deployment"})
		return
	}
	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{Message: "Deployment not found"})
		return
	}

	// Parse manifest
	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to parse manifest"})
		return
	}

	normalized, _ := manifest.ValidateAndNormalize(m)
	domainName := normalized.Edge.Domain

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := TLSInfoResponse{
		Domain:     domainName,
		Validation: "time_only",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	snapshot, err := tlsinfo.RunSnapshot(ctx, domainName, s.tlsService, s.tlsALPNRunner)
	resp.ALPN = &snapshot.ALPN
	if err != nil {
		resp.OK = false
		resp.Valid = false
		resp.Error = fmt.Sprintf("TLS probe failed: %v", err)
		httputil.WriteJSON(w, http.StatusOK, resp)
		return
	}

	resp.Valid = snapshot.Probe.Valid
	resp.Issuer = snapshot.Probe.Issuer
	resp.Subject = snapshot.Probe.Subject
	resp.NotBefore = snapshot.Probe.NotBefore
	resp.NotAfter = snapshot.Probe.NotAfter
	resp.DaysRemaining = snapshot.Probe.DaysRemaining
	resp.SerialNumber = snapshot.Probe.SerialNumber
	resp.SANs = snapshot.Probe.SANs
	resp.OK = true

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// handleTLSRenew forces a TLS certificate renewal via Caddy.
// POST /api/v1/deployments/{id}/edge/tls/renew
func (s *Server) handleTLSRenew(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	repo := s.deploymentRepo
	if repo == nil {
		repo = s.repo
	}
	if repo == nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Deployment repository unavailable"})
		return
	}

	// Get deployment
	deployment, err := repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to get deployment"})
		return
	}
	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{Message: "Deployment not found"})
		return
	}

	// Parse manifest
	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Message: "Failed to parse manifest"})
		return
	}

	normalized, _ := manifest.ValidateAndNormalize(m)
	if normalized.Target.VPS == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Message: "Deployment does not have a VPS target"})
		return
	}

	domainName := normalized.Edge.Domain
	cfg := ssh.ConfigFromManifest(normalized)
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// Caddy command to force certificate renewal
	// caddy reload will re-obtain certificates if needed
	// We can also try caddy trust to install the root CA (for local dev)
	cmd := fmt.Sprintf(
		"caddy trust 2>/dev/null; "+
			"systemctl reload caddy && "+
			"sleep 3 && "+
			"curl -sf https://%s >/dev/null && echo 'Certificate valid'",
		domainName,
	)

	result, err := s.sshRunner.Run(ctx, cfg, cmd)

	resp := TLSRenewResponse{
		Domain:    domainName,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil {
		resp.OK = false
		resp.Message = "Failed to renew TLS certificate"
		resp.Output = err.Error()
	} else if result.ExitCode != 0 {
		resp.OK = false
		resp.Message = "Certificate renewal may have failed"
		resp.Output = result.Stderr
		if result.Stdout != "" {
			resp.Output = result.Stdout + "\n" + result.Stderr
		}
	} else {
		resp.OK = true
		resp.Message = "TLS certificate renewed/validated successfully"
		resp.Output = result.Stdout
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}
