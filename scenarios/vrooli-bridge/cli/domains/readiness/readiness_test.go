package readiness

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	clitest "vrooli-bridge/cli/internal/testutil"
)

func TestStatusRendersCanonicalEndpointAndCandidateEvidence(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/readiness", r.URL.Path)
		_, _ = io.WriteString(w, `{"status":"ready","endpoint":"http://192.168.1.173:18767","port":18767,"endpoint_source":"configured","reachability_mode":"lan","local_api":true,"last_candidate":{"host":"mac","state":"FAILED","category":"control_plane_unreachable","source_ip":"192.168.1.176"}}`)
	}))
	var out bytes.Buffer
	err := status(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, Stdout: &out}))
	require.NoError(t, err)
	require.Contains(t, out.String(), "http://192.168.1.173:18767")
	require.Contains(t, out.String(), "candidate source: 192.168.1.176")
}

func TestConfigureSendsOnlyEndpointAndMode(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, "/api/v1/readiness/endpoint", r.URL.Path)
		_, _ = io.WriteString(w, `{}`)
	}))
	var out bytes.Buffer
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "endpoint"}, {Name: "reachability-mode"}}}
	err := configure(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, Stdout: &out, Schema: schema, Flags: map[string]string{"endpoint": "https://bridge.example.test", "reachability-mode": "manual"}}))
	require.NoError(t, err)
	require.Contains(t, out.String(), "Bridge advertised endpoint configured")
}

func TestFirewallAllowSendsExplicitConfirmation(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/readiness/firewall", r.URL.Path)
		_, _ = io.WriteString(w, `{"status":"changed","changed":true}`)
	}))
	var out bytes.Buffer
	schema := firewallArgs(true)
	err := firewallAction("allow", true)(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, Stdout: &out, Schema: schema, Flags: map[string]string{"candidate-ip": "192.168.1.176", "confirm": "true"}}))
	require.NoError(t, err)
	require.Contains(t, out.String(), "Firewall allow: changed")
}
