package preview

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/gorilla/mux"
)

type RuntimeHandler struct {
	logger *log.Logger
}

func NewRuntimeHandler(logger *log.Logger) *RuntimeHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &RuntimeHandler{logger: logger}
}

func (h *RuntimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	moduleName := strings.TrimSpace(vars["module"])
	version := strings.TrimSpace(vars["version"])
	runtimePath := strings.TrimSpace(vars["path"])
	if runtimePath == "" {
		runtimePath = "index.js"
	}
	if strings.HasPrefix(r.URL.Path, "/preview/runtime/npm/") {
		h.servePackageRuntime(w, moduleName, version, runtimePath)
		return
	}
	if !vendoredReactRuntimeVersions[version] {
		http.Error(w, fmt.Sprintf("preview runtime %s@%s is not vendored", moduleName, version), http.StatusNotFound)
		return
	}
	entry, ok := runtimeEntry(moduleName, runtimePath)
	if !ok {
		http.Error(w, "preview runtime module not found", http.StatusNotFound)
		return
	}
	root, err := findNodeModulesRoot()
	if err != nil {
		h.logger.Printf("preview.runtime node_modules: %v", err)
		http.Error(w, "preview runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	source := filepath.Join(root, entry)
	result := esbuild.Build(esbuild.BuildOptions{
		EntryPoints: []string{source},
		Bundle:      true,
		Format:      esbuild.FormatESModule,
		Platform:    esbuild.PlatformBrowser,
		Target:      esbuild.ES2020,
		External:    runtimeExternals(moduleName),
		Define: map[string]string{
			"process.env.NODE_ENV": `"development"`,
		},
		Write:    false,
		LogLevel: esbuild.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		h.logger.Printf("preview.runtime bundle %s@%s/%s: %s", moduleName, version, runtimePath, result.Errors[0].Text)
		http.Error(w, "preview runtime bundle failed", http.StatusInternalServerError)
		return
	}
	if len(result.OutputFiles) == 0 {
		http.Error(w, "preview runtime bundle empty", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(result.OutputFiles[0].Contents)
}

func (h *RuntimeHandler) servePackageRuntime(w http.ResponseWriter, moduleName, version, runtimePath string) {
	unescaped, err := url.PathUnescape(moduleName)
	if err == nil {
		moduleName = unescaped
	}
	if !safePackageName(moduleName) {
		http.Error(w, "preview dependency runtime package not found", http.StatusNotFound)
		return
	}
	if matched, ok := internaldepsResolveExact(version, installedPackageVersionCandidates(moduleName)); !ok || matched != version {
		http.Error(w, fmt.Sprintf("preview dependency runtime %s@%s is not installed", moduleName, version), http.StatusNotFound)
		return
	}
	entry, ok := packageRuntimeEntry(moduleName, runtimePath)
	if !ok {
		http.Error(w, "preview dependency runtime module not found", http.StatusNotFound)
		return
	}
	root, err := findNodeModulesRoot()
	if err != nil {
		h.logger.Printf("preview.runtime node_modules: %v", err)
		http.Error(w, "preview runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	source := filepath.Join(root, entry)
	result := esbuild.Build(esbuild.BuildOptions{
		EntryPoints: []string{source},
		Bundle:      true,
		Format:      esbuild.FormatESModule,
		Platform:    esbuild.PlatformBrowser,
		Target:      esbuild.ES2020,
		External: []string{
			"react",
			"react/*",
			"react-dom",
			"react-dom/*",
		},
		Define: map[string]string{
			"process.env.NODE_ENV": `"development"`,
		},
		Write:    false,
		LogLevel: esbuild.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		h.logger.Printf("preview.runtime bundle %s@%s/%s: %s", moduleName, version, runtimePath, result.Errors[0].Text)
		http.Error(w, "preview runtime bundle failed", http.StatusInternalServerError)
		return
	}
	if len(result.OutputFiles) == 0 {
		http.Error(w, "preview runtime bundle empty", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(result.OutputFiles[0].Contents)
}

func runtimeEntry(moduleName, runtimePath string) (string, bool) {
	if strings.Contains(runtimePath, "..") || strings.HasPrefix(runtimePath, "/") {
		return "", false
	}
	switch moduleName {
	case "react":
		switch runtimePath {
		case "index.js", "jsx-runtime.js", "jsx-dev-runtime.js":
			return filepath.Join("react", runtimePath), true
		}
	case "react-dom":
		switch runtimePath {
		case "index.js", "client.js":
			return filepath.Join("react-dom", runtimePath), true
		}
	}
	return "", false
}

func runtimeExternals(moduleName string) []string {
	if moduleName == "react-dom" {
		return []string{"react"}
	}
	return nil
}

func packageRuntimeEntry(moduleName, runtimePath string) (string, bool) {
	if !safePackageName(moduleName) || strings.Contains(runtimePath, "..") || strings.HasPrefix(runtimePath, "/") {
		return "", false
	}
	if runtimePath != "index.js" {
		return filepath.Join(filepath.FromSlash(moduleName), filepath.FromSlash(runtimePath)), true
	}
	root, err := findNodeModulesRoot()
	if err != nil {
		return "", false
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(moduleName), "package.json"))
	if err != nil {
		return "", false
	}
	var pkg struct {
		Module string `json:"module"`
		Main   string `json:"main"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return "", false
	}
	entry := strings.TrimSpace(pkg.Module)
	if entry == "" {
		entry = strings.TrimSpace(pkg.Main)
	}
	if entry == "" {
		entry = "index.js"
	}
	if strings.Contains(entry, "..") || strings.HasPrefix(entry, "/") {
		return "", false
	}
	return filepath.Join(filepath.FromSlash(moduleName), filepath.FromSlash(entry)), true
}

func safePackageName(name string) bool {
	if name == "" || strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
		return false
	}
	parts := strings.Split(name, "/")
	if strings.HasPrefix(name, "@") {
		return len(parts) == 2 && parts[0] != "@" && parts[1] != ""
	}
	return len(parts) == 1 && parts[0] != ""
}

func internaldepsResolveExact(version string, candidates []string) (string, bool) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "", false
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == version {
			return version, true
		}
	}
	return "", false
}

func findNodeModulesRoot() (string, error) {
	for _, candidate := range []string{
		"../../../ui/node_modules",
		"../ui/node_modules",
		"ui/node_modules",
		"scenarios/react-component-library/ui/node_modules",
	} {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("react-component-library ui/node_modules not found")
}
