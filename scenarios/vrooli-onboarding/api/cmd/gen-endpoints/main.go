// Command gen-endpoints renders the onboarding REST contract from the
// router registrations in api/main.go. The checked-in endpoints.json remains
// the place for human-facing descriptions and requirement links, but its
// route/method set is derived from executable registrations and cannot drift
// silently.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
)

type route struct {
	Path   string
	Method string
}

type manifest struct {
	Schema      string            `json:"$schema"`
	Version     string            `json:"version"`
	Service     string            `json:"service"`
	Categories  []string          `json:"categories"`
	CLICommands []json.RawMessage `json:"cli_commands,omitempty"`
	Endpoints   []json.RawMessage `json:"endpoints"`
}

func main() {
	routerPath := flag.String("router", "main.go", "Go source containing router registrations")
	metadataPath := flag.String("metadata", "../.vrooli/endpoints.json", "existing endpoint metadata used for descriptions and requirements")
	outputPath := flag.String("output", "../.vrooli/endpoints.json", "generated endpoint declaration")
	flag.Parse()

	if err := generate(*routerPath, *metadataPath, *outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}

func generate(routerPath, metadataPath, outputPath string) error {
	routes, err := routesFromSource(routerPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("read metadata %s: %w", metadataPath, err)
	}
	var source manifest
	if err := json.Unmarshal(data, &source); err != nil {
		return fmt.Errorf("decode metadata %s: %w", metadataPath, err)
	}

	metadata := make(map[string]json.RawMessage, len(source.Endpoints))
	for _, raw := range source.Endpoints {
		var endpoint struct {
			Path   string `json:"path"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(raw, &endpoint); err != nil {
			return fmt.Errorf("decode endpoint metadata: %w", err)
		}
		key := endpointKey(endpoint.Method, endpoint.Path)
		if endpoint.Path == "" || endpoint.Method == "" {
			return fmt.Errorf("endpoint metadata has empty path or method")
		}
		if _, exists := metadata[key]; exists {
			return fmt.Errorf("duplicate endpoint metadata %s", key)
		}
		metadata[key] = raw
	}

	generated := make([]json.RawMessage, 0, len(routes))
	seen := make(map[string]bool, len(routes))
	for _, route := range routes {
		key := endpointKey(route.Method, route.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		raw, ok := metadata[key]
		if !ok {
			return fmt.Errorf("router route %s has no endpoint metadata", key)
		}
		generated = append(generated, raw)
	}
	var orphaned []string
	for key := range metadata {
		if !seen[key] {
			orphaned = append(orphaned, key)
		}
	}
	if len(orphaned) > 0 {
		sort.Strings(orphaned)
		return fmt.Errorf("endpoint metadata has no router registration: %v", orphaned)
	}
	source.Endpoints = generated

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(source); err != nil {
		return fmt.Errorf("encode generated endpoints: %w", err)
	}
	if err := os.WriteFile(outputPath, output.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}
	return nil
}

func routesFromSource(path string) ([]route, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse router source %s: %w", path, err)
	}
	routes := make([]route, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		methods, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || methods.Sel.Name != "Methods" {
			return true
		}
		handle, ok := methods.X.(*ast.CallExpr)
		if !ok || len(handle.Args) == 0 {
			return true
		}
		handleName, ok := handle.Fun.(*ast.SelectorExpr)
		if !ok || handleName.Sel.Name != "HandleFunc" {
			return true
		}
		pathLiteral, ok := handle.Args[0].(*ast.BasicLit)
		if !ok || pathLiteral.Kind != token.STRING {
			return true
		}
		pathValue, err := strconv.Unquote(pathLiteral.Value)
		if err != nil {
			return true
		}
		for _, methodArg := range call.Args {
			methodLiteral, ok := methodArg.(*ast.BasicLit)
			if !ok || methodLiteral.Kind != token.STRING {
				continue
			}
			method, err := strconv.Unquote(methodLiteral.Value)
			if err == nil {
				routes = append(routes, route{Path: pathValue, Method: method})
			}
		}
		return true
	})
	if len(routes) == 0 {
		return nil, fmt.Errorf("no HandleFunc(...).Methods(...) registrations found in %s", path)
	}
	return routes, nil
}

func endpointKey(method, path string) string { return method + " " + path }
