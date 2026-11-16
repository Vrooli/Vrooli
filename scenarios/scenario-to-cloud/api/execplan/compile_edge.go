package execplan

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/edge"
)

// deriveEdgeRoute resolves the one public route from the closure's declared
// listeners through the edge policy (a scenario's public listener need not be
// named "ui"). A deployment without declared listeners keeps the launch
// contract: the manifest's ui port behind the apex host.
func deriveEdgeRoute(in CompileInputs, uiPort int) (host string, port int, listenerID string) {
	spec := deriveEdgeSpec(in, uiPort)
	if len(spec.Routes) > 0 {
		r := spec.Routes[0]
		return r.Host, r.UpstreamPort, r.ListenerID
	}
	return strings.TrimSpace(in.Manifest.Edge.Domain), uiPort, in.Deployment.ScenarioID + "/ui"
}

func deriveEdgeSpec(in CompileInputs, uiPort int) domain.EdgeSpec {
	domainName := strings.TrimSpace(in.Manifest.Edge.Domain)
	listenerID := in.Deployment.ScenarioID + "/ui"
	if in.Closure != nil && len(in.Closure.Listeners) > 0 && domainName != "" {
		spec, err := edge.Derive(edge.PolicyInputs{
			DeploymentID:    in.Deployment.ID,
			ScenarioID:      in.Deployment.ScenarioID,
			Environment:     in.Deployment.Environment,
			Domain:          domainName,
			Listeners:       in.Closure.Listeners,
			Ports:           in.Manifest.Ports,
			ACMEEmail:       in.Manifest.Edge.Caddy.Email,
			ACMEEnvironment: in.Manifest.Edge.ACMEEnvironment,
		})
		if err == nil {
			return spec
		}
	}
	return domain.EdgeSpec{SchemaVersion: "1", DeploymentID: in.Deployment.ID, ScenarioID: in.Deployment.ScenarioID, Domain: domainName, Routes: []domain.EdgeRoute{{Host: domainName, UpstreamPort: uiPort, ListenerID: listenerID}}, FirewallAllow: []int{80, 443}, ACMEEnvironment: in.Manifest.Edge.ACMEEnvironment, ACMEEmail: strings.TrimSpace(in.Manifest.Edge.Caddy.Email)}
}

func encodeEdgeSpec(spec domain.EdgeSpec) string {
	encoded, err := json.Marshal(spec)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(encoded)
}
