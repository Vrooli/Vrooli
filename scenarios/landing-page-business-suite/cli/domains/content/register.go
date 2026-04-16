package content

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
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
		{Name: "seo", Method: "GET", Path: "/seo/{slug}", Description: "Get SEO metadata for variant"},
		{Name: "admin-variant-seo-update", Method: "PUT", Path: "/admin/variants/{slug}/seo", Description: "Update variant SEO"},
		{Name: "sitemap", Method: "GET", Path: "/sitemap.xml", Description: "Fetch sitemap", Root: true},
		{Name: "robots", Method: "GET", Path: "/robots.txt", Description: "Fetch robots.txt", Root: true},
	})
	commands = append(commands, cliapp.Command{
		Name:        "admin-assets-upload",
		NeedsAPI:    true,
		Description: "Upload asset (admin)",
		Run:         func(args []string) error { return runAssetsUpload(deps, args) },
	})
	return cliapp.CommandGroup{Title: "Content", Commands: commands}
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
