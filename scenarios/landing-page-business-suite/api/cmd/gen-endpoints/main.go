// gen-endpoints derives .vrooli/endpoints.json from the API route registry.
//
// Transport ownership and deliberate REST exceptions are recorded in
// docs/reference/transport-canon.md. The mux registry remains the authoritative
// endpoint inventory; this command parses registration calls rather than asking
// contributors to keep a second endpoint list in sync by hand.
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

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/measures"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const schemaReference = "../../test-genie/schemas/endpoints.schema.json"

type endpointManifest struct {
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

type route struct {
	Path   string
	Method string
}

func main() {
	output := flag.String("output", "../.vrooli/endpoints.json", "path to write endpoints.json")
	routes := flag.String("routes", "routes.go", "path to the mux route registry")
	flag.Parse()

	if err := generate(*routes, *output); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}

func generate(routesPath, outputPath string) error {
	routes, err := inventoryRoutes(routesPath)
	if err != nil {
		return err
	}

	manifest := endpointManifest{
		Schema:    schemaReference,
		Version:   "1.0.0",
		Endpoints: endpointsFor(routes),
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal endpoint manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return fmt.Errorf("create endpoint manifest directory: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0o600); err != nil {
		return fmt.Errorf("write endpoint manifest: %w", err)
	}
	return nil
}

// inventoryRoutes combines literal mux routes with Connect procedures that
// are mounted dynamically by generated handlers. Adding an RPC to the Measures
// proto therefore changes this inventory without a second handwritten list.
func inventoryRoutes(routesPath string) ([]route, error) {
	literalRoutes, err := registeredRoutes(routesPath)
	if err != nil {
		return nil, err
	}
	return sortedUniqueRoutes(append(literalRoutes, connectRoutes()...)), nil
}

// registeredRoutes extracts literal mux registrations of the canonical form
// router.HandleFunc("/path", handler).Methods("GET", ...). Keeping this AST
// based makes formatting changes harmless while deliberately rejecting dynamic
// paths or methods, which would make endpoint inventory unverifiable.
func registeredRoutes(path string) ([]route, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse route registry %s: %w", path, err)
	}

	seen := make(map[route]struct{})
	var routes []route
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || selectorName(call.Fun) != "Methods" {
			return true
		}
		handleCall, ok := selectorReceiverCall(call.Fun)
		if !ok || selectorName(handleCall.Fun) != "HandleFunc" || len(handleCall.Args) == 0 {
			return true
		}
		path, ok := stringLiteral(handleCall.Args[0])
		if !ok {
			return true
		}
		for _, argument := range call.Args {
			method, ok := stringLiteral(argument)
			if !ok {
				continue
			}
			candidate := route{Path: path, Method: method}
			if _, exists := seen[candidate]; !exists {
				seen[candidate] = struct{}{}
				routes = append(routes, candidate)
			}
		}
		return true
	})
	if len(routes) == 0 {
		return nil, fmt.Errorf("no literal HandleFunc(...).Methods(...) registrations found in %s", path)
	}
	return sortedUniqueRoutes(routes), nil
}

func connectRoutes() []route {
	routes := serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_account_proto.Services().ByName("AccountService"))
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_pricing_proto.Services().ByName("PricingService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_billing_proto.Services().ByName("LandingPagePaymentsService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_branding_proto.Services().ByName("BrandingService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_settings_proto.Services().ByName("StripeSettingsService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_variant_space_proto.Services().ByName("VariantSpaceService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_seo_proto.Services().ByName("SeoService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_config_proto.Services().ByName("LandingConfigService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_bundles_proto.Services().ByName("BundleAdminService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_coupons_proto.Services().ByName("CouponAdminService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_assets_proto.Services().ByName("AssetsService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_variant_proto.Services().ByName("VariantService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_metrics_proto.Services().ByName("MetricsService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_feedback_proto.Services().ByName("FeedbackService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_intelligence_proto.Services().ByName("IntelligenceService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_admin_proto.Services().ByName("AdministrationService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_admin_proto.Services().ByName("AdminAuthService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_admin_proto.Services().ByName("AdminResetService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_admin_proto.Services().ByName("AdminProfileService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_deployment_proto.Services().ByName("DeploymentService"))...)
	routes = append(routes, serviceRoutes(lpbsv1.File_landing_page_business_suite_v1_docs_proto.Services().ByName("DocsService"))...)
	routes = append(routes, serviceRoutesNamed(lpbsv1.File_landing_page_business_suite_v1_download_proto.Services().ByName("DownloadService"), "AuthorizeDownload", "ListDownloadApps", "CreateDownloadApp", "SaveDownloadApp", "DeleteDownloadApp")...)
	return append(routes, serviceRoutes(measuresv1.File_landing_page_business_suite_v1_measures_measures_proto.Services().ByName("MeasuresService"))...)
}

func serviceRoutes(service protoreflect.ServiceDescriptor) []route {
	if service == nil {
		return nil
	}
	routes := make([]route, 0, service.Methods().Len())
	for index := 0; index < service.Methods().Len(); index++ {
		method := service.Methods().Get(index)
		routes = append(routes, route{Path: "/" + string(service.FullName()) + "/" + string(method.Name()), Method: "POST"})
	}
	return routes
}

func serviceRoutesNamed(service protoreflect.ServiceDescriptor, names ...string) []route {
	if service == nil {
		return nil
	}
	routes := make([]route, 0, len(names))
	for _, name := range names {
		method := service.Methods().ByName(protoreflect.Name(name))
		if method != nil {
			routes = append(routes, route{Path: "/" + string(service.FullName()) + "/" + string(method.Name()), Method: "POST"})
		}
	}
	return routes
}

func sortedUniqueRoutes(routes []route) []route {
	seen := make(map[route]struct{}, len(routes))
	unique := make([]route, 0, len(routes))
	for _, route := range routes {
		if _, exists := seen[route]; exists {
			continue
		}
		seen[route] = struct{}{}
		unique = append(unique, route)
	}
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].Path == unique[j].Path {
			return unique[i].Method < unique[j].Method
		}
		return unique[i].Path < unique[j].Path
	})
	return unique
}

func endpointsFor(routes []route) []endpoint {
	endpoints := make([]endpoint, 0, len(routes))
	for _, route := range routes {
		endpoints = append(endpoints, endpoint{
			ID:          endpointID(route),
			Path:        route.Path,
			Method:      route.Method,
			Summary:     route.Method + " " + route.Path,
			Description: "Generated from api/routes.go; update the mux registration, then run make endpoints.",
			Category:    categoryFor(route.Path),
		})
	}
	return endpoints
}

func selectorName(expr ast.Expr) string {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return selector.Sel.Name
}

func selectorReceiverCall(expr ast.Expr) (*ast.CallExpr, bool) {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	call, ok := selector.X.(*ast.CallExpr)
	return call, ok
}

func stringLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

func endpointID(route route) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(route.Method))
	for _, r := range route.Path {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func categoryFor(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for _, segment := range segments {
		switch segment {
		case "admin", "billing", "metrics", "feedback", "waitlist", "usage", "ai", "auth", "downloads", "variants", "branding", "assets", "seo":
			return segment
		}
	}
	return "system"
}
