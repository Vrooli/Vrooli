package downloads

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	commands := deps.EndpointCommands([]support.EndpointDef{
		{Name: "admin-download-apps-list", Method: "GET", Path: "/admin/download-apps", Description: "List download apps"},
		{Name: "admin-download-apps-create", Method: "POST", Path: "/admin/download-apps", Description: "Create download app"},
		{Name: "admin-download-apps-save", Method: "PUT", Path: "/admin/download-apps/{app_key}", Description: "Update download app"},
		{Name: "admin-download-apps-delete", Method: "DELETE", Path: "/admin/download-apps/{app_key}", Description: "Delete download app"},
		{Name: "admin-download-storage-get", Method: "GET", Path: "/admin/download-storage", Description: "Get download storage"},
		{Name: "admin-download-storage-update", Method: "PUT", Path: "/admin/download-storage", Description: "Update download storage"},
		{Name: "admin-download-storage-test", Method: "POST", Path: "/admin/download-storage/test", Description: "Test download storage"},
		{Name: "admin-download-artifacts-list", Method: "GET", Path: "/admin/download-artifacts", Description: "List download artifacts"},
		{Name: "admin-download-artifacts-by-app", Method: "GET", Path: "/admin/download-artifacts/by-app", Description: "List download artifacts by app"},
		{Name: "admin-download-artifacts-presign-upload", Method: "POST", Path: "/admin/download-artifacts/presign-upload", Description: "Presign upload for artifact"},
		{Name: "admin-download-artifacts-commit", Method: "POST", Path: "/admin/download-artifacts/commit", Description: "Commit download artifact"},
		{Name: "admin-download-artifacts-presign-get", Method: "GET", Path: "/admin/download-artifacts/{artifact_id}/presign-get", Description: "Presign get for artifact"},
		{Name: "admin-download-assets-apply", Method: "POST", Path: "/admin/download-assets/apply", Description: "Apply download artifact"},
		{Name: "admin-download-assets-set-current", Method: "POST", Path: "/admin/download-assets/set-current", Description: "Set artifact as current"},
	})
	commands = append(commands, cliapp.Command{
		Name:        "admin-downloads-upload-managed",
		NeedsAPI:    true,
		Description: "Upload + apply managed artifact (presign → upload → commit → apply)",
		Run:         func(args []string) error { return runUploadManaged(deps, args) },
	})
	return cliapp.CommandGroup{Title: "Admin Commerce - Downloads", Commands: commands}
}

func runUploadManaged(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("admin-downloads-upload-managed", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to artifact file")
	appKey := fs.String("app-key", "", "Download app key")
	platform := fs.String("platform", "", "Platform (windows, mac, linux)")
	releaseVersion := fs.String("release-version", "", "Release version (e.g., 1.2.3)")
	releaseNotes := fs.String("release-notes", "", "Release notes")
	checksum := fs.String("checksum", "", "Checksum string (optional)")
	requiresEntitlement := fs.Bool("requires-entitlement", false, "Require entitlement to download")
	metadata := fs.String("metadata", "", "Asset metadata JSON or @file.json")
	sha512Flag := fs.String("sha512", "", "Precomputed base64-encoded SHA512 (computed automatically if omitted)")
	contentType := fs.String("content-type", "", "Override content-type for upload")
	remoteProfile := fs.String("remote-profile", "", "Remote profile ID for proxying admin calls")
	skipApply := fs.Bool("skip-apply", false, "Skip apply step (upload + commit only)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: admin-downloads-upload-managed --file <path> --app-key <app> --platform <platform> --release-version <version> [options]")
	}

	pathValue := strings.TrimSpace(*filePath)
	appKeyValue := strings.TrimSpace(*appKey)
	platformValue, err := support.NormalizeDownloadPlatform(*platform)
	if err != nil {
		return err
	}
	releaseVersionValue := strings.TrimSpace(*releaseVersion)
	if pathValue == "" || appKeyValue == "" || releaseVersionValue == "" {
		return fmt.Errorf("usage: admin-downloads-upload-managed --file <path> --app-key <app> --platform <platform> --release-version <version> [options]")
	}
	if _, err := os.Stat(pathValue); err != nil {
		return fmt.Errorf("artifact file not found: %w", err)
	}

	sha512Value := strings.TrimSpace(*sha512Flag)
	if sha512Value == "" {
		sha512Value, err = support.ComputeSHA512(pathValue)
		if err != nil {
			return fmt.Errorf("compute sha512: %w", err)
		}
	}

	contentTypeValue := support.ResolveContentType(pathValue, strings.TrimSpace(*contentType))
	remoteProfileID := strings.TrimSpace(*remoteProfile)

	var assetMetadata map[string]interface{}
	if strings.TrimSpace(*metadata) != "" {
		metadataBytes, err := support.ParseBody(*metadata)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(metadataBytes, &assetMetadata); err != nil {
			return fmt.Errorf("metadata must be a JSON object: %w", err)
		}
	}

	presignRespBytes, err := deps.RequestAdminJSON(remoteProfileID, "POST", "/admin/download-artifacts/presign-upload", map[string]interface{}{
		"filename":        filepath.Base(pathValue),
		"content_type":    contentTypeValue,
		"app_key":         appKeyValue,
		"platform":        platformValue,
		"release_version": releaseVersionValue,
	})
	if err != nil {
		return err
	}
	var presignResp struct {
		UploadURL       string            `json:"upload_url"`
		RequiredHeaders map[string]string `json:"required_headers"`
		Bucket          string            `json:"bucket"`
		ObjectKey       string            `json:"object_key"`
	}
	if err := json.Unmarshal(presignRespBytes, &presignResp); err != nil {
		return fmt.Errorf("parse presign response: %w", err)
	}
	if strings.TrimSpace(presignResp.UploadURL) == "" || strings.TrimSpace(presignResp.Bucket) == "" || strings.TrimSpace(presignResp.ObjectKey) == "" {
		return fmt.Errorf("presign response missing required fields")
	}

	artifactFile, err := os.Open(pathValue)
	if err != nil {
		return fmt.Errorf("open artifact file: %w", err)
	}
	defer artifactFile.Close()

	uploadReq, err := http.NewRequest("PUT", presignResp.UploadURL, artifactFile)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	for key, value := range presignResp.RequiredHeaders {
		if strings.EqualFold(key, "host") || strings.TrimSpace(value) == "" {
			continue
		}
		uploadReq.Header.Set(key, value)
	}
	if uploadReq.Header.Get("Content-Type") == "" {
		uploadReq.Header.Set("Content-Type", contentTypeValue)
	}

	uploadClient := &http.Client{Timeout: deps.ScenarioApp().HTTPClient.Timeout()}
	uploadResp, err := uploadClient.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer uploadResp.Body.Close()
	if uploadResp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(uploadResp.Body)
		if len(bodyBytes) > 0 {
			return fmt.Errorf("upload failed (%d): %s", uploadResp.StatusCode, strings.TrimSpace(string(bodyBytes)))
		}
		return fmt.Errorf("upload failed (%d)", uploadResp.StatusCode)
	}

	commitRespBytes, err := deps.RequestAdminJSON(remoteProfileID, "POST", "/admin/download-artifacts/commit", map[string]interface{}{
		"bucket":            presignResp.Bucket,
		"object_key":        presignResp.ObjectKey,
		"original_filename": filepath.Base(pathValue),
		"content_type":      contentTypeValue,
		"app_key":           appKeyValue,
		"platform":          platformValue,
		"release_version":   releaseVersionValue,
		"sha512":            sha512Value,
	})
	if err != nil {
		return err
	}
	var artifactResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(commitRespBytes, &artifactResp); err != nil {
		return fmt.Errorf("parse commit response: %w", err)
	}
	if artifactResp.ID == 0 {
		return fmt.Errorf("commit response missing artifact id")
	}

	var applyRespBytes []byte
	if !*skipApply {
		applyPayload := map[string]interface{}{
			"app_key":         appKeyValue,
			"platform":        platformValue,
			"artifact_id":     artifactResp.ID,
			"release_version": releaseVersionValue,
		}
		if strings.TrimSpace(*releaseNotes) != "" {
			applyPayload["release_notes"] = strings.TrimSpace(*releaseNotes)
		}
		if strings.TrimSpace(*checksum) != "" {
			applyPayload["checksum"] = strings.TrimSpace(*checksum)
		}
		if *requiresEntitlement {
			applyPayload["requires_entitlement"] = true
		}
		if assetMetadata != nil {
			applyPayload["metadata"] = assetMetadata
		}
		applyRespBytes, err = deps.RequestAdminJSON(remoteProfileID, "POST", "/admin/download-assets/apply", applyPayload)
		if err != nil {
			return err
		}
	}

	if *jsonOut {
		result := map[string]json.RawMessage{"artifact": json.RawMessage(commitRespBytes)}
		if !*skipApply && len(applyRespBytes) > 0 {
			result["asset"] = json.RawMessage(applyRespBytes)
		}
		out, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("encode response: %w", err)
		}
		cliutil.PrintJSON(out)
		return nil
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Uploaded artifact %s (id: %d)", filepath.Base(pathValue), artifactResp.ID)},
		Changes: func() []string {
			if *skipApply {
				return []string{"Upload and commit completed", "Apply step skipped"}
			}
			return []string{
				"Upload and commit completed",
				fmt.Sprintf("Applied artifact to %s/%s", appKeyValue, platformValue),
			}
		}(),
		NextCommand: []string{"landing-page-business-suite admin-download-artifacts-list --json"},
	})
}
