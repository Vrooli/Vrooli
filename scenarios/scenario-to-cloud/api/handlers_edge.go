package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
)

// DNSCheckResponse is the response from the DNS check endpoint.
type DNSCheckResponse struct {
	OK          bool                   `json:"ok"`
	Domain      string                 `json:"domain"`
	VPSHost     string                 `json:"vps_host"`
	DomainIPs   []string               `json:"domain_ips"`
	VPSIPs      []string               `json:"vps_ips"`
	PointsToVPS bool                   `json:"points_to_vps"`
	Message     string                 `json:"message"`
	Hint        string                 `json:"hint,omitempty"`
	HintData    *domain.DNSARecordHint `json:"hint_data,omitempty"`
	Timestamp   string                 `json:"timestamp"`
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
	OK            bool     `json:"ok"`
	Domain        string   `json:"domain"`
	Valid         bool     `json:"valid"`
	Issuer        string   `json:"issuer,omitempty"`
	Subject       string   `json:"subject,omitempty"`
	NotBefore     string   `json:"not_before,omitempty"`
	NotAfter      string   `json:"not_after,omitempty"`
	DaysRemaining int      `json:"days_remaining"`
	SerialNumber  string   `json:"serial_number,omitempty"`
	SANs          []string `json:"sans,omitempty"`
	Error         string   `json:"error,omitempty"`
	Timestamp     string   `json:"timestamp"`
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

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{Message: "Failed to get deployment"})
		return
	}
	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{Message: "Deployment not found"})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{Message: "Failed to parse manifest"})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{Message: "Deployment does not have a VPS target"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	vpsHost := normalized.Target.VPS.Host
	domain := normalized.Edge.Domain

	compare := s.dnsService.CompareDomainToVPS(ctx, domain, vpsHost)
	resp := buildDNSCheckResponse(compare)
	resp.Timestamp = time.Now().UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusOK, resp)
}

func buildDNSCheckResponse(compare domain.DNSComparisonResult) DNSCheckResponse {
	response := DNSCheckResponse{
		Domain:      compare.Domain.Host,
		VPSHost:     compare.VPS.Host,
		DomainIPs:   compare.Domain.IPs,
		VPSIPs:      compare.VPS.IPs,
		PointsToVPS: compare.PointsToVPS,
	}

	if compare.VPS.Error != nil {
		response.OK = false
		response.Message = fmt.Sprintf("Failed to resolve VPS host: %s", compare.VPS.Error.Message)
		return response
	}

	if compare.Domain.Error != nil {
		response.OK = false
		response.Message = fmt.Sprintf("Failed to resolve domain: %s", compare.Domain.Error.Message)
		if len(compare.VPS.IPs) > 0 {
			hint, hintData := dns.BuildARecordHint(compare.Domain.Host, compare.VPS.IPs[0])
			response.Hint = hint
			response.HintData = &hintData
		}
		return response
	}

	response.OK = true
	if compare.PointsToVPS {
		response.Message = fmt.Sprintf("Domain %s correctly points to VPS (%s)", compare.Domain.Host, strings.Join(compare.Domain.IPs, ", "))
	} else {
		response.Message = fmt.Sprintf("Domain %s resolves to %s, not your VPS (%s)", compare.Domain.Host, strings.Join(compare.Domain.IPs, ", "), strings.Join(compare.VPS.IPs, ", "))
		if len(compare.VPS.IPs) > 0 {
			hint, hintData := dns.BuildARecordHint(compare.Domain.Host, compare.VPS.IPs[0])
			response.Hint = hint
			response.HintData = &hintData
		}
	}

	return response
}

// handleCaddyControl handles Caddy service control actions.
// POST /api/v1/deployments/{id}/edge/caddy
func (s *Server) handleCaddyControl(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req CaddyControlRequest
	if !decodeRequestBody(w, r, &req) {
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
		writeAPIError(w, http.StatusBadRequest, APIError{
			Message: "Invalid action",
			Hint:    "Valid actions: start, stop, restart, reload",
		})
		return
	}

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{Message: "Failed to get deployment"})
		return
	}
	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{Message: "Deployment not found"})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{Message: "Failed to parse manifest"})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{Message: "Deployment does not have a VPS target"})
		return
	}

	cfg := sshConfigFromManifest(normalized)
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

	writeJSON(w, http.StatusOK, resp)
}

// handleTLSInfo retrieves detailed TLS certificate information.
// GET /api/v1/deployments/{id}/edge/tls
func (s *Server) handleTLSInfo(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{Message: "Failed to get deployment"})
		return
	}
	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{Message: "Deployment not found"})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{Message: "Failed to parse manifest"})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	domain := normalized.Edge.Domain

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Use openssl to get certificate info directly from the domain
	// This doesn't require SSH - it checks the public certificate
	cmd := fmt.Sprintf(
		"echo | timeout 10 openssl s_client -servername %s -connect %s:443 2>/dev/null | openssl x509 -noout -text 2>/dev/null",
		domain, domain,
	)

	// Try to run locally first (works if openssl is available)
	result, err := runLocalCommand(ctx, cmd)

	resp := TLSInfoResponse{
		Domain:    domain,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil || result == "" {
		resp.OK = false
		resp.Valid = false
		resp.Error = "Unable to retrieve TLS certificate. The domain may not have HTTPS configured yet."
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Parse the certificate output
	parseTLSCertOutput(result, &resp)
	resp.OK = true

	writeJSON(w, http.StatusOK, resp)
}

// handleTLSRenew forces a TLS certificate renewal via Caddy.
// POST /api/v1/deployments/{id}/edge/tls/renew
func (s *Server) handleTLSRenew(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{Message: "Failed to get deployment"})
		return
	}
	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{Message: "Deployment not found"})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{Message: "Failed to parse manifest"})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{Message: "Deployment does not have a VPS target"})
		return
	}

	domain := normalized.Edge.Domain
	cfg := sshConfigFromManifest(normalized)
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
		domain,
	)

	result, err := s.sshRunner.Run(ctx, cfg, cmd)

	resp := TLSRenewResponse{
		Domain:    domain,
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

	writeJSON(w, http.StatusOK, resp)
}

// parseTLSCertOutput parses openssl x509 -text output into TLSInfoResponse.
func parseTLSCertOutput(output string, resp *TLSInfoResponse) {
	lines := strings.Split(output, "\n")

	// Patterns to match
	issuerRegex := regexp.MustCompile(`Issuer:\s*(.+)`)
	subjectRegex := regexp.MustCompile(`Subject:\s*(.+)`)
	notBeforeRegex := regexp.MustCompile(`Not Before:\s*(.+)`)
	notAfterRegex := regexp.MustCompile(`Not After\s*:\s*(.+)`)
	sanRegex := regexp.MustCompile(`DNS:([^,\s]+)`)

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if match := issuerRegex.FindStringSubmatch(line); len(match) > 1 {
			// Extract CN if present
			issuer := match[1]
			if cnIdx := strings.Index(issuer, "CN = "); cnIdx >= 0 {
				resp.Issuer = strings.TrimSpace(issuer[cnIdx+5:])
				if commaIdx := strings.Index(resp.Issuer, ","); commaIdx >= 0 {
					resp.Issuer = resp.Issuer[:commaIdx]
				}
			} else {
				resp.Issuer = issuer
			}
		}

		if match := subjectRegex.FindStringSubmatch(line); len(match) > 1 {
			resp.Subject = match[1]
		}

		if match := notBeforeRegex.FindStringSubmatch(line); len(match) > 1 {
			resp.NotBefore = match[1]
		}

		if match := notAfterRegex.FindStringSubmatch(line); len(match) > 1 {
			resp.NotAfter = match[1]
			// Parse expiry date to calculate days remaining
			if t, err := time.Parse("Jan  2 15:04:05 2006 MST", match[1]); err == nil {
				resp.DaysRemaining = int(time.Until(t).Hours() / 24)
				resp.Valid = resp.DaysRemaining > 0
			} else if t, err := time.Parse("Jan 2 15:04:05 2006 GMT", match[1]); err == nil {
				resp.DaysRemaining = int(time.Until(t).Hours() / 24)
				resp.Valid = resp.DaysRemaining > 0
			}
		}

		// Serial number may be on next line
		if strings.Contains(line, "Serial Number:") {
			// Check if on same line or next
			if colonIdx := strings.Index(line, ":"); colonIdx >= 0 {
				afterColon := strings.TrimSpace(line[colonIdx+1:])
				if afterColon != "" && afterColon != "0" {
					resp.SerialNumber = afterColon
				} else if i+1 < len(lines) {
					resp.SerialNumber = strings.TrimSpace(lines[i+1])
				}
			}
		}

		// SANs
		if strings.Contains(line, "Subject Alternative Name") && i+1 < len(lines) {
			sanLine := lines[i+1]
			matches := sanRegex.FindAllStringSubmatch(sanLine, -1)
			for _, m := range matches {
				if len(m) > 1 {
					resp.SANs = append(resp.SANs, m[1])
				}
			}
		}
	}

	// If we found any valid data, mark as valid (unless already determined invalid by expiry)
	if resp.Issuer != "" || resp.Subject != "" {
		if resp.DaysRemaining == 0 && resp.NotAfter == "" {
			resp.Valid = true // Assume valid if we got cert data but couldn't parse expiry
		}
	}
}

// runLocalCommand runs a command locally and returns stdout.
func runLocalCommand(ctx context.Context, cmd string) (string, error) {
	execCmd := exec.CommandContext(ctx, "bash", "-c", cmd)
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
