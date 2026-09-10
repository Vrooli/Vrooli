package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/vps"

	"github.com/gorilla/mux"
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

// handleCaddyControl controls the edge proxy through the privilege
// broker's caddy actions (validate, reload). Start, stop and restart of the
// caddy unit have no broker action and are refused with the owner that
// would have to exist; the route apply/rollback verbs already validate and
// reload the proxy for every deployment change.
// POST /api/v1/deployments/{id}/edge/caddy
func (s *Server) handleCaddyControl(w http.ResponseWriter, r *http.Request) {
	var req CaddyControlRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	brokerActions := map[string]string{"reload": "edge.caddy.reload", "validate": "edge.caddy.validate"}
	action, ok := brokerActions[req.Action]
	if !ok {
		switch req.Action {
		case "start", "stop", "restart":
			apierrors.Write(w, apierrors.Newf(apierrors.CodeUnsupportedCapability, "Caddy %s has no target owner action; reload and validate are the supported edge controls", req.Action).
				WithDetail("required_owner", "privilegebroker edge.caddy."+req.Action))
		default:
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Message: "Invalid action", Hint: "Valid actions: reload, validate"})
		}
		return
	}
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	output, actErr := managementRepair(ctx, s.runtimeFor(dc), action, map[string]any{"caddy": map[string]any{"deployment_id": dc.ID}})
	resp := CaddyControlResponse{Action: req.Action, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	if actErr != nil {
		resp.OK = false
		resp.Message = fmt.Sprintf("Failed to %s Caddy", req.Action)
		resp.Output = actErr.Message
		httputil.WriteJSON(w, http.StatusOK, resp)
		return
	}
	resp.OK = true
	resp.Message = fmt.Sprintf("Caddy %s completed through the target owner", req.Action)
	resp.Output = output
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

	snapshot, err := tlsinfo.RunSnapshot(ctx, domainName, s.tlsService, s.tlsALPNRunner)
	resp := buildTLSInfoResponse(domainName, snapshot, err)
	resp.Timestamp = time.Now().UTC().Format(time.RFC3339)

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// handleTLSRenew asks the edge proxy to reload through the target owner
// (Caddy renews on reload) and verifies the domain from the cloud side.
// POST /api/v1/deployments/{id}/edge/tls/renew
func (s *Server) handleTLSRenew(w http.ResponseWriter, r *http.Request) {
	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return
	}
	domainName := dc.Manifest.Edge.Domain
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	verify := func(ctx context.Context, domain string) error {
		if s.tlsService == nil {
			return nil
		}
		probe, err := s.tlsService.Probe(ctx, domain)
		if err != nil {
			return err
		}
		if !probe.Valid {
			return fmt.Errorf("certificate for %s did not validate: %s", domain, probe.ValidationError)
		}
		return nil
	}
	result := vps.RunCaddyTLSRenew(ctx, s.runtimeFor(dc), dc.ID, domainName, verify)

	resp := TLSRenewResponse{
		Domain:    domainName,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		OK:        result.OK,
		Message:   result.Message,
		Output:    result.Output,
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}
