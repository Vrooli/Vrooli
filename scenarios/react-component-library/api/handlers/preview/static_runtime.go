package preview

import (
	"encoding/json"
	"fmt"
	"strings"

	internaldeps "react-component-library/internal/deps"
	internalpreview "react-component-library/internal/preview"
)

func buildImportMapJSON(b internalpreview.Bundle) (string, []string) {
	reactVersion, reactDOMVersion, warnings := resolveReactRuntimeVersions(b.Dependencies)
	imports := map[string]string{
		"react":                 runtimeURL("react", reactVersion, "", &warnings),
		"react/jsx-runtime":     runtimeURL("react", reactVersion, "jsx-runtime", &warnings),
		"react/jsx-dev-runtime": runtimeURL("react", reactVersion, "jsx-dev-runtime", &warnings),
		"react-dom":             runtimeURL("react-dom", reactDOMVersion, "", &warnings),
		"react-dom/client":      runtimeURL("react-dom", reactDOMVersion, "client", &warnings),
	}
	if version, ok := internaldeps.ResolveRangeToLatest("10.4.1", packageRuntimeCandidatesFor("@testing-library/dom")); ok {
		imports["@testing-library/dom"] = packageRuntimeURL("@testing-library/dom", version, "", &warnings)
	} else {
		warnings = append(warnings, "preview: dependency \"@testing-library/dom\" cannot be resolved from the governed preview runtime store")
	}
	// ClassMerge is a catalog foundation, so its runtime peers are part of the
	// deterministic preview substrate even when a stale or incomplete derived
	// dependency projection omits the transitive @deps header. They are already
	// governed by tools/preview-runtime/package.json; exposing them here keeps
	// specimens that import ClassMerge from failing in the browser with an
	// unresolved bare module specifier.
	for name, versionRange := range map[string]string{
		"clsx":           "^2.1.1",
		"tailwind-merge": "^2.6.1",
	} {
		if version, ok := internaldeps.ResolveRangeToLatest(versionRange, packageRuntimeCandidatesFor(name)); ok {
			imports[name] = packageRuntimeURL(name, version, "", &warnings)
		}
	}
	for _, d := range b.Dependencies {
		name := strings.TrimSpace(d.DepName)
		if name == "" || isReactRuntimeDep(name) {
			continue
		}
		// Catalog dependencies are folded into the preview bundle from their
		// governed source files by the esbuild resolver. They are not npm
		// packages and must not be reported as missing runtime-store entries.
		if strings.HasPrefix(name, "react-component-library:") || strings.HasPrefix(name, "@vrooli/react-component-library/") {
			continue
		}
		version, ok := internaldeps.ResolveRangeToLatest(d.VersionRange, packageRuntimeCandidatesFor(name))
		if !ok {
			warnings = append(warnings, fmt.Sprintf(
				"preview: dependency %q cannot be resolved from declared range %q; "+
					"package is absent from the governed preview runtime store. Populate with: %s",
				name, d.VersionRange, previewDependencyPopulateCommand(name, d.VersionRange),
			))
			continue
		}
		imports[name] = packageRuntimeURL(name, version, "", &warnings)
	}
	if _, ok := imports["lucide-react"]; !ok {
		if version, resolved := internaldeps.ResolveRangeToLatest("^0.424.0", packageRuntimeCandidatesFor("lucide-react")); resolved {
			imports["lucide-react"] = packageRuntimeURL("lucide-react", version, "", &warnings)
		}
	}
	raw, err := json.MarshalIndent(map[string]map[string]string{"imports": imports}, "", "  ")
	if err != nil {
		return `{"imports":{}}`, append(warnings, "preview: failed to encode importmap: "+err.Error())
	}
	return string(raw), warnings
}

func resolveReactRuntimeVersions(declarations []internaldeps.Declaration) (string, string, []string) {
	reactVersion := defaultReactRuntimeVersion
	reactDOMVersion := ""
	var warnings []string
	for _, d := range declarations {
		name := strings.TrimSpace(d.DepName)
		switch name {
		case "react":
			version, ok := internaldeps.ResolveRangeToLatest(d.VersionRange, reactRuntimeCandidates)
			if !ok {
				warnings = append(warnings, fmt.Sprintf("preview: cannot pin dependency %q from range %q", name, d.VersionRange))
				continue
			}
			reactVersion = version
		case "react-dom":
			version, ok := internaldeps.ResolveRangeToLatest(d.VersionRange, reactRuntimeCandidates)
			if !ok {
				warnings = append(warnings, fmt.Sprintf("preview: cannot pin dependency %q from range %q", name, d.VersionRange))
				continue
			}
			reactDOMVersion = version
		}
	}
	if reactDOMVersion == "" {
		reactDOMVersion = reactVersion
	}
	return reactVersion, reactDOMVersion, warnings
}

func isReactRuntimeDep(name string) bool {
	return name == "react" || name == "react-dom" || strings.HasPrefix(name, "react/")
}

func runtimeURL(name, version, subpath string, warnings *[]string) string {
	resolvedVersion := strings.TrimSpace(version)
	if !vendoredReactRuntimeVersions[resolvedVersion] {
		*warnings = append(*warnings, fmt.Sprintf("preview: runtime %s@%s is not vendored; using %s", name, resolvedVersion, defaultReactRuntimeVersion))
		resolvedVersion = defaultReactRuntimeVersion
	}
	if subpath != "" {
		return "/preview/runtime/" + name + "@" + resolvedVersion + "/" + subpath + ".js"
	}
	return "/preview/runtime/" + name + "@" + resolvedVersion + "/index.js"
}

func packageRuntimeURL(name, version, subpath string, warnings *[]string) string {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		*warnings = append(*warnings, "preview: dependency runtime URL missing package name or version")
		return ""
	}
	if subpath != "" {
		return "/preview/runtime/npm/" + name + "@" + version + "/" + strings.TrimPrefix(subpath, "/")
	}
	return "/preview/runtime/npm/" + name + "@" + version + "/index.js"
}
