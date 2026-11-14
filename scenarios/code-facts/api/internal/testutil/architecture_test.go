package testutil_test

import (
	"bufio"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var codeFactsDomains = []string{"analysis", "cache", "catalog", "indexcontrol", "proof", "retrieval", "targets"}

func TestDomainPackagesAreTransportFreeAndIndependent(t *testing.T) {
	module := readCodeFactsModule(t)
	domainSet := map[string]bool{}
	for _, domain := range codeFactsDomains {
		domainSet[domain] = true
	}
	for _, domain := range codeFactsDomains {
		root := filepath.Join("..", domain)
		walkSourceFiles(t, root, func(path string) {
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return
			}
			payload, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(payload)
			for _, ambient := range []string{"time.Now()", "os.Getenv(", "http.DefaultClient", "log.Default()"} {
				if strings.Contains(text, ambient) {
					t.Errorf("%s uses ambient dependency %s", path, ambient)
				}
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, payload, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, imported := range file.Imports {
				name, _ := strconv.Unquote(imported.Path.Value)
				if name == "net/http" || name == "connectrpc.com/connect" || name == "github.com/gorilla/mux" {
					t.Errorf("%s imports transport package %s", path, name)
				}
				prefix := module + "/internal/"
				if strings.HasPrefix(name, prefix) {
					other := strings.Split(strings.TrimPrefix(name, prefix), "/")[0]
					if other != domain && domainSet[other] {
						t.Errorf("domain %s imports sibling domain %s in %s", domain, other, path)
					}
				}
			}
		})
	}
}

func TestZoneMapMatchesTopLevelPackageTree(t *testing.T) {
	wantInternal := []string{"analysis", "cache", "catalog", "database", "facts", "fswatch", "httpc", "httpx", "indexcontrol", "logging", "module", "modules", "proof", "registration", "retrieval", "targets", "testutil"}
	wantHandlers := []string{"facts", "health", "validation"}
	assertDirectorySet(t, "..", wantInternal)
	assertDirectorySet(t, filepath.Join("..", "..", "handlers"), wantHandlers)
	architecture := readRelative(t, filepath.Join("..", "..", "..", "docs", "concepts", "ARCHITECTURE.md"))
	for _, directory := range append(prefixAll("api/internal/", wantInternal), prefixAll("api/handlers/", wantHandlers)...) {
		if !strings.Contains(architecture, "`"+directory+"/`") {
			t.Errorf("Zone Map does not declare %s/", directory)
		}
	}
	if !strings.Contains(architecture, "`api/main.go`") {
		t.Error("Zone Map does not declare the API composition root")
	}
}

func TestSeamRegistryMatchesDomainContracts(t *testing.T) {
	seams := readRelative(t, filepath.Join("..", "..", "..", "docs", "internal", "SEAMS.md"))
	for _, seam := range []string{
		"targets.FileSystem", "targets.Resolver", "catalog.Repository", "catalog.Clock",
		"analysis.Analyzer", "analysis.ProjectionStore", "retrieval.Embedder",
		"retrieval.VectorStore", "retrieval.Reranker", "retrieval.Admission",
		"proof.ContractSource", "indexcontrol.JobStore", "indexcontrol.ProcessRunner", "cache.Repository",
		"logging.Logger",
	} {
		if !strings.Contains(seams, "`"+seam+"`") {
			t.Errorf("seam registry does not declare %s", seam)
		}
	}
}

func assertDirectorySet(t *testing.T, root string, want []string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, entry := range entries {
		if entry.IsDir() {
			got = append(got, entry.Name())
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("directories under %s = %v, want %v", root, got, want)
	}
}

func prefixAll(prefix string, names []string) []string {
	result := make([]string, len(names))
	for i, name := range names {
		result[i] = prefix + name
	}
	return result
}

func readRelative(t *testing.T, path string) string {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func readCodeFactsModule(t *testing.T) string {
	t.Helper()
	file, err := os.Open(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if module, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(module)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	t.Fatal("module directive not found")
	return ""
}

func walkSourceFiles(t *testing.T, root string, visit func(string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			visit(path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}
