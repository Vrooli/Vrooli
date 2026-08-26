// Package gen renders a scenario's .vrooli/endpoints.json from the
// endpoint descriptors its handler modules export, and enforces that the
// API surface is covered by the scenario's cli/manifest.json.
//
// This is the one shared generator body that replaces the ~285-line
// gen-endpoints/main.go copy that used to live (and drift) in every
// proto-first scenario. Each scenario's api/cmd/gen-endpoints/main.go is
// now a thin wrapper:
//
//	package main
//
//	import (
//		"flag"
//		"fmt"
//		"os"
//
//		gen "github.com/vrooli/api-core/endpoints/gen"
//		"<scenario>/internal/modules"
//	)
//
//	func main() {
//		output := flag.String("output", "../.vrooli/endpoints.json", "path to write endpoints.json")
//		flag.Parse()
//		if err := gen.Generate(modules.AllEndpoints(), "../cli/manifest.json", *output); err != nil {
//			fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
//			os.Exit(1)
//		}
//	}
//
// CI runs `make endpoints && git diff --exit-code .vrooli/endpoints.json`
// so a regenerated file that differs from what's checked in fails the
// build with an actionable diff. The fix is always: run `make endpoints`
// locally and commit.
package gen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/api-core/endpoints"
)

const (
	// endpointsSchemaRef is the $schema reference written into every
	// scenario's endpoints.json. It is a documentation pointer (relative to
	// the generated file), uniform across the fleet.
	endpointsSchemaRef = "../../../../scripts/scenarios/schemas/endpoints.schema.json"
	endpointsVersion   = "1.0.0"
)

// manifestOut is the top-level shape written to endpoints.json. There is no
// cli_commands[] or cli_mapping section: those had zero runtime consumers
// and only fed circular self-checks. The CLI surface SSOT is
// cli/manifest.json; this file describes the API surface.
type manifestOut struct {
	Schema    string                         `json:"$schema"`
	Version   string                         `json:"version"`
	Endpoints []endpoints.EndpointDescriptor `json:"endpoints"`
}

// Generate writes outputPath from eps after validating the transport
// contract and CLI coverage contract against the cli manifest at
// manifestPath. It returns an error (and writes nothing) if either
// contract is violated, so drift fails `make endpoints` with an actionable
// message rather than shipping.
func Generate(eps []endpoints.EndpointDescriptor, manifestPath, outputPath string) error {
	if err := ValidateAgainstManifest(eps, manifestPath); err != nil {
		return err
	}

	m := manifestOut{
		Schema:    endpointsSchemaRef,
		Version:   endpointsVersion,
		Endpoints: eps,
	}
	// Disable HTML-safe escaping so '<', '>' and '&' stay readable — endpoint
	// paths, example curls and arg placeholders contain those characters.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("marshal endpoints manifest: %w", err)
	}
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}
	return nil
}

// ValidateAgainstManifest runs the transport contract and CLI coverage
// contract for a set of endpoint descriptors against the scenario's
// cli/manifest.json, WITHOUT writing any file. Generate calls it before
// emitting endpoints.json. Scenarios that must assemble their endpoints.json
// themselves — e.g. ones still carrying un-migrated, hand-authored REST
// sections the modules registry does not yet produce — call it directly to
// enforce the exact same single-sourced contract instead of reimplementing it.
func ValidateAgainstManifest(eps []endpoints.EndpointDescriptor, manifestPath string) error {
	mf, err := loadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load cli manifest %s: %w", manifestPath, err)
	}
	if err := validateTransport(eps); err != nil {
		return err
	}
	return validateCLICoverage(eps, mf)
}

// --- cli/manifest.json (minimal read-only view) ----------------------------

// manifestFile is the subset of cli/manifest.json the generator needs:
// each command's connect-rpc binding and the omitted
// methods. Full structural/schema validation of the manifest is cli-health's
// job (the manifestvalidation service) and the per-scenario
// RequireProtoServiceCoverage test; here we only read what the coverage gate
// requires.
type manifestFile struct {
	Groups []struct {
		Name     string `json:"name"`
		Commands []struct {
			Name    string `json:"name"`
			Binding struct {
				Service string `json:"service"`
				Method  string `json:"method"`
			} `json:"binding"`
		} `json:"commands"`
	} `json:"groups"`
	Omitted []struct {
		Service string `json:"service"`
		Method  string `json:"method"`
	} `json:"omitted"`
}

type binding struct {
	group   string
	command string
}

type manifestView struct {
	bindings map[string]binding  // "Service/Method" -> group+command
	omitted  map[string]struct{} // "Service/Method"
}

func loadManifest(path string) (manifestView, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return manifestView{}, err
	}
	var mf manifestFile
	if err := json.Unmarshal(data, &mf); err != nil {
		return manifestView{}, fmt.Errorf("parse: %w", err)
	}
	v := manifestView{
		bindings: make(map[string]binding),
		omitted:  make(map[string]struct{}),
	}
	for _, g := range mf.Groups {
		for _, c := range g.Commands {
			if c.Binding.Service == "" || c.Binding.Method == "" {
				continue
			}
			key := c.Binding.Service + "/" + c.Binding.Method
			if _, dup := v.bindings[key]; dup {
				return manifestView{}, fmt.Errorf("duplicate binding %s in cli manifest", key)
			}
			v.bindings[key] = binding{group: g.Name, command: c.Name}
		}
	}
	for _, o := range mf.Omitted {
		if o.Service == "" || o.Method == "" {
			continue
		}
		v.omitted[o.Service+"/"+o.Method] = struct{}{}
	}
	return v, nil
}

// --- CLI coverage gate ------------------------------------------------------

// validateCLICoverage enforces the API↔CLI coverage gate using
// cli/manifest.json as the single source of truth for the CLI surface:
//
//   - Every manifest binding must have a matching Connect endpoint
//     (catches a binding to a method no handler exposes).
//   - Every Connect endpoint that is NOT bound must be declared in the
//     manifest's omitted[] (server streams, multipart edges with
//     hand-appended commands, etc.). An unbound, un-omitted Connect endpoint
//     is drift: the manifest neither binds nor justifies it.
//
// REST endpoints (non-/vrooli. paths) are handled by validateTransport; their
// CLI form is not a connect-rpc binding the manifest can express.
func validateCLICoverage(eps []endpoints.EndpointDescriptor, mf manifestView) error {
	var violations []string

	endpointKeys := make(map[string]struct{})
	for _, e := range eps {
		svc, method, ok := parseProcedure(e.Path)
		if !ok {
			continue
		}
		key := svc + "/" + method
		endpointKeys[key] = struct{}{}
		fullKey := ""
		if fullService, _, ok := parseProcedureFull(e.Path); ok {
			fullKey = fullService + "/" + method
			endpointKeys[fullKey] = struct{}{}
		}

		if _, bound := mf.bindings[key]; bound {
			continue
		}
		if fullKey != "" {
			if _, bound := mf.bindings[fullKey]; bound {
				continue
			}
		}

		// Not bound: must be explicitly omitted in the manifest.
		if _, omitted := mf.omitted[key]; !omitted {
			if fullKey != "" {
				if _, omitted := mf.omitted[fullKey]; omitted {
					continue
				}
			}
			violations = append(violations, fmt.Sprintf(
				"endpoint %q (%s) is a Connect procedure with no binding and no omission in cli/manifest.json; "+
					"bind it under groups[] or declare it in omitted[] with a reason",
				e.ID, key))
		}
	}

	// Every binding must have a matching endpoint.
	for key, b := range mf.bindings {
		if _, ok := endpointKeys[key]; !ok {
			violations = append(violations, fmt.Sprintf(
				"cli/manifest.json binds %s to command %q but no API endpoint exposes that method; "+
					"remove the binding or register the endpoint",
				key, b.group+" "+b.command))
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		return fmt.Errorf("API↔CLI coverage validation failed (cli/manifest.json is the source of truth):\n  - %s",
			strings.Join(violations, "\n  - "))
	}
	return nil
}

// parseProcedure extracts the short service name and method from a Connect
// procedure path. Vrooli proto services are namespaced under
// vrooli.<scenario>.v1[.<domain>].<Service>, so the service is the last
// dotted segment before '/' and the method follows. Returns ok=false for
// any path that is not a "/…/…" Connect procedure (REST paths, "/health").
func parseProcedure(path string) (service, method string, ok bool) {
	if !strings.HasPrefix(path, "/vrooli.") {
		return "", "", false
	}
	p := strings.TrimPrefix(path, "/")
	slash := strings.IndexByte(p, '/')
	if slash < 0 {
		return "", "", false
	}
	left, method := p[:slash], p[slash+1:]
	dot := strings.LastIndexByte(left, '.')
	if dot < 0 || method == "" {
		return "", "", false
	}
	return left[dot+1:], method, true
}

// parseProcedureFull returns the fully-qualified service name from a Connect
// procedure path. Coverage accepts both this canonical name and the historic
// short service alias so manifests can qualify bindings when descriptor-backed
// consumers need an unambiguous request schema.
func parseProcedureFull(path string) (service, method string, ok bool) {
	if !strings.HasPrefix(path, "/vrooli.") {
		return "", "", false
	}
	p := strings.TrimPrefix(path, "/")
	slash := strings.IndexByte(p, '/')
	if slash < 0 || slash == 0 || slash == len(p)-1 {
		return "", "", false
	}
	return p[:slash], p[slash+1:], true
}

// --- transport contract ----------------------------------------------------

// validateTransport enforces the proto/Connect-RPC anti-drift contract:
// every EndpointDescriptor.Path must either be a generated Connect procedure
// constant (which starts with "/vrooli.") or carry a RESTException declaring
// one of the mechanically-allowed REST reasons.
func validateTransport(eps []endpoints.EndpointDescriptor) error {
	var violations []string
	for _, e := range eps {
		isConnect := strings.HasPrefix(e.Path, "/vrooli.")
		hasException := e.RESTException != nil
		switch {
		case isConnect && hasException:
			violations = append(violations, fmt.Sprintf(
				"endpoint %q: Path %q is a Connect procedure but has RESTException set; remove RESTException",
				e.ID, e.Path))
		case !isConnect && !hasException:
			violations = append(violations, fmt.Sprintf(
				"endpoint %q: Path %q is not a Connect procedure (must start with %q) and has no RESTException; "+
					"either reference a generated *Procedure constant from packages/proto/gen, or tag with "+
					"RESTException{Reason: one of multipart_upload|webhook_receiver|third_party_shape|ops_probe}",
				e.ID, e.Path, "/vrooli."))
		case !isConnect && hasException:
			if !validRESTReason(e.RESTException.Reason) {
				violations = append(violations, fmt.Sprintf(
					"endpoint %q: RESTException.Reason %q is not one of the allowed reasons "+
						"(multipart_upload, webhook_receiver, third_party_shape, ops_probe)",
					e.ID, e.RESTException.Reason))
			}
		}
	}
	if len(violations) > 0 {
		return fmt.Errorf("transport validation failed:\n  - %s", strings.Join(violations, "\n  - "))
	}
	return nil
}

func validRESTReason(r endpoints.RESTReason) bool {
	switch r {
	case endpoints.RESTReasonMultipartUpload,
		endpoints.RESTReasonWebhookReceiver,
		endpoints.RESTReasonThirdPartyShape,
		endpoints.RESTReasonOpsProbe,
		endpoints.RESTReasonStreamUpgrade:
		return true
	}
	return false
}
