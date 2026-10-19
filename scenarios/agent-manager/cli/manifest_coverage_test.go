package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func TestAgentManagerManifestCoversAgentManagerService(t *testing.T) {
	manifest, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatalf("read CLI manifest: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, manifest, apipb.File_agent_manager_v1_api_service_proto, "AgentManagerService")

	parsed, err := cliapp.ParseManifest(manifest)
	if err != nil {
		t.Fatalf("parse CLI manifest: %v", err)
	}
	bound := 0
	var visit func([]cliapp.ManifestGroup)
	visit = func(groups []cliapp.ManifestGroup) {
		for _, group := range groups {
			for _, command := range group.Commands {
				if group.Name == "run" && command.Name == "event-page" &&
					canonicalAgentManagerService(command.Binding.Service) && command.Binding.Method == "GetRunEvents" {
					bound++
				}
			}
			visit(group.Groups)
		}
	}
	visit(parsed.Groups)
	if bound != 1 {
		t.Fatalf("run event-page binds AgentManagerService.GetRunEvents %d times, want exactly once", bound)
	}
	for _, omission := range parsed.Omitted {
		if canonicalAgentManagerService(omission.Service) && omission.Method == "GetRunEvents" {
			t.Fatal("AgentManagerService.GetRunEvents must be bound, not omitted")
		}
	}
}

// TestManifestConnectCommandsHaveExplicitHandlerReceiverMethods keeps the
// compatibility manifest honest: generated protobuf methods are not available
// through Connect unless the Agent Manager handler package implements them.
func TestManifestConnectCommandsHaveExplicitHandlerReceiverMethods(t *testing.T) {
	manifestBytes, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatalf("read CLI manifest: %v", err)
	}
	manifest, err := cliapp.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("parse CLI manifest: %v", err)
	}

	handlerDir := filepath.Join("..", "api", "internal", "handlers")
	methods := map[string]bool{}
	entries, err := os.ReadDir(handlerDir)
	if err != nil {
		t.Fatalf("read handler sources: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(handlerDir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse handler source %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			if receiverTypeName(fn.Recv.List[0].Type) == "AgentManagerConnectHandler" {
				methods[fn.Name.Name] = true
			}
		}
	}

	checked := 0
	var visit func([]cliapp.ManifestGroup)
	visit = func(groups []cliapp.ManifestGroup) {
		for _, group := range groups {
			for _, command := range group.Commands {
				if command.Binding.Kind != "connect-rpc" || !canonicalAgentManagerService(command.Binding.Service) {
					continue
				}
				checked++
				if !methods[command.Binding.Method] {
					t.Errorf("manifest command binds AgentManagerService.%s, but the registered handler has no explicit receiver method", command.Binding.Method)
				}
			}
			visit(group.Groups)
		}
	}
	visit(manifest.Groups)
	if checked == 0 {
		t.Fatal("expected at least one explicit AgentManagerService Connect binding")
	}
}

func receiverTypeName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		if ident, ok := value.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func canonicalAgentManagerService(service string) bool {
	if dot := strings.LastIndexByte(service, '.'); dot >= 0 {
		service = service[dot+1:]
	}
	return service == "AgentManagerService"
}
