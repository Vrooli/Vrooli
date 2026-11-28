package facts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndpointAnatomyConnectProcedureAndMountChainPositive(t *testing.T) {
	root := writeConnectAnatomyFixture(t, `const CodeFactsServiceSearchProcedure = "/vrooli.code_facts.v1.facts.CodeFactsService/Search"`, `connectx.RegisterServices(r, connectx.ServiceMount{Path: servicePath, Handler: serviceHandler})`)
	response, err := scanEndpointAnatomy(root, "code-facts", "CodeFactsService Search code facts Connect", 10)
	if err != nil {
		t.Fatal(err)
	}
	var found *EndpointAnatomy
	for i := range response.Results {
		if response.Results[i].ID == "code-facts:endpoint:facts_search" {
			found = &response.Results[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("Connect procedure not returned: %#v", response)
	}
	if found.Transport != "connect" || found.ConnectService != "CodeFactsService" || found.ConnectMethod != "Search" || found.State != "CONNECT_STATIC_MOUNT_REGISTERED" || found.RoutePath != "/vrooli.code_facts.v1.facts.CodeFactsService/Search" {
		t.Fatalf("unexpected Connect result: %#v", found)
	}
	if len(found.ConnectChain) != 6 {
		t.Fatalf("expected declaration, generated procedure/method registration, constructor, Mount closure, ServiceMount and RegisterServices provenance: %#v", found.ConnectChain)
	}
	for _, edge := range found.ConnectChain {
		if edge.Path == "" || edge.Line <= 0 || edge.SHA256 == "" || edge.Basis == "" {
			t.Fatalf("incomplete chain edge provenance: %#v", edge)
		}
	}
}

func TestEndpointAnatomyConnectUnsupportedAndAmbiguousChainsStayPartial(t *testing.T) {
	cases := []struct{ name, generatedPath, registration, want string }{
		{"missing generated constant", "", `connectx.RegisterServices(r, connectx.ServiceMount{Path: servicePath, Handler: serviceHandler})`, "CONNECT_ROUTE_UNKNOWN"},
		{"malformed ServiceMount", `const CodeFactsServiceSearchProcedure = "/vrooli.code_facts.v1.facts.CodeFactsService/Search"`, `connectx.RegisterServices(r, connectx.ServiceMount{Path: servicePath})`, "CONNECT_MOUNT_PARTIAL"},
		{"unsupported registration call", `const CodeFactsServiceSearchProcedure = "/vrooli.code_facts.v1.facts.CodeFactsService/Search"`, `registerService(r, servicePath, serviceHandler)`, "CONNECT_MOUNT_PARTIAL"},
		{"ambiguous duplicate registration", `const CodeFactsServiceSearchProcedure = "/vrooli.code_facts.v1.facts.CodeFactsService/Search"`, "connectx.RegisterServices(r, connectx.ServiceMount{Path: servicePath, Handler: serviceHandler})\nconnectx.RegisterServices(r, connectx.ServiceMount{Path: servicePath, Handler: serviceHandler})", "CONNECT_MOUNT_UNKNOWN"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			root := writeConnectAnatomyFixture(t, tt.generatedPath, tt.registration)
			response, err := scanEndpointAnatomy(root, "code-facts", "CodeFactsService Search code facts Connect", 10)
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range response.Results {
				if row.ID == "code-facts:endpoint:facts_search" {
					if row.State != tt.want {
						t.Fatalf("got state %s, want %s (%+v)", row.State, tt.want, row)
					}
					if row.State != "CONNECT_STATIC_MOUNT_REGISTERED" && len(response.Degradations) == 0 {
						t.Fatalf("partial result must degrade explicitly: %+v", response)
					}
					return
				}
			}
			t.Fatalf("expected explicit partial/unknown candidate: %+v", response)
		})
	}
}

func writeConnectAnatomyFixture(t *testing.T, generatedDeclaration, registration string) string {
	t.Helper()
	repo := t.TempDir()
	root := filepath.Join(repo, "scenarios", "code-facts")
	moduleDir := filepath.Join(root, "api", "handlers", "facts")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	module := `package facts
import (
  factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
  "github.com/vrooli/api-core/connectx"
  "github.com/vrooli/api-core/module"
)
var Endpoints = []module.EndpointDescriptor{{ID:"facts_search", Path:factsconnect.CodeFactsServiceSearchProcedure, Method:"POST", Summary:"Search code facts"}}
func Module() module.Module { return module.Module{Mount:func(r any) {
 servicePath, serviceHandler := factsconnect.NewCodeFactsServiceHandler(nil)
 REGISTRATION
}} }
`
	module = strings.Replace(module, "REGISTRATION", registration, 1)
	if err := os.WriteFile(filepath.Join(moduleDir, "module.go"), []byte(module), 0o600); err != nil {
		t.Fatal(err)
	}
	generatedDir := filepath.Join(repo, "packages", "proto", "gen", "go", "code-facts", "v1", "facts", "facts_v1connect")
	if err := os.MkdirAll(generatedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	generated := "package factsconnect\n" + generatedDeclaration + "\nfunc NewCodeFactsServiceHandler() { connect.NewUnaryHandler(CodeFactsServiceSearchProcedure, nil) }\n"
	if err := os.WriteFile(filepath.Join(generatedDir, "facts.connect.go"), []byte(generated), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}
