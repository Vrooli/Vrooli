package content

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	commands := deps.EndpointCommands([]support.EndpointDef{
		{Name: "branding", Method: "GET", Path: "/branding", Description: "Get public branding"},
		{Name: "admin-branding-get", Method: "GET", Path: "/admin/branding", Description: "Get branding (admin)"},
		{Name: "admin-branding-update", Method: "PUT", Path: "/admin/branding", Description: "Update branding (admin)"},
		{Name: "admin-branding-clear-field", Method: "POST", Path: "/admin/branding/clear-field", Description: "Clear branding field (admin)"},
		{Name: "admin-assets-list", Method: "GET", Path: "/admin/assets", Description: "List assets (admin)"},
		{Name: "admin-assets-get", Method: "GET", Path: "/admin/assets/{id}", Description: "Get asset (admin)"},
		{Name: "admin-assets-delete", Method: "DELETE", Path: "/admin/assets/{id}", Description: "Delete asset (admin)"},
		{Name: "uploads-get", Method: "GET", Path: "/uploads/{path}", Description: "Fetch uploaded asset by path", AllowRawPath: true},
		{Name: "sitemap", Method: "GET", Path: "/sitemap.xml", Description: "Fetch sitemap", Root: true},
		{Name: "robots", Method: "GET", Path: "/robots.txt", Description: "Fetch robots.txt", Root: true},
	})
	commands = append(commands, seoCommand(deps), updateVariantSEOCommand(deps), cliapp.Command{
		Name:        "admin-assets-upload",
		NeedsAPI:    true,
		Description: "Upload asset (admin)",
		Run:         func(args []string) error { return runAssetsUpload(deps, args) },
	})
	return cliapp.CommandGroup{Title: "Content", Commands: commands}
}

func seoClient(deps support.Dependencies) (lpbsconnect.SeoServiceClient, error) {
	core := deps.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app is not initialized")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return lpbsconnect.NewSeoServiceClient(httpClient, baseURL), nil
}

func adminSEOClient(deps support.Dependencies) (lpbsconnect.SeoServiceClient, error) {
	httpClient, baseURL, err := deps.AdminConnectHTTPClient()
	if err != nil {
		return nil, err
	}
	return lpbsconnect.NewSeoServiceClient(httpClient, baseURL), nil
}

func seoCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(
		func(op cliapp.OperationContext) (map[string]any, error) {
			client, err := seoClient(deps)
			if err != nil {
				return nil, err
			}
			response, err := client.GetVariantSEO(context.Background(), connect.NewRequest(&lpbsv1.GetVariantSEORequest{Slug: strings.TrimSpace(op.Positional("slug"))}))
			if err != nil {
				return nil, cliapp.WrapAPIError("get variant SEO", err, nil)
			}
			return legacySEOResponse(response.Msg), nil
		},
		func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Fetched variant SEO metadata."}}
		},
	)
	return (cliapp.Command{
		Name:         "seo",
		NeedsAPI:     true,
		Description:  "Get SEO metadata for a variant through the generated Connect contract (SLUG) [--json]",
		Args:         cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "slug", Required: true}}},
		Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction},
	}).WithPrimitive(operation)
}

func updateVariantSEOCommand(deps support.Dependencies) cliapp.Command {
	operation := cliapp.Action(
		func(op cliapp.OperationContext) (map[string]any, error) {
			payload, err := support.ParseBody(op.Flag("body"))
			if err != nil {
				return nil, err
			}
			if len(payload) == 0 {
				return nil, fmt.Errorf("--body JSON payload is required")
			}
			config := &sharedv1.VariantSEOConfig{}
			if err := protojson.Unmarshal(payload, config); err != nil {
				return nil, fmt.Errorf("parse SEO configuration: %w", err)
			}
			client, err := adminSEOClient(deps)
			if err != nil {
				return nil, err
			}
			response, err := client.UpdateVariantSEO(context.Background(), connect.NewRequest(&lpbsv1.UpdateVariantSEORequest{
				Slug: strings.TrimSpace(op.Positional("slug")), Config: config,
			}))
			if err != nil {
				return nil, cliapp.WrapAPIError("update variant SEO", err, nil)
			}
			return map[string]any{"success": response.Msg.GetSuccess(), "updated_at": response.Msg.GetUpdatedAt()}, nil
		},
		func(cliapp.OperationContext, map[string]any) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Updated variant SEO metadata."}}
		},
	)
	return (cliapp.Command{
		Name:         "admin-variant-seo-update",
		NeedsAPI:     true,
		Description:  "Update variant SEO through the generated Connect contract (SLUG --body JSON) [--json]",
		Args:         cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "slug", Required: true}}, Flags: []cliapp.Flag{{Name: "body", Description: "SEO JSON payload or @file.json", Required: true}}},
		Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction},
	}).WithPrimitive(operation)
}

func legacySEOResponse(response *lpbsv1.SEOResponse) map[string]any {
	result := map[string]any{
		"site_name": response.GetSiteName(), "title": response.GetTitle(), "description": response.GetDescription(),
		"og_title": response.GetOgTitle(), "og_description": response.GetOgDescription(), "noindex": response.GetNoindex(),
	}
	for key, value := range map[string]string{
		"og_image_url": response.GetOgImageUrl(), "twitter_card": response.GetTwitterCard(), "canonical_url": response.GetCanonicalUrl(),
		"favicon_url": response.GetFaviconUrl(), "apple_touch_icon_url": response.GetAppleTouchIconUrl(), "theme_primary_color": response.GetThemePrimaryColor(),
	} {
		if value != "" {
			result[key] = value
		}
	}
	if structuredData := response.GetStructuredData(); structuredData != nil {
		result["structured_data"] = structuredData.AsMap()
	}
	return result
}

func runAssetsUpload(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("admin-assets-upload", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to file")
	category := fs.String("category", "", "Asset category")
	altText := fs.String("alt-text", "", "Alt text")
	uploadedBy := fs.String("uploaded-by", "", "Uploaded by")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	pathValue := strings.TrimSpace(*filePath)
	if pathValue == "" {
		return fmt.Errorf("usage: admin-assets-upload --file path [--category=...] [--alt-text=...] [--uploaded-by=...] [--json]")
	}

	buf, contentType, err := support.BuildMultipartForm(pathValue, "file", map[string]string{
		"category":    strings.TrimSpace(*category),
		"alt_text":    strings.TrimSpace(*altText),
		"uploaded_by": strings.TrimSpace(*uploadedBy),
	})
	if err != nil {
		return err
	}

	endpoint, err := deps.ResolveURL("/admin/assets/upload", false, nil)
	if err != nil {
		return err
	}
	session, err := deps.LoadAdminSession()
	if err != nil {
		return err
	}
	if strings.TrimSpace(session.Session) == "" {
		return fmt.Errorf("admin session not configured. Run admin-login first")
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	for key, value := range deps.ScenarioApp().APIClient.AuthHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Cookie", fmt.Sprintf("admin_session=%s", session.Session))
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{Timeout: deps.ScenarioApp().HTTPClient.Timeout()}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = deps.ClearAdminSession()
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return cliutil.ParseAPIError(resp.StatusCode, data)
	}

	if *jsonOut {
		cliutil.PrintJSON(data)
		return nil
	}
	cliutil.PrintJSON(data)
	return nil
}
