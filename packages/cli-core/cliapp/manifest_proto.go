package cliapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// cliManifestSchemaFile is the canonical filename in .vrooli/schemas/.
// Matches the schema's $id "cli-manifest/v1".
const cliManifestSchemaFile = "cli-manifest.schema.json"

// RequireProtoServiceCoverage is a test-only helper that asserts:
//
//  1. `raw` is a syntactically and semantically valid cli/manifest.json
//     under .vrooli/schemas/cli-manifest.schema.json.
//  2. Every method on the named service of the provided file descriptor
//     either appears as a `binding` on some command (any group) OR is
//     listed in the manifest's `omitted` array with a reason. Bindings
//     and omissions referring to methods not present in the descriptor
//     fail the test (catches typos).
//
// Pass once per bound proto service in a scenario; multi-service scenarios
// invoke this helper once per service.
func RequireProtoServiceCoverage(t *testing.T, raw []byte, fd protoreflect.FileDescriptor, serviceName string) {
	t.Helper()
	if err := validateManifestAgainstSchema(t, raw); err != nil {
		t.Fatalf("cli manifest fails schema validation: %v", err)
	}
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatalf("cli manifest fails structural parse: %v", err)
	}

	service := lookupService(fd, serviceName)
	if service == nil {
		t.Fatalf("proto file descriptor %s does not declare service %q", fd.Path(), serviceName)
	}

	bound := make(map[string]string) // method -> "<group>/<command>"
	for _, g := range m.Groups {
		for _, c := range g.Commands {
			if c.Binding.Service != serviceName {
				continue
			}
			key := c.Binding.Method
			if existing, dup := bound[key]; dup {
				t.Fatalf("manifest binds %s.%s twice: %s and %s/%s", serviceName, key, existing, g.Name, c.Name)
			}
			bound[key] = g.Name + "/" + c.Name
		}
	}

	omitted := make(map[string]string)
	for _, o := range m.Omitted {
		if o.Service != serviceName {
			continue
		}
		if _, dup := omitted[o.Method]; dup {
			t.Fatalf("manifest omits %s.%s twice", serviceName, o.Method)
		}
		omitted[o.Method] = o.Reason
	}

	methods := service.Methods()
	known := make(map[string]struct{}, methods.Len())
	var uncovered []string
	for i := 0; i < methods.Len(); i++ {
		name := string(methods.Get(i).Name())
		known[name] = struct{}{}
		_, isBound := bound[name]
		_, isOmitted := omitted[name]
		switch {
		case isBound && isOmitted:
			t.Fatalf("manifest both binds and omits %s.%s; pick one", serviceName, name)
		case isBound || isOmitted:
			// covered
		default:
			uncovered = append(uncovered, name)
		}
	}
	if len(uncovered) > 0 {
		sort.Strings(uncovered)
		t.Fatalf("service %s has methods not covered by manifest (add a binding or an omitted entry): %s", serviceName, strings.Join(uncovered, ", "))
	}

	var ghost []string
	for k := range bound {
		if _, ok := known[k]; !ok {
			ghost = append(ghost, "binding:"+k)
		}
	}
	for k := range omitted {
		if _, ok := known[k]; !ok {
			ghost = append(ghost, "omitted:"+k)
		}
	}
	if len(ghost) > 0 {
		sort.Strings(ghost)
		t.Fatalf("manifest references methods not present on service %s: %s", serviceName, strings.Join(ghost, ", "))
	}
}

func lookupService(fd protoreflect.FileDescriptor, name string) protoreflect.ServiceDescriptor {
	services := fd.Services()
	for i := 0; i < services.Len(); i++ {
		s := services.Get(i)
		if string(s.Name()) == name {
			return s
		}
	}
	return nil
}

// validateManifestAgainstSchema schema-validates raw manifest bytes against
// .vrooli/schemas/cli-manifest.schema.json. Locates the repo root by walking
// up from the test's working directory until .vrooli/repo-contract.json is
// found.
func validateManifestAgainstSchema(t *testing.T, raw []byte) error {
	t.Helper()
	repoRoot, err := findRepoRootFromCWD()
	if err != nil {
		return fmt.Errorf("locate repo root: %w", err)
	}
	schemaPath, err := repocontract.SchemaPath(repoRoot, cliManifestSchemaFile)
	if err != nil {
		return fmt.Errorf("resolve schema path: %w", err)
	}
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema %s: %w", schemaPath, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(cliManifestSchemaFile, bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	schema, err := compiler.Compile(cliManifestSchemaFile)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("unmarshal manifest: %w", err)
	}
	return schema.Validate(doc)
}

func findRepoRootFromCWD() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .vrooli/repo-contract.json above %s", dir)
		}
		dir = parent
	}
}
