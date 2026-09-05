// gen-endpoints derives the Agent Manager endpoint inventory from its mux and
// generated Connect registrations. The inventory is deliberately generated so
// adding a route cannot leave .vrooli/endpoints.json stale.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type manifest struct {
	Schema    string     `json:"$schema"`
	Version   string     `json:"version"`
	Endpoints []endpoint `json:"endpoints"`
}
type endpoint struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Method      string `json:"method"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Category    string `json:"category"`
}
type route struct{ Path, Method string }

func main() {
	output := flag.String("output", "../.vrooli/endpoints.json", "endpoint manifest output")
	flag.Parse()
	if err := generate(*output); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}

func generate(output string) error {
	routes, err := muxRoutes("internal/handlers/handlers.go")
	if err != nil {
		return err
	}
	routes = append(routes, muxRoutesMust("internal/wiring/routes.go")...)
	routes = append(routes, connectRoutes(domainpb.File_agent_manager_v1_domain_episode_proto)...)
	routes = append(routes, connectRoutes(measurepb.File_agent_manager_v1_measures_measures_proto)...)
	seen := map[route]bool{}
	unique := make([]route, 0, len(routes))
	for _, r := range routes {
		if !seen[r] {
			seen[r] = true
			unique = append(unique, r)
		}
	}
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].Path == unique[j].Path {
			return unique[i].Method < unique[j].Method
		}
		return unique[i].Path < unique[j].Path
	})
	entries := make([]endpoint, 0, len(unique))
	for _, r := range unique {
		entries = append(entries, endpoint{ID: id(r), Path: r.Path, Method: r.Method, Summary: r.Method + " " + r.Path, Description: "Generated from Agent Manager route and proto registrations; run make endpoints after route changes.", Category: category(r.Path)})
	}
	body, err := json.MarshalIndent(manifest{Schema: "../../../../scripts/scenarios/schemas/endpoints.schema.json", Version: "1.0.0", Endpoints: entries}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o750); err != nil {
		return err
	}
	return os.WriteFile(output, append(body, '\n'), 0o600)
}

func muxRoutes(path string) ([]route, error) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}
	var routes []route
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || method(call.Fun) != "Methods" {
			return true
		}
		handle, ok := receiver(call.Fun)
		if !ok || method(handle.Fun) != "HandleFunc" || len(handle.Args) == 0 {
			return true
		}
		path, ok := literal(handle.Args[0])
		if !ok {
			return true
		}
		for _, arg := range call.Args {
			if verb, ok := literal(arg); ok {
				routes = append(routes, route{path, verb})
			}
		}
		return true
	})
	return routes, nil
}

func muxRoutesMust(path string) []route {
	routes, err := muxRoutes(path)
	if err != nil {
		panic(err)
	}
	return routes
}

func connectRoutes(file protoreflect.FileDescriptor) []route {
	var routes []route
	for i := 0; i < file.Services().Len(); i++ {
		service := file.Services().Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			routes = append(routes, route{"/" + string(service.FullName()) + "/" + string(service.Methods().Get(j).Name()), "POST"})
		}
	}
	return routes
}

func method(expr ast.Expr) string {
	if s, ok := expr.(*ast.SelectorExpr); ok {
		return s.Sel.Name
	}
	return ""
}

func receiver(expr ast.Expr) (*ast.CallExpr, bool) {
	s, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	c, ok := s.X.(*ast.CallExpr)
	return c, ok
}

func literal(expr ast.Expr) (string, bool) {
	b, ok := expr.(*ast.BasicLit)
	if !ok || b.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(b.Value)
	return s, err == nil
}

func id(r route) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(r.Method))
	for _, ch := range r.Path {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			b.WriteRune(unicode.ToLower(ch))
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func category(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 2 && parts[0] == "api" && parts[1] == "v1" {
		return parts[2]
	}
	return "system"
}
