package preflight

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/reach"
)

// ReachFactory selects the reach for a target that may have no deployment
// record yet (the wizard runs preflight and fixes before the deployment
// exists). Production hands back the router, whose SSH adapter resolves the
// connection from the credential binding when one exists and from the
// operator's ambient identity otherwise.
type ReachFactory func(target identity.TargetRef) reach.Reach

// preflightTarget is the reach target for a raw host/user/workdir.
func preflightTarget(host string, port int, user, workdir string) identity.TargetRef {
	return identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: host, Port: port, User: user, Workdir: workdir}}
}

// HandleOpenFirewallPorts allows inbound edge ports through the privilege
// broker's edge.ufw.allow action (80 and 443 only; the broker refuses
// anything else and never enables or reloads the firewall).
func HandleOpenFirewallPorts(reachFor ReachFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req FirewallFixRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}
		ports := req.Ports
		if len(ports) == 0 {
			ports = []int{80, 443}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		target := preflightTarget(req.Host, req.Port, req.User, domain.DefaultVPSWorkdir)
		rr := reachFor(target)
		var failures []string
		for _, port := range ports {
			if port <= 0 {
				continue
			}
			res, err := hostRepair(ctx, rr, target, "edge.ufw.allow", map[string]any{"edge": map[string]any{"port": port}})
			if failure := verbFailure(res, err); failure != "" {
				failures = append(failures, fmt.Sprintf("port %d: %s", port, failure))
			}
		}
		resp := FirewallFixResponse{OK: len(failures) == 0, Ports: ports, Timestamp: time.Now().UTC().Format(time.RFC3339)}
		if resp.OK {
			resp.Message = "Firewall rules updated through the target owner"
		} else {
			resp.Message = "Firewall update refused or failed: " + strings.Join(failures, "; ")
		}
		httputil.WriteJSON(w, http.StatusOK, resp)
	}
}

// HandleStopScenarioProcesses stops one scenario through the lifecycle
// owner (privilegebroker process.stop.scoped). Stopping "all vrooli
// processes" is the top-level `vrooli stop` verb; there is no pattern kill.
func HandleStopScenarioProcesses(reachFor ReachFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req StopScenarioProcessesRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}
		if req.Workdir == "" {
			req.Workdir = domain.DefaultVPSWorkdir
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		target := preflightTarget(req.Host, req.Port, req.User, req.Workdir)
		rr := reachFor(target)
		if req.ScenarioID != "" {
			res, err := hostRepair(ctx, rr, target, "process.stop.scoped", map[string]any{"process": map[string]any{"scenario": req.ScenarioID, "workdir": req.Workdir}})
			if failure := verbFailure(res, err); failure != "" {
				writeStopScenarioResponse(w, false, "stop_scenario", failure, strings.TrimSpace(res.Stdout))
				return
			}
			writeStopScenarioResponse(w, true, "stop_scenario", "Stopped scenario "+req.ScenarioID+" through the lifecycle owner", strings.TrimSpace(res.Stdout))
			return
		}
		res, err := rr.Exec(ctx, target, reach.Command{Verb: "stop", Args: []string{"--json"}, RequiredScope: "vrooli:write", Effectful: true, Timeout: 2 * time.Minute})
		if err != nil {
			writeStopScenarioResponse(w, false, "stop_all", err.Error(), "")
			return
		}
		if res.ExitCode != 0 {
			writeStopScenarioResponse(w, false, "stop_all", fmt.Sprintf("vrooli stop exited %d", res.ExitCode), strings.TrimSpace(res.Stderr))
			return
		}
		writeStopScenarioResponse(w, true, "stop_all", "Stopped every vrooli-managed process through the lifecycle owner", strings.TrimSpace(res.Stdout))
	}
}
