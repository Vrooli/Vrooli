package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	accesspb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/access"
)

// The CLI talks to a local test server, so the package runs detached from any
// agent session that runs it (which would redirect it to the live scenario).
func TestMain(m *testing.M) { os.Exit(cliapptest.RunWithoutAgentSession(m)) }

type manifestCommand struct {
	Name  string `json:"name"`
	Flags []struct {
		Name      string   `json:"name"`
		Required  bool     `json:"required"`
		Values    []string `json:"values"`
		Bool      bool     `json:"bool"`
		LocalOnly bool     `json:"local_only"`
	} `json:"flags"`
	Binding struct {
		Service string `json:"service"`
		Method  string `json:"method"`
	} `json:"binding"`
	Governance struct {
		Effect      string `json:"effect"`
		RunEligible bool   `json:"run_eligible"`
	} `json:"governance"`
}

type manifestGroup struct {
	Name     string            `json:"name"`
	Commands []manifestCommand `json:"commands"`
}

// writeCommands is every manifest command that changes Money Ledger.
func writeCommands(t *testing.T) map[string][]manifestCommand {
	t.Helper()
	var manifest struct {
		Groups []manifestGroup `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(cliapptest.ReadManifest(t), &manifest))
	writes := map[string][]manifestCommand{}
	for _, group := range manifest.Groups {
		for _, command := range group.Commands {
			if command.Governance.Effect != "read" {
				writes[group.Name] = append(writes[group.Name], command)
			}
		}
	}
	require.NotEmpty(t, writes, "the manifest declares no write commands; the reader is out of date")
	return writes
}

// Every write command offers --local-owner and is not offered to agent
// actions: Money Ledger refuses an agent's write whatever it presents.
func TestEveryWriteCommandIsTheOwners(t *testing.T) {
	for _, commands := range writeCommands(t) {
		for _, command := range commands {
			require.False(t, command.Governance.RunEligible, "%s writes the owner's records and must not be run-eligible", command.Name)
			found := false
			for _, flag := range command.Flags {
				if flag.Name == localOwnerFlag {
					found = flag.Bool && flag.LocalOnly
				}
			}
			require.True(t, found, "%s must declare --%s as a local-only bool", command.Name, localOwnerFlag)
		}
	}
}

// recordingServer answers every RPC the way Money Ledger answers one with no
// identity, or with a fixed refusal once one is presented, and remembers the
// credential each procedure received. No command reaches a report.
type recordingServer struct {
	mu       sync.Mutex
	received map[string]string
}

func (s *recordingServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("Authorization")
	s.mu.Lock()
	s.received[r.URL.Path] = authorization
	s.mu.Unlock()
	refusal := connect.NewError(connect.CodeFailedPrecondition, errors.New("fixture stops here"))
	if authorization == "" {
		refusal = connect.NewError(connect.CodeUnauthenticated, errors.New("no identity was presented"))
	}
	_ = connect.NewErrorWriter().Write(w, r, refusal)
}

func (s *recordingServer) credential(path string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.received[path]
	return value, ok
}

// Every write command, run with --local-owner, presents the owner session it
// exchanged for; run without it, presents nothing and tells the operator how
// to write. One path serves them all, so none can drift.
func TestEveryWriteCommandPresentsTheOwnerSessionOnlyWhenAsked(t *testing.T) {
	withOwnerSeams(t, func(context.Context) error { return nil }, func(context.Context) (string, error) { return "owner-session", nil })
	server := &recordingServer{received: map[string]string{}}
	core := cliapptest.NewTestApp(t, server)
	manifest := cliapptest.ReadManifest(t)
	ledgerGroup, err := Register(core, manifest)
	require.NoError(t, err)
	ingestGroup, err := RegisterIngest(core, manifest)
	require.NoError(t, err)
	groups := []cliapp.SubcommandGroup{ledgerGroup, ingestGroup}

	for group, commands := range writeCommands(t) {
		for _, command := range commands {
			procedure := "/vrooli.money_ledger.v1." + packageOf(command.Binding.Service) + "." + command.Binding.Service + "/" + command.Binding.Method
			argv := []string{}
			for _, flag := range command.Flags {
				if flag.Required {
					argv = append(argv, "--"+flag.Name, validValue(flag.Values))
				}
			}
			cmd := cliapptest.FindCommand(t, groups, group, command.Name)

			_, err := cliapptest.RunCommand(t, cmd, core, argv...)
			require.Error(t, err, command.Name)
			require.ErrorContains(t, err, "--local-owner", "%s must tell the operator how to write", command.Name)
			credential, reached := server.credential(procedure)
			require.True(t, reached, "%s did not call %s", command.Name, procedure)
			require.Empty(t, credential, "%s presented a credential without --local-owner", command.Name)

			_, err = cliapptest.RunCommand(t, cmd, core, append(argv, "--"+localOwnerFlag)...)
			require.ErrorContains(t, err, "fixture stops here", command.Name)
			credential, _ = server.credential(procedure)
			require.Equal(t, "Bearer owner-session", credential, "%s with --local-owner must present the owner session", command.Name)
		}
	}
}

// validValue is a value every required flag accepts: the first member of a
// declared vocabulary, otherwise "1", which reads as an id, a name or a number.
func validValue(values []string) string {
	if len(values) > 0 {
		return values[0]
	}
	return "1"
}

func packageOf(service string) string {
	if strings.HasPrefix(service, "Ingest") {
		return "ingest"
	}
	return "ledger"
}

// The write-access report says who the caller writes as, or why it may not,
// in words the operator can act on.
func TestWriteAccessReport(t *testing.T) {
	allowed := writeAccessReport(nil, &accesspb.GetWriteAccessResponse{CanWrite: true, Actor: "operator", Reason: "writes are recorded as operator"})
	require.Contains(t, allowed.Summary[0], "records its writes as operator")
	refused := writeAccessReport(nil, &accesspb.GetWriteAccessResponse{Refusal: accesspb.WriteRefusal_WRITE_REFUSAL_NOT_OWNER, Reason: "this person is not named as the owner"})
	require.Contains(t, refused.Summary[0], "may not write")
	require.Equal(t, []string{"refusal=not_owner this person is not named as the owner"}, refused.Results)
}
