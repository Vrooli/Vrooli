package export

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/storage"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// WriteHTMLBundle packages the scenario-built ReplayExportPage with its ReplaySpec
// and resolved assets. The UI bundle remains the only replay presentation owner.
func WriteHTMLBundle(
	ctx context.Context,
	writer io.Writer,
	spec *exportsv1.ReplaySpec,
	storageClient storage.StorageInterface,
	log *logrus.Logger,
	baseURL string,
) error {
	if spec == nil {
		return fmt.Errorf("html export requires replay spec")
	}
	uiDist, err := resolveReplayUIDist()
	if err != nil {
		return err
	}
	zipWriter := zip.NewWriter(writer)
	assetPaths := make(map[string]string, len(spec.GetAssets()))
	for index, asset := range spec.GetAssets() {
		if asset == nil {
			continue
		}
		localPath, err := writeReplayAsset(ctx, zipWriter, asset, index, storageClient, log, baseURL)
		if err != nil {
			if log != nil {
				log.WithError(err).WithField("asset_id", asset.GetId()).Warn("Failed to include replay asset in HTML bundle")
			}
			continue
		}
		if localPath != "" && strings.TrimSpace(asset.GetId()) != "" {
			assetPaths[asset.GetId()] = localPath
		}
	}

	standaloneSpec := proto.Clone(spec).(*exportsv1.ReplaySpec)
	for _, asset := range standaloneSpec.GetAssets() {
		if localPath := assetPaths[asset.GetId()]; localPath != "" {
			asset.Source = localPath
			asset.Thumbnail = localPath
		}
	}
	if err := writeReplayUIBundle(zipWriter, uiDist, standaloneSpec); err != nil {
		_ = zipWriter.Close()
		return err
	}

	generatedAt := time.Now().UTC().Format(time.RFC3339)
	readme := fmt.Sprintf("Vrooli Ascension Replay\n\nGenerated: %s\nFrames: %d\n\nOpen index.html in a browser to view the replay.\n",
		generatedAt, len(spec.GetFrames()))
	if err := writeZipFile(zipWriter, "README.txt", []byte(readme)); err != nil {
		_ = zipWriter.Close()
		return err
	}
	return zipWriter.Close()
}

func resolveReplayUIDist() (string, error) {
	var roots []string
	if scenarioDir := strings.TrimSpace(os.Getenv("VROOLI_SCENARIO_DIR")); scenarioDir != "" {
		roots = append(roots, scenarioDir)
	}
	if repoRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); repoRoot != "" {
		roots = append(roots, filepath.Join(repoRoot, "scenarios", "browser-automation-studio"))
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve BAS UI replay bundle: %w", err)
	}
	for dir := filepath.Clean(cwd); ; dir = filepath.Dir(dir) {
		roots = append(roots, dir, filepath.Join(dir, "scenarios", "browser-automation-studio"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	for _, root := range roots {
		candidate := filepath.Join(root, "ui", "dist")
		if info, statErr := os.Stat(filepath.Join(candidate, "src", "export", "composer.html")); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("BAS UI replay bundle is missing; build the scenario-owned ui/dist before exporting HTML")
}

func writeReplayUIBundle(zipWriter *zip.Writer, uiDist string, spec *exportsv1.ReplaySpec) error {
	pagePath := filepath.Join(uiDist, "src", "export", "composer.html")
	page, err := os.ReadFile(pagePath)
	if err != nil {
		return fmt.Errorf("read built replay composer page: %w", err)
	}
	payload, err := protojson.Marshal(spec)
	if err != nil {
		return fmt.Errorf("marshal replay spec bootstrap: %w", err)
	}
	bootstrap, err := json.Marshal(struct {
		PayloadJSON string `json:"payloadJson"`
	}{PayloadJSON: string(payload)})
	if err != nil {
		return fmt.Errorf("encode replay bootstrap: %w", err)
	}
	headEnd := strings.Index(strings.ToLower(string(page)), "</head>")
	if headEnd < 0 {
		return fmt.Errorf("built replay composer page has no head element")
	}
	injection := []byte("<script>window.__BAS_EXPORT_BOOTSTRAP__=" + string(bootstrap) + ";</script>\n")
	page = append(page[:headEnd], append(injection, page[headEnd:]...)...)
	page = []byte(strings.ReplaceAll(string(page), "../../", ""))
	if err := writeZipFile(zipWriter, "index.html", page); err != nil {
		return err
	}

	for _, source := range []string{"export/composer.js", "assets"} {
		sourcePath := filepath.Join(uiDist, filepath.FromSlash(source))
		if source == "assets" {
			err = filepath.WalkDir(sourcePath, func(filePath string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || strings.HasSuffix(entry.Name(), ".map") || strings.HasSuffix(entry.Name(), ".gz") {
					return nil
				}
				rel, relErr := filepath.Rel(uiDist, filePath)
				if relErr != nil {
					return relErr
				}
				return writeFileToZip(zipWriter, filepath.ToSlash(rel), filePath)
			})
		} else {
			err = writeFileToZip(zipWriter, source, sourcePath)
		}
		if err != nil {
			return fmt.Errorf("package built replay asset %s: %w", source, err)
		}
	}
	return nil
}

func writeFileToZip(zipWriter *zip.Writer, archivePath, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeZipFileFromReader(zipWriter, archivePath, file)
}

func writeReplayAsset(
	ctx context.Context,
	zipWriter *zip.Writer,
	asset *exportsv1.ReplayAsset,
	index int,
	storageClient storage.StorageInterface,
	log *logrus.Logger,
	baseURL string,
) (string, error) {
	source := strings.TrimSpace(asset.GetSource())
	if source == "" || strings.HasPrefix(source, "inline:") {
		return "", nil
	}

	filename := buildAssetFilename(asset, index)
	if filename == "" {
		return "", nil
	}
	targetPath := path.Join("assets", filename)

	if storageClient != nil {
		if objectName, ok := resolveScreenshotObjectName(source); ok {
			reader, _, err := storageClient.GetScreenshot(ctx, objectName)
			if err != nil {
				return "", err
			}
			defer reader.Close()
			if err := writeZipFileFromReader(zipWriter, targetPath, reader); err != nil {
				return "", err
			}
			return targetPath, nil
		}
	}

	assetURL, err := resolveAssetURL(source, baseURL)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("asset download failed (%d)", resp.StatusCode)
	}
	if err := writeZipFileFromReader(zipWriter, targetPath, resp.Body); err != nil {
		return "", err
	}
	if log != nil {
		log.WithField("asset_url", assetURL).Debug("Downloaded replay asset for HTML export")
	}
	return targetPath, nil
}

func buildAssetFilename(asset *exportsv1.ReplayAsset, index int) string {
	rawID := strings.TrimSpace(asset.GetId())
	if rawID == "" {
		rawID = fmt.Sprintf("asset-%d", index+1)
	}
	safeID := sanitizeAssetID(rawID)
	ext := resolveAssetExtension(asset.GetSource())
	if ext == "" {
		ext = ".bin"
	}
	return safeID + ext
}

func sanitizeAssetID(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "asset"
	}
	return result
}

func resolveAssetExtension(source string) string {
	if source == "" {
		return ""
	}
	parsed, err := url.Parse(source)
	if err == nil {
		if ext := path.Ext(parsed.Path); ext != "" {
			return ext
		}
	}
	if ext := path.Ext(source); ext != "" {
		return ext
	}
	return ""
}

func resolveScreenshotObjectName(source string) (string, bool) {
	parsed, err := url.Parse(source)
	if err != nil {
		return "", false
	}
	assetPath := parsed.Path
	if assetPath == "" {
		return "", false
	}
	assetPath = strings.TrimPrefix(assetPath, "/")
	for _, prefix := range []string{"api/v1/screenshots/thumbnail/", "api/v1/screenshots/"} {
		if strings.HasPrefix(assetPath, prefix) {
			return strings.TrimPrefix(assetPath, prefix), true
		}
	}
	return "", false
}

func resolveAssetURL(source string, baseURL string) (string, error) {
	parsed, err := url.Parse(source)
	if err == nil && parsed.IsAbs() {
		return parsed.String(), nil
	}
	if strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("base URL required to resolve asset %q", source)
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimPrefix(source, "/")
	rel, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(rel).String(), nil
}

func writeZipFile(zipWriter *zip.Writer, name string, data []byte) error {
	if zipWriter == nil {
		return fmt.Errorf("zip writer is nil")
	}
	entry, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = entry.Write(data)
	return err
}

func writeZipFileFromReader(zipWriter *zip.Writer, name string, reader io.Reader) error {
	if zipWriter == nil {
		return fmt.Errorf("zip writer is nil")
	}
	entry, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, reader)
	return err
}
