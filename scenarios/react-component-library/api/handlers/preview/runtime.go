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
	logger   *log.Logger
	repoRoot string
}

func NewRuntimeHandler(logger *log.Logger) *RuntimeHandler {
	return NewRuntimeHandlerAtRoot(logger, "")
}

func NewRuntimeHandlerAtRoot(logger *log.Logger, repoRoot string) *RuntimeHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &RuntimeHandler{logger: logger, repoRoot: discoverRepoRoot(repoRoot)}
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
	root, err := findVendoredNodeModulesRoot()
	if err != nil {
		h.logger.Printf("preview.runtime node_modules: %v", err)
		http.Error(w, "preview runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	source, err := filepath.Abs(filepath.Join(root, entry))
	if err != nil {
		h.logger.Printf("preview.runtime source %s@%s/%s: %v", moduleName, version, runtimePath, err)
		http.Error(w, "preview runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	result := buildRuntimeESM(source, runtimeWrapper(moduleName, runtimePath), runtimeExternals(moduleName))
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

func buildRuntimeESM(source, wrapper string, externals []string) esbuild.BuildResult {
	opts := esbuild.BuildOptions{
		Bundle:   true,
		Format:   esbuild.FormatESModule,
		Platform: esbuild.PlatformBrowser,
		Target:   esbuild.ES2020,
		External: externals,
		Define: map[string]string{
			"process.env.NODE_ENV": `"development"`,
		},
		Write:    false,
		LogLevel: esbuild.LogLevelSilent,
	}
	if runtimeNeedsReactRequireShim(externals) {
		opts.Banner = map[string]string{
			"js": `import ReactDefault, * as ReactNS from "react";
var require = (name) => {
  if (name === "react") return ReactDefault || ReactNS;
  throw Error("Unsupported preview runtime require: " + name);
};`,
		}
	}
	if wrapper != "" {
		opts.Stdin = &esbuild.StdinOptions{
			Contents:   fmt.Sprintf(wrapper, jsImportString(filepath.ToSlash(source))),
			ResolveDir: filepath.Dir(source),
			Sourcefile: "preview-runtime-wrapper.js",
			Loader:     esbuild.LoaderJS,
		}
	} else {
		opts.EntryPoints = []string{source}
	}
	return esbuild.Build(opts)
}

func runtimeWrapper(moduleName, runtimePath string) string {
	switch moduleName {
	case "react":
		switch runtimePath {
		case "index.js":
			return `import runtime from %s;
export default runtime;
export const Children = runtime.Children;
export const Component = runtime.Component;
export const Fragment = runtime.Fragment;
export const Profiler = runtime.Profiler;
export const PureComponent = runtime.PureComponent;
export const StrictMode = runtime.StrictMode;
export const Suspense = runtime.Suspense;
export const cloneElement = runtime.cloneElement;
export const createContext = runtime.createContext;
export const createElement = runtime.createElement;
export const createFactory = runtime.createFactory;
export const createRef = runtime.createRef;
export const forwardRef = runtime.forwardRef;
export const isValidElement = runtime.isValidElement;
export const lazy = runtime.lazy;
export const memo = runtime.memo;
export const startTransition = runtime.startTransition;
export const useCallback = runtime.useCallback;
export const useContext = runtime.useContext;
export const useDebugValue = runtime.useDebugValue;
export const useDeferredValue = runtime.useDeferredValue;
export const useEffect = runtime.useEffect;
export const useId = runtime.useId;
export const useImperativeHandle = runtime.useImperativeHandle;
export const useInsertionEffect = runtime.useInsertionEffect;
export const useLayoutEffect = runtime.useLayoutEffect;
export const useMemo = runtime.useMemo;
export const useReducer = runtime.useReducer;
export const useRef = runtime.useRef;
export const useState = runtime.useState;
export const useSyncExternalStore = runtime.useSyncExternalStore;
export const useTransition = runtime.useTransition;
export const version = runtime.version;
`
		case "jsx-runtime.js", "jsx-dev-runtime.js":
			return `import runtime from %s;
export default runtime;
export const Fragment = runtime.Fragment;
export const jsx = runtime.jsx;
export const jsxs = runtime.jsxs;
export const jsxDEV = runtime.jsxDEV;
`
		}
	case "react-dom":
		switch runtimePath {
		case "index.js":
			return `import runtime from %s;
export default runtime;
export const createPortal = runtime.createPortal;
export const flushSync = runtime.flushSync;
export const findDOMNode = runtime.findDOMNode;
export const hydrate = runtime.hydrate;
export const render = runtime.render;
export const unmountComponentAtNode = runtime.unmountComponentAtNode;
export const unstable_batchedUpdates = runtime.unstable_batchedUpdates;
export const version = runtime.version;
`
		case "client.js":
			return `import runtime from %s;
export default runtime;
export const createRoot = runtime.createRoot;
export const hydrateRoot = runtime.hydrateRoot;
`
		}
	}
	return ""
}

func jsImportString(s string) string {
	raw, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(raw)
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
	if matched, ok := internaldepsResolveExact(version, installedPackageVersionCandidatesAtRoot(h.repoRoot, moduleName)); !ok || matched != version {
		http.Error(w, fmt.Sprintf("preview dependency runtime %s@%s is not installed in the governed preview runtime store; populate with: %s", moduleName, version, previewDependencyPopulateCommand(moduleName, version)), http.StatusNotFound)
		return
	}
	entry, ok := packageRuntimeEntryAtRoot(h.repoRoot, moduleName, runtimePath)
	if !ok {
		http.Error(w, "preview dependency runtime module not found", http.StatusNotFound)
		return
	}
	root, err := previewRuntimePackageRoot(h.repoRoot, moduleName, version)
	if err != nil {
		h.logger.Printf("preview.runtime node_modules: %v", err)
		http.Error(w, "preview runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	source, err := filepath.Abs(filepath.Join(root, entry))
	if err != nil {
		h.logger.Printf("preview.runtime source %s@%s/%s: %v", moduleName, version, runtimePath, err)
		http.Error(w, "preview runtime unavailable", http.StatusServiceUnavailable)
		return
	}
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
		return []string{"react", "react/*"}
	}
	return nil
}

func runtimeNeedsReactRequireShim(externals []string) bool {
	for _, external := range externals {
		if external == "react" {
			return true
		}
	}
	return false
}

func packageRuntimeEntry(moduleName, runtimePath string) (string, bool) {
	root, err := findVendoredNodeModulesRoot()
	if err != nil {
		return "", false
	}
	return packageRuntimeEntryFromRoot(root, moduleName, runtimePath)
}

func packageRuntimeEntryAtRoot(repoRoot, moduleName, runtimePath string) (string, bool) {
	root, err := previewRuntimePackageRoot(repoRoot, moduleName, "")
	if err != nil {
		return "", false
	}
	return packageRuntimeEntryFromRoot(root, moduleName, runtimePath)
}

func packageRuntimeEntryFromRoot(root, moduleName, runtimePath string) (string, bool) {
	if !safePackageName(moduleName) || strings.Contains(runtimePath, "..") || strings.HasPrefix(runtimePath, "/") {
		return "", false
	}
	if runtimePath != "index.js" {
		return filepath.Join(filepath.FromSlash(moduleName), filepath.FromSlash(runtimePath)), true
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(moduleName), "package.json"))
	if err != nil {
		return "", false
	}
	var pkg struct {
		Exports json.RawMessage `json:"exports"`
		Module  string          `json:"module"`
		Main    string          `json:"main"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return "", false
	}
	entry := packageExportsEntry(pkg.Exports)
	if entry == "" {
		entry = strings.TrimSpace(pkg.Module)
	}
	if entry == "" {
		entry = strings.TrimSpace(pkg.Main)
	}
	if entry == "" {
		entry = "index.js"
	}
	entry, ok := cleanPackageRuntimeEntry(entry)
	if !ok {
		return "", false
	}
	return filepath.Join(filepath.FromSlash(moduleName), filepath.FromSlash(entry)), true
}

func packageExportsEntry(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return resolvePackageExportValue(value, true)
}

func resolvePackageExportValue(value any, root bool) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if root {
			if dot, ok := typed["."]; ok {
				if entry := resolvePackageExportValue(dot, false); entry != "" {
					return entry
				}
			}
		}
		for _, condition := range []string{"browser", "import", "module", "default"} {
			if candidate, ok := typed[condition]; ok {
				if entry := resolvePackageExportValue(candidate, false); entry != "" {
					return entry
				}
			}
		}
	}
	return ""
}

func cleanPackageRuntimeEntry(entry string) (string, bool) {
	entry = strings.TrimSpace(entry)
	entry = strings.TrimPrefix(entry, "./")
	if entry == "" || strings.Contains(entry, "..") || strings.HasPrefix(entry, "/") {
		return "", false
	}
	return entry, true
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

func findVendoredNodeModulesRoot() (string, error) {
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
