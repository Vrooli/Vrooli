// Package readiness exposes owner-visible Bridge host readiness. It is a REST
// exception because it is an operational inspection document, not a workflow
// mutation; onboarding remains the typed RPC owner of durable admission events.
package readiness

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/database"
	"vrooli-bridge/internal/hostbroker"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/onboard"
	internalreadiness "vrooli-bridge/internal/readiness"

	"github.com/gorilla/mux"
)

const fixedAPIPort = 18767

type report struct {
	Status         string     `json:"status"`
	Endpoint       string     `json:"endpoint"`
	Port           int        `json:"port"`
	EndpointSource string     `json:"endpoint_source"`
	Mode           string     `json:"reachability_mode"`
	LocalAPI       bool       `json:"local_api"`
	Checks         checks     `json:"checks"`
	LastCandidate  *candidate `json:"last_candidate,omitempty"`
	Firewall       *firewall  `json:"firewall,omitempty"`
}
type checks struct {
	Listener bool `json:"listener"`
	API      bool `json:"api"`
	Auth     bool `json:"auth"`
	Storage  bool `json:"storage"`
	Keys     bool `json:"keys"`
}
type candidate struct {
	Host     string `json:"host"`
	Endpoint string `json:"endpoint"`
	Mode     string `json:"mode"`
	State    string `json:"state"`
	Category string `json:"category,omitempty"`
	SourceIP string `json:"source_ip,omitempty"`
}
type firewall struct {
	Available       bool   `json:"available"`
	Inspectable     bool   `json:"inspectable"`
	Active          bool   `json:"active"`
	RuleFound       bool   `json:"rule_found"`
	Privileged      bool   `json:"privileged"`
	BrokerAvailable bool   `json:"broker_available"`
	BrokerStatus    string `json:"broker_status,omitempty"`
}

// Module mounts a small owner-only readiness card data source. It reports the
// canonical default endpoint and the last durable candidate admission result;
// it never pretends local health proves remote reachability.
func Module(pinger database.Pinger, onboardSvc onboard.Service, endpoints *internalreadiness.Store, keyReady bool, dependencies ...any) module.Module {
	var inspector onboard.FirewallInspector
	var broker hostbroker.Client
	for _, dependency := range dependencies {
		switch candidate := dependency.(type) {
		case onboard.FirewallInspector:
			inspector = candidate
		case hostbroker.Client:
			broker = candidate
		}
	}
	h := func(w http.ResponseWriter, r *http.Request) {
		if _, err := auth.RequireOwner(r.Context()); err != nil {
			http.Error(w, "owner authentication required", http.StatusUnauthorized)
			return
		}
		selected, err := endpoints.Resolve(r.Context())
		if err != nil {
			http.Error(w, "read bridge endpoint configuration: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		// Reaching this owner-gated handler proves the live HTTP listener and
		// authenticator boundary. Database Ping is the storage/API dependency
		// check; the key readiness bit is set only after cpkeys.LoadOrCreate
		// completed during process boot.
		report := report{Endpoint: strings.TrimSpace(selected.URL), Port: fixedAPIPort, EndpointSource: selected.Source, Mode: selected.Mode, Checks: checks{Listener: true, Auth: true, Keys: keyReady}}
		if err := pinger.PingContext(r.Context()); err == nil {
			report.LocalAPI = true
			report.Checks.API = true
			report.Checks.Storage = true
		}
		ops, err := onboardSvc.ListOps(r.Context(), onboard.ListFilter{Limit: 1})
		if err == nil && len(ops) == 1 {
			op := ops[0]
			report.LastCandidate = &candidate{Host: op.Host, Endpoint: op.ControlPlaneURL, Mode: op.ReachabilityMode, State: op.State.String(), Category: string(op.FailureReason)}
			if _, events, getErr := onboardSvc.GetOp(r.Context(), op.ID); getErr == nil {
				for _, event := range events {
					if event.StepID != onboard.StepAdmission {
						continue
					}
					if ip := admissionSourceIP(event.Detail); ip != "" {
						report.LastCandidate.SourceIP = ip
					}
				}
			}
			if report.LastCandidate.Category == string(onboard.FailureControlPlaneUnreachable) {
				report.LastCandidate.SourceIP = remediationCandidateIP(report.LastCandidate.Host, report.LastCandidate.SourceIP, report.LastCandidate.Endpoint)
			}
			if report.LastCandidate.Category == string(onboard.FailureControlPlaneUnreachable) && report.LastCandidate.SourceIP != "" {
				if broker != nil {
					result, brokerErr := broker.Call(r.Context(), hostbroker.AdmissionRequest("bridge.ufw.inspect", brokerRequestID("inspect"), report.LastCandidate.SourceIP))
					if brokerErr == nil {
						report.Firewall = &firewall{Available: result.Evidence.Available, Inspectable: true, Active: result.Evidence.Active, RuleFound: result.Evidence.RuleFound, Privileged: true, BrokerAvailable: true, BrokerStatus: result.Status}
					} else {
						report.Firewall = &firewall{BrokerStatus: "unavailable"}
					}
				} else if inspector != nil {
					observed := inspector.InspectUFW(r.Context(), report.LastCandidate.SourceIP, fixedAPIPort)
					report.Firewall = &firewall{Available: observed.Available, Inspectable: observed.Inspectable, Active: observed.Active, RuleFound: observed.RuleFound, Privileged: observed.Privileged}
				}
			}
		}
		if report.LastCandidate != nil && report.LastCandidate.Category == string(onboard.FailureControlPlaneUnreachable) && report.LastCandidate.State == onboard.StateFailed.String() {
			report.Status = "candidate_blocked"
		} else if report.LocalAPI && report.Endpoint != "" {
			report.Status = "ready"
		} else {
			report.Status = "not_ready"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	}
	firewallAction := func(w http.ResponseWriter, r *http.Request) {
		if _, err := auth.RequireOwner(r.Context()); err != nil {
			http.Error(w, "owner authentication required", http.StatusUnauthorized)
			return
		}
		if broker == nil {
			http.Error(w, "privilege broker unavailable; re-run sudo vrooli setup", http.StatusServiceUnavailable)
			return
		}
		var body struct {
			Action      string `json:"action"`
			Confirm     bool   `json:"confirm"`
			CandidateIP string `json:"candidate_ip"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "decode firewall action: "+err.Error(), http.StatusBadRequest)
			return
		}
		action := strings.TrimSpace(body.Action)
		if action != "preview" && action != "inspect" && action != "verify" && action != "allow" && action != "revoke" {
			http.Error(w, "unsupported firewall action", http.StatusBadRequest)
			return
		}
		if (action == "allow" || action == "revoke") && !body.Confirm {
			http.Error(w, "confirmation is required for firewall mutation", http.StatusBadRequest)
			return
		}
		ops, err := onboardSvc.ListOps(r.Context(), onboard.ListFilter{Limit: 1})
		if err != nil || len(ops) != 1 || ops[0].FailureReason != onboard.FailureControlPlaneUnreachable || ops[0].State != onboard.StateFailed {
			http.Error(w, "no failed candidate admission is available for firewall remediation", http.StatusConflict)
			return
		}
		_, events, err := onboardSvc.GetOp(r.Context(), ops[0].ID)
		if err != nil {
			http.Error(w, "read candidate admission evidence: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		observed := ""
		for _, event := range events {
			if event.StepID == onboard.StepAdmission {
				observed = admissionSourceIP(event.Detail)
			}
		}
		ip := remediationCandidateIP(ops[0].Host, observed, ops[0].ControlPlaneURL)
		if net.ParseIP(ip) == nil {
			http.Error(w, "failed candidate admission did not record a usable source IP", http.StatusConflict)
			return
		}
		if requested := strings.TrimSpace(body.CandidateIP); requested != "" && requested != ip {
			http.Error(w, "candidate_ip does not match the latest failed candidate admission", http.StatusBadRequest)
			return
		}
		if action == "preview" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "preview", "action": "bridge.ufw.allow", "subject": map[string]any{"scenario": "vrooli-bridge", "candidate_ip": ip, "port": fixedAPIPort}})
			return
		}
		result, err := broker.Call(r.Context(), hostbroker.AdmissionRequest("bridge.ufw."+action, brokerRequestID(action), ip))
		if err != nil {
			http.Error(w, "privilege broker unavailable; re-run sudo vrooli setup: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
	configure := func(w http.ResponseWriter, r *http.Request) {
		if _, err := auth.RequireOwner(r.Context()); err != nil {
			http.Error(w, "owner authentication required", http.StatusUnauthorized)
			return
		}
		var body struct {
			Endpoint string `json:"endpoint"`
			Mode     string `json:"reachability_mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "decode endpoint configuration: "+err.Error(), http.StatusBadRequest)
			return
		}
		selected, err := endpoints.Save(r.Context(), body.Endpoint, body.Mode)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid endpoint configuration: %v", err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report{Status: "configured", Endpoint: selected.URL, EndpointSource: selected.Source, Mode: selected.Mode, Port: fixedAPIPort})
	}
	return module.Module{Name: "readiness", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/readiness", h).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/readiness/endpoint", configure).Methods(http.MethodPut)
		r.HandleFunc("/api/v1/readiness/firewall", firewallAction).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}

func brokerRequestID(action string) string {
	return fmt.Sprintf("bridge-%s-%d", action, time.Now().UnixNano())
}

func admissionSourceIP(detail string) string {
	const prefix = "candidate source "
	_, rest, found := strings.Cut(detail, prefix)
	if !found {
		return ""
	}
	value, _, _ := strings.Cut(rest, ";")
	return strings.TrimSpace(value)
}

// remediationCandidateIP rejects legacy SSH_CONNECTION evidence that equals
// the Bridge endpoint itself, then resolves a literal or candidate hostname as
// a compatibility fallback. New probes report the route-selected source IP.
func remediationCandidateIP(host, observed, endpoint string) string {
	if ip := net.ParseIP(strings.TrimSpace(observed)); ip != nil && !sameEndpointIP(ip, endpoint) {
		return ip.String()
	}
	if ip := net.ParseIP(strings.TrimSpace(host)); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
		return ip.String()
	}
	ips, err := net.LookupIP(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if err != nil {
		return ""
	}
	for _, ip := range ips {
		if ip.To4() != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
			return ip.String()
		}
	}
	return ""
}

func sameEndpointIP(ip net.IP, endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	endpointIP := net.ParseIP(u.Hostname())
	return endpointIP != nil && endpointIP.Equal(ip)
}

func Schema() string { return "" }
