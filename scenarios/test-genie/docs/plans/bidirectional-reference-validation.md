# Bidirectional Reference Validation — Implementation Plan

## Context (What Exists Today)

### Current Docs Phase Implementation

The docs phase is a core, non-optional validation phase in test-genie's testing architecture. Key files:

- **Runner**: `scenarios/test-genie/api/internal/docs/runner.go` - Core validation engine
- **Types**: `scenarios/test-genie/api/internal/docs/types.go` - Result/observation types
- **Config**: `scenarios/test-genie/api/internal/docs/config.go` - Configuration settings
- **Phase**: `scenarios/test-genie/api/internal/orchestrator/phases/phase_docs.go` - Phase orchestration

### Current Validation Checks

The docs phase currently validates **5 core checks**:

| Check | Description | Config |
|-------|-------------|--------|
| Markdown Structure | Unclosed code fences (``` or ~~~) | `markdown.enabled` |
| Mermaid Diagrams | Valid headers + bracket balance | `mermaid.enabled`, `mermaid.strict` |
| Local Links | Referenced files exist | `links.enabled` |
| External Links | HTTP/HTTPS URLs respond | `links.strict_external` |
| Absolute Paths | No OS-rooted paths | `absolute_paths.enabled`, `absolute_paths.allow` |

### What's Missing

The documentation-health skill (`scenarios/prompt-manager/skills/core/documentation-health.md`) establishes bidirectional reference formats that need validation:

1. **`// DOC:` comments in code** - Code files reference documentation
2. **`[CODE: ...]` references in docs** - Documentation references code files
3. **Manifest coverage** - All docs should be registered in `docs/manifest.json`
4. **Coverage metrics** - What percentage of exports have DOC comments?

Currently, test-genie validates that *markdown links work*, but not that *code↔docs references are valid*.

---

## Goal

Extend the docs phase to validate bidirectional code↔documentation references, ensuring:

1. `[CODE: ...]` references in docs point to valid files/functions
2. `// DOC:` comments in code point to valid documentation files
3. All docs in `docs/` are registered in `manifest.json` (optional warning)
4. Coverage metrics are reported (informational, not gating)

---

## Reference Formats to Validate

### Doc-to-Code References (`[CODE: ...]`)

Found in markdown files, links documentation to implementation:

```markdown
- [CODE: src/auth/authenticator.ts#authenticateUser]
- [CODE: api/handlers/auth.go:AuthHandler]
- [CODE: src/utils/format.ts]
```

**Patterns:**
- `[CODE: path/to/file.ext]` - File reference
- `[CODE: path/to/file.ext#functionName]` - Function reference
- `[CODE: path/to/file.ext:lineNumber]` - Line reference

**Regex:** `\[CODE:\s*([^\]]+)\]`

### Code-to-Doc References (`// DOC:`)

Found in code files, links implementation to documentation:

```typescript
// DOC: docs/reference/api-endpoints.md#user-authentication
// DOC: PRD.md#OT-P0-003
export function authenticateUser(token: string): Promise<User> {
```

**Patterns:**
- `// DOC: path/to/doc.md`
- `// DOC: path/to/doc.md#section`
- `/* DOC: path/to/doc.md */` (block comment variant)

**Regex:** `//\s*DOC:\s*([^\s]+)|/\*\s*DOC:\s*([^\s*]+)`

---

## Proposed Configuration Schema

Extend `.vrooli/testing.json`:

```json
{
  "docs": {
    "markdown": { "enabled": true },
    "mermaid": { "enabled": true, "strict": true },
    "links": {
      "enabled": true,
      "ignore": ["http://localhost:*"],
      "max_concurrency": 6,
      "timeout_ms": 5000,
      "strict_external": false
    },
    "absolute_paths": {
      "enabled": true,
      "allow": ["/api/"]
    },
    "references": {
      "enabled": true,
      "validate_code_refs": true,
      "validate_doc_refs": true,
      "code_extensions": [".ts", ".tsx", ".js", ".jsx", ".go", ".py"],
      "strict": false
    },
    "manifest": {
      "enabled": true,
      "require_all_docs_registered": false
    }
  }
}
```

**New Configuration Fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `references.enabled` | bool | true | Enable bidirectional reference validation |
| `references.validate_code_refs` | bool | true | Validate `[CODE: ...]` references in docs |
| `references.validate_doc_refs` | bool | true | Validate `// DOC:` comments in code |
| `references.code_extensions` | []string | [".ts", ".tsx", ".go", ...] | File extensions to scan for DOC comments |
| `references.strict` | bool | false | If true, broken refs fail; if false, warn |
| `manifest.enabled` | bool | true | Check manifest.json coverage |
| `manifest.require_all_docs_registered` | bool | false | Warn if docs not in manifest |

---

## Implementation Steps

### 1) Update Types (`types.go`)

Add new fields to `Summary`:

```go
type Summary struct {
    // ... existing fields ...

    // Bidirectional reference tracking
    CodeRefsFound      int // [CODE: ...] references found in docs
    CodeRefsBroken     int // [CODE: ...] references that don't resolve
    DocRefsFound       int // DOC: comments found in code
    DocRefsBroken      int // DOC: comments pointing to missing docs

    // Manifest tracking
    DocsInManifest     int // Docs registered in manifest.json
    DocsNotInManifest  int // Docs missing from manifest
}
```

### 2) Update Config (`config.go`)

Add new configuration structs:

```go
type ReferencesConfig struct {
    Enabled          *bool    `json:"enabled"`
    ValidateCodeRefs *bool    `json:"validate_code_refs"`
    ValidateDocRefs  *bool    `json:"validate_doc_refs"`
    CodeExtensions   []string `json:"code_extensions"`
    Strict           *bool    `json:"strict"`
}

type ManifestConfig struct {
    Enabled                  *bool `json:"enabled"`
    RequireAllDocsRegistered *bool `json:"require_all_docs_registered"`
}

type Settings struct {
    // ... existing fields ...
    References *ReferencesConfig `json:"references"`
    Manifest   *ManifestConfig   `json:"manifest"`
}
```

Add defaults:

```go
func defaultReferencesConfig() *ReferencesConfig {
    enabled := true
    validateCodeRefs := true
    validateDocRefs := true
    strict := false
    return &ReferencesConfig{
        Enabled:          &enabled,
        ValidateCodeRefs: &validateCodeRefs,
        ValidateDocRefs:  &validateDocRefs,
        CodeExtensions:   []string{".ts", ".tsx", ".js", ".jsx", ".go", ".py"},
        Strict:           &strict,
    }
}
```

### 3) Implement `[CODE: ...]` Validation in Runner

Add to `runner.go`:

```go
var codeRefPattern = regexp.MustCompile(`\[CODE:\s*([^\]]+)\]`)

func (r *Runner) extractCodeRefs(content string) []string {
    matches := codeRefPattern.FindAllStringSubmatch(content, -1)
    refs := make([]string, 0, len(matches))
    for _, m := range matches {
        if len(m) > 1 {
            refs = append(refs, strings.TrimSpace(m[1]))
        }
    }
    return refs
}

func (r *Runner) validateCodeRef(ref string) error {
    // Parse ref: path/file.ext or path/file.ext#func or path/file.ext:line
    filePath := ref
    if idx := strings.Index(ref, "#"); idx != -1 {
        filePath = ref[:idx]
    } else if idx := strings.Index(ref, ":"); idx != -1 {
        // Check if it's a line number (digits only after colon)
        afterColon := ref[idx+1:]
        if _, err := strconv.Atoi(afterColon); err == nil {
            filePath = ref[:idx]
        }
    }

    fullPath := filepath.Join(r.scenarioDir, filePath)
    if _, err := os.Stat(fullPath); os.IsNotExist(err) {
        return fmt.Errorf("file not found: %s", filePath)
    }
    return nil
}
```

### 4) Implement `// DOC:` Validation

Add code file scanning:

```go
var docRefPattern = regexp.MustCompile(`(?://|/\*)\s*DOC:\s*([^\s\*]+)`)

func (r *Runner) scanCodeFilesForDocRefs(ctx context.Context) ([]string, error) {
    var docRefs []string

    err := filepath.WalkDir(r.scenarioDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return err
        }

        // Skip non-code files
        ext := filepath.Ext(path)
        if !r.isCodeExtension(ext) {
            return nil
        }

        // Skip common excludes
        if r.shouldSkipPath(path) {
            return nil
        }

        content, err := os.ReadFile(path)
        if err != nil {
            return nil // Non-fatal, skip file
        }

        matches := docRefPattern.FindAllStringSubmatch(string(content), -1)
        for _, m := range matches {
            if len(m) > 1 {
                docRefs = append(docRefs, m[1])
            }
        }

        return nil
    })

    return docRefs, err
}

func (r *Runner) validateDocRef(ref string) error {
    // Strip section anchor if present
    docPath := ref
    if idx := strings.Index(ref, "#"); idx != -1 {
        docPath = ref[:idx]
    }

    // Check relative to scenario root
    fullPath := filepath.Join(r.scenarioDir, docPath)
    if _, err := os.Stat(fullPath); os.IsNotExist(err) {
        return fmt.Errorf("doc not found: %s", docPath)
    }
    return nil
}
```

### 5) Implement Manifest Coverage Check

```go
func (r *Runner) checkManifestCoverage() (registered int, missing []string, err error) {
    manifestPath := filepath.Join(r.scenarioDir, "docs", "manifest.json")

    // Read manifest
    data, err := os.ReadFile(manifestPath)
    if os.IsNotExist(err) {
        return 0, nil, nil // No manifest = skip check
    }
    if err != nil {
        return 0, nil, err
    }

    var manifest struct {
        Sections []struct {
            Documents []struct {
                Path string `json:"path"`
            } `json:"documents"`
        } `json:"sections"`
    }
    if err := json.Unmarshal(data, &manifest); err != nil {
        return 0, nil, err
    }

    // Collect registered paths
    registeredPaths := make(map[string]bool)
    for _, section := range manifest.Sections {
        for _, doc := range section.Documents {
            registeredPaths[doc.Path] = true
            registered++
        }
    }

    // Find unregistered docs
    docsDir := filepath.Join(r.scenarioDir, "docs")
    err = filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return err
        }
        if !strings.HasSuffix(path, ".md") {
            return nil
        }

        relPath, _ := filepath.Rel(docsDir, path)
        if !registeredPaths[relPath] && relPath != "manifest.json" {
            missing = append(missing, relPath)
        }
        return nil
    })

    return registered, missing, err
}
```

### 6) Integrate into Run() Method

Update the main `Run()` method in `runner.go`:

```go
func (r *Runner) Run(ctx context.Context) *RunResult {
    // ... existing checks ...

    // New: Validate [CODE: ...] references in docs
    if r.settings.References != nil && *r.settings.References.Enabled {
        if *r.settings.References.ValidateCodeRefs {
            for _, docFile := range markdownFiles {
                content, _ := os.ReadFile(docFile)
                refs := r.extractCodeRefs(string(content))
                for _, ref := range refs {
                    r.summary.CodeRefsFound++
                    if err := r.validateCodeRef(ref); err != nil {
                        r.summary.CodeRefsBroken++
                        r.addObservation(...)
                    }
                }
            }
        }

        if *r.settings.References.ValidateDocRefs {
            docRefs, _ := r.scanCodeFilesForDocRefs(ctx)
            for _, ref := range docRefs {
                r.summary.DocRefsFound++
                if err := r.validateDocRef(ref); err != nil {
                    r.summary.DocRefsBroken++
                    r.addObservation(...)
                }
            }
        }
    }

    // New: Check manifest coverage
    if r.settings.Manifest != nil && *r.settings.Manifest.Enabled {
        registered, missing, _ := r.checkManifestCoverage()
        r.summary.DocsInManifest = registered
        r.summary.DocsNotInManifest = len(missing)
        if *r.settings.Manifest.RequireAllDocsRegistered && len(missing) > 0 {
            r.addObservation(...)
        }
    }

    // ... continue with result aggregation ...
}
```

### 7) Update Summary Output

Extend the summary output to include new metrics:

```go
func (r *Runner) buildSummaryObservation() Observation {
    // Include new fields in summary
    details := fmt.Sprintf(
        "Files: %d, External links: %d, Local links: %d, Broken: %d, "+
        "Code refs: %d (broken: %d), Doc refs: %d (broken: %d), "+
        "Manifest coverage: %d/%d",
        r.summary.FilesChecked,
        r.summary.ExternalLinks,
        r.summary.LocalLinks,
        r.summary.BrokenLinks,
        r.summary.CodeRefsFound, r.summary.CodeRefsBroken,
        r.summary.DocRefsFound, r.summary.DocRefsBroken,
        r.summary.DocsInManifest, r.summary.DocsInManifest+r.summary.DocsNotInManifest,
    )
    // ...
}
```

---

## Testing Strategy

### Unit Tests (`runner_test.go`)

Add tests for:

1. **`[CODE: ...]` extraction**
   - Test regex captures various formats
   - Test path with `#function` suffix
   - Test path with `:line` suffix

2. **`[CODE: ...]` validation**
   - Valid file path → pass
   - Missing file → fail
   - Path with valid function → pass (file exists check only)

3. **`// DOC:` extraction**
   - Test single-line comment format
   - Test block comment format
   - Test with/without section anchors

4. **`// DOC:` validation**
   - Valid doc path → pass
   - Missing doc → fail
   - Path with `#section` → pass (file exists check only)

5. **Manifest coverage**
   - All docs registered → pass
   - Missing docs → warning
   - No manifest → skip check

6. **Configuration**
   - References disabled → skip validation
   - Strict mode → failures instead of warnings

### Integration Test

Create a test fixture with:
- Docs containing valid and broken `[CODE: ...]` references
- Code files with valid and broken `// DOC:` comments
- Manifest with some docs missing

---

## Documentation Updates

### Update Phase README

File: `scenarios/test-genie/docs/phases/docs/README.md`

Add section:

```markdown
## Bidirectional Reference Validation

The docs phase validates bidirectional references between code and documentation:

### Code References in Docs (`[CODE: ...]`)

Documentation can reference code files using the `[CODE: ...]` syntax:

- `[CODE: src/auth/token.ts]` - Reference to a file
- `[CODE: src/auth/token.ts#validateToken]` - Reference to a function
- `[CODE: api/main.go:142]` - Reference to a line

The docs phase validates that referenced files exist.

### Doc References in Code (`// DOC:`)

Code files can reference documentation using `// DOC:` comments:

```typescript
// DOC: docs/reference/api-endpoints.md#authentication
export function authenticate() { ... }
```

The docs phase validates that referenced docs exist.

### Configuration

```json
{
  "docs": {
    "references": {
      "enabled": true,
      "validate_code_refs": true,
      "validate_doc_refs": true,
      "strict": false
    }
  }
}
```
```

---

## Validation Commands (Post-Implementation)

```bash
# Build test-genie
cd scenarios/test-genie && make build

# Run docs phase with reference validation
test-genie execute test-genie --phases docs

# Test against a scenario with known references
test-genie execute landing-page-business-suite --phases docs --fail-fast

# Verify configuration override
echo '{"docs":{"references":{"strict":true}}}' > /tmp/test.json
test-genie execute test-genie --phases docs --config /tmp/test.json
```

---

## Deliverables

1. Extended `runner.go` with `[CODE: ...]` and `// DOC:` validation
2. Updated `types.go` with new summary fields
3. Updated `config.go` with new configuration options
4. Unit tests for all new validation logic
5. Updated docs phase README with reference validation section
6. Updated manifest.json to include new docs page

---

## Future Considerations (Out of Scope)

- **Function existence validation**: Currently we only check if the file exists, not if the function/symbol exists. This would require language-specific parsing.
- **Line number validation**: We don't validate that line numbers are within file bounds.
- **Automatic reference generation**: Could generate `[CODE: ...]` links from imports or exports.
- **Coverage metrics**: Track what percentage of exports have `// DOC:` comments.
