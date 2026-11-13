package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/pathfilter"
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func createTestDir(t *testing.T, basePath string, relPath string, files map[string]string) string {
	t.Helper()
	dirPath := filepath.Join(basePath, relPath)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dirPath, err)
	}
	for filename, content := range files {
		createTestFile(t, filepath.Join(dirPath, filename), content)
	}
	return dirPath
}

func createTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// newLightScannerForTest supplies catalog-shaped facts from small filesystem
// fixtures. Production LightScanner has no filesystem inventory fallback.
func newLightScannerForTest(root string, timeout time.Duration, excludes ...string) *LightScanner {
	scanner := NewLightScanner(root, timeout, excludes...)
	return scanner.withCodeFacts(localCatalogFactsSource())
}

func localCatalogFactsSource() ReductionFacts {
	return reductionSourceFunc(func(_ context.Context, request *connect.Request[facts.DescribeCodeFactsRequest]) (*connect.Response[facts.CodeFactsReport], error) {
		report := &facts.CodeFactsReport{}
		if len(request.Msg.GetInclude()) == 1 && request.Msg.GetInclude()[0] == facts.FactFamily_FACT_FAMILY_FILE_METRICS {
			metrics := make([]*facts.GenericFact, 0)
			root := request.Msg.GetTarget().GetPath()
			err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				relPath, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return nil
				}
				if info.IsDir() {
					if pathfilter.SkipDir(info.Name()) {
						return filepath.SkipDir
					}
					return nil
				}
				if !info.Mode().IsRegular() || !pathfilter.IsSourceExt(filepath.Ext(path)) {
					return nil
				}
				rel := filepath.ToSlash(relPath)
				language, supported := languageForTestExtension(filepath.Ext(path))
				var metric FileMetric
				var metricErr error
				if supported {
					metric, metricErr = fileMetricForTest(path, rel, language)
				} else {
					var lines int
					lines, metricErr = countLines(path)
					metric = FileMetric{Path: rel, Lines: lines, CodeLines: lines, Extension: filepath.Ext(path)}
				}
				if metricErr != nil {
					return nil
				}
				metrics = append(metrics, &facts.GenericFact{Family: facts.FactFamily_FACT_FAMILY_FILE_METRICS, Subject: rel, Attributes: fileMetricAttributesForTest(metric)})
				return nil
			})
			if err != nil {
				return nil, err
			}
			report.Facts = metrics
		}
		return connect.NewResponse(report), nil
	})
}

func TestCatalogFixtureSourceMatchesMetricExtFiltering(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"main.go": "package main\n", "guide.md": "# guide\n", "module.unknown": "data\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	metrics, err := newLightScannerForTest(root, time.Second).collectFileMetricsContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].Path != "main.go" || metrics[0].Lines != 1 {
		t.Fatalf("fixture facts = %+v", metrics)
	}
}

func TestCatalogFixtureSourceExcludesConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"api/main.go", "generated/main.go"} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	metrics, err := newLightScannerForTest(root, time.Second, "generated/**").collectFileMetricsContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || strings.TrimSpace(metrics[0].Path) != "api/main.go" {
		t.Fatalf("excluded fixture facts = %+v", metrics)
	}
}
