// Package pathfilter provides a canonical source of truth for which filesystem
// paths should be skipped during code-quality scanning across Vrooli scenarios.
//
// The package is exclusion-based: it identifies directories and files that are
// universally generated, vendored, or contain runtime data and should never be
// flagged by quality scanners. Everything not excluded is assumed to be source
// code worth scanning. This design ensures new source directories (sidecar
// processes, workflow folders, etc.) are scanned automatically without updating
// this package.
//
// # Directory filtering
//
// SkipDir checks a directory base name against the canonical skip set:
//
//   - Dependencies: node_modules, vendor
//   - Build/deployment artifacts: platforms, dist, build, bin, bundle, artifacts
//   - Runtime data: data, logs, coverage, playwright-driver
//   - Language caches: __pycache__, target, obj
//   - Temporary: tmp, temp, storybook-static, venv
//   - All dot-directories (matched by "." prefix): .git, .vrooli, .cache, etc.
//
// Usage with filepath.Walk:
//
//	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
//	    if err != nil {
//	        return nil
//	    }
//	    if info.IsDir() && pathfilter.SkipDir(info.Name()) {
//	        return filepath.SkipDir
//	    }
//	    // process file ...
//	    return nil
//	})
//
// Usage with filepath.WalkDir:
//
//	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
//	    if err != nil {
//	        return nil
//	    }
//	    if d.IsDir() && pathfilter.SkipDir(d.Name()) {
//	        return filepath.SkipDir
//	    }
//	    // process file ...
//	    return nil
//	})
//
// # Source extension filtering
//
// IsSourceExt identifies file extensions that represent scannable source code.
// Combined with SkipDir it provides a complete "should I scan this file?" check.
//
// # Generated file detection
//
// IsGeneratedFile matches filenames against known generated-code patterns
// (protobuf output, codegen output, TypeScript declarations, minified assets).
//
// # Scenario-specific policy
//
// This package encodes only universally-true exclusions. Scenario-specific
// policies (e.g. restricting scans to api/ and ui/ roots, excluding test files,
// or skipping config files) remain in each scenario's own code.
package pathfilter
