// Package modules is the single registration point for the scenario's
// API modules' static metadata. Both api/main.go and
// api/cmd/gen-endpoints/main.go import this package to enumerate
// domains uniformly.
//
// The runtime Module(...) constructors stay inline in main.go's
// server.New(...) call — they need live deps (db handle, clock, logger)
// and abstracting them is needless ceremony. This package only handles
// the static side: the Endpoints slice each handler exports for
// codegen, and the Schema() function each handler re-exports for
// EnsureSchemas.
//
// Adding a domain: register its Endpoints in AllEndpoints, its Schema in
// AllSchemas, and (for Connect-mounted domains) its proto FileDescriptor
// in AllProtoFiles. The runtime constructor lands in main.go's server.New
// call.
package modules

import (
	"landing-page-react-vite-api/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	accountH "landing-page-react-vite-api/handlers/account"
	adminH "landing-page-react-vite-api/handlers/admin"
	assetsH "landing-page-react-vite-api/handlers/assets"
	brandingH "landing-page-react-vite-api/handlers/branding"
	bundlesH "landing-page-react-vite-api/handlers/bundles"
	configH "landing-page-react-vite-api/handlers/config"
	contentH "landing-page-react-vite-api/handlers/content"
	docsH "landing-page-react-vite-api/handlers/docs"
	downloadH "landing-page-react-vite-api/handlers/download"
	healthH "landing-page-react-vite-api/handlers/health"
	metricsH "landing-page-react-vite-api/handlers/metrics"
	paymentsH "landing-page-react-vite-api/handlers/payments"
	resetH "landing-page-react-vite-api/handlers/reset"
	seoH "landing-page-react-vite-api/handlers/seo"
	variantH "landing-page-react-vite-api/handlers/variant"
	variantspaceH "landing-page-react-vite-api/handlers/variantspace"

	systemdb "landing-page-react-vite-api/internal/database"
	paymentsettingssvc "landing-page-react-vite-api/internal/paymentsettings"
	plansvc "landing-page-react-vite-api/internal/plan"
	stripesvc "landing-page-react-vite-api/internal/stripe"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, brandingH.Endpoints...)
	out = append(out, contentH.Endpoints...)
	out = append(out, variantH.Endpoints...)
	out = append(out, metricsH.Endpoints...)
	out = append(out, seoH.Endpoints...)
	out = append(out, docsH.Endpoints...)
	out = append(out, variantspaceH.Endpoints...)
	out = append(out, paymentsH.Endpoints...)
	out = append(out, bundlesH.Endpoints...)
	out = append(out, accountH.Endpoints...)
	out = append(out, adminH.Endpoints...)
	out = append(out, resetH.Endpoints...)
	out = append(out, downloadH.Endpoints...)
	out = append(out, assetsH.Endpoints...)
	out = append(out, configH.Endpoints...)
	return out
}

// ProtoFileEntry pairs a domain module's name with the proto
// FileDescriptor whose RPCs that module exposes via Connect-RPC. The
// global parity test walks every entry and asserts each rpc method in
// the FileDescriptor has exactly one matching EndpointDescriptor in
// AllEndpoints().
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "branding", File: landingv1.File_landing_page_react_vite_v1_branding_proto},
		{Module: "content", File: landingv1.File_landing_page_react_vite_v1_content_proto},
		{Module: "variant", File: landingv1.File_landing_page_react_vite_v1_variant_proto},
		{Module: "metrics", File: landingv1.File_landing_page_react_vite_v1_metrics_proto},
		{Module: "seo", File: landingv1.File_landing_page_react_vite_v1_seo_proto},
		{Module: "docs", File: landingv1.File_landing_page_react_vite_v1_docs_proto},
		{Module: "variant_space", File: landingv1.File_landing_page_react_vite_v1_variant_space_proto},
		{Module: "payments", File: landingv1.File_landing_page_react_vite_v1_billing_proto},
		{Module: "bundles", File: landingv1.File_landing_page_react_vite_v1_bundles_proto},
		{Module: "account", File: landingv1.File_landing_page_react_vite_v1_account_proto},
		// admin.proto carries both AdminAuthService and AdminResetService, split
		// across handlers/admin and handlers/reset.
		{Module: "admin", File: landingv1.File_landing_page_react_vite_v1_admin_proto},
		{Module: "download", File: landingv1.File_landing_page_react_vite_v1_download_proto},
		{Module: "assets", File: landingv1.File_landing_page_react_vite_v1_assets_proto},
		{Module: "config", File: landingv1.File_landing_page_react_vite_v1_config_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(systemdb.SystemSchema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(brandingH.Schema),
		// variant must precede content: content_sections FKs variants(id).
		apidb.SchemaProviderFunc(variantH.Schema),
		apidb.SchemaProviderFunc(contentH.Schema),
		apidb.SchemaProviderFunc(metricsH.Schema),
		apidb.SchemaProviderFunc(seoH.Schema),
		apidb.SchemaProviderFunc(docsH.Schema),
		apidb.SchemaProviderFunc(variantspaceH.Schema),
		// Financial cluster: plan (bundle_products/bundle_prices) before stripe
		// (subscriptions/checkout/credits); payment settings is independent.
		apidb.SchemaProviderFunc(plansvc.Schema),
		apidb.SchemaProviderFunc(paymentsettingssvc.Schema),
		apidb.SchemaProviderFunc(stripesvc.Schema),
		// Admin cluster: admin_users, download_apps/download_assets.
		apidb.SchemaProviderFunc(adminH.Schema),
		apidb.SchemaProviderFunc(downloadH.Schema),
		apidb.SchemaProviderFunc(assetsH.Schema),
	}
}
