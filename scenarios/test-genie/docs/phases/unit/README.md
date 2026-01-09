# Unit Phase

**ID**: `unit`
**Timeout**: 60 seconds (configurable)
**Optional**: No
**Requires Runtime**: No

The unit phase executes unit tests for all detected languages (Go, Node.js, Python) and collects coverage metrics. Tests run in isolation without external dependencies.

## What Gets Tested

```mermaid
graph TB
    subgraph "Unit Phase"
        DETECT[Detect Languages<br/>Go, Node, Python]

        subgraph "Go Runner"
            GO_TEST[go test ./...]
            GO_COV[Coverage Report]
            GO_REQ[REQ Tag Extraction]
        end

        subgraph "Node Runner"
            NODE_TEST[pnpm test --run]
            NODE_COV[Coverage Report]
            NODE_REQ[Vitest Reporter]
        end

        subgraph "Python Runner"
            PY_TEST[pytest]
            PY_COV[Coverage Report]
            PY_REQ[Marker Extraction]
        end
    end

    DETECT --> GO_TEST
    DETECT --> NODE_TEST
    DETECT --> PY_TEST

    GO_TEST --> GO_COV --> GO_REQ
    NODE_TEST --> NODE_COV --> NODE_REQ
    PY_TEST --> PY_COV --> PY_REQ

    GO_REQ --> RESULTS[Phase Results]
    NODE_REQ --> RESULTS
    PY_REQ --> RESULTS

    style GO_TEST fill:#00ADD8
    style NODE_TEST fill:#729B1B
    style PY_TEST fill:#3776AB
    style RESULTS fill:#c8e6c9
```

## Language Detection

| Language | Detection (walks entire scenario) | What Runs |
|----------|-----------------------------------|------------|
| Go | Any `go.mod` (e.g., `api/`, extra libs) | `go test ./...` per workspace |
| Node.js | Any `package.json` **with** `test` script | `<pkg-manager> test` per workspace |
| Python | Dir with `pyproject.toml` / `requirements.txt` **and** `test_*.py` | `pytest` (fallback `python -m unittest discover`) per workspace |

## Coverage Thresholds

| Level | Default | Behavior |
|-------|---------|----------|
| Pass | ≥80% | Phase passes |
| Warning | 70-80% | Phase passes with warning |
| Error | <70% | Phase fails |

## Requirement Tagging

Tag tests with `[REQ:ID]` to track requirement coverage:

### Go
```go
func TestCreateProject(t *testing.T) {
    t.Run("creates project [REQ:MY-PROJECT-CREATE]", func(t *testing.T) {
        // Test implementation
    })
}
```

### TypeScript (Vitest)
```typescript
describe('projectStore [REQ:MY-PROJECT-CRUD]', () => {
    it('creates project', () => { /* ... */ });
});
```

### Python
```python
@pytest.mark.requirement("MY-PROJECT-CREATE")
def test_create_project():
    pass
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All tests pass, coverage met |
| 1 | Test failures or coverage below error threshold |

## Workspace Discovery & Reporting

- The unit phase now **enumerates every unit-testable workspace** across the scenario (e.g., extra Playwright drivers, helpers outside `api/`/`ui/`).
- Before running, it emits a workspace list showing the command it will use (per language).
- Languages that aren't detected are no longer noisy in the output; missing test scripts or missing test files show up as warnings under that language instead.
- The runner continues through all languages/workspaces even if one fails, then reports the first failure alongside the full summary.

## Configuration

```json
{
  "phases": {
    "unit": {
      "timeout": 120,
      "coverageWarn": 85,
      "coverageError": 75,
      "go": {
        "race": true,
        "packages": ["./..."]
      },
      "node": {
        "framework": "vitest"
      }
    }
  }
}
```

## Related Documentation

- [Scenario Unit Testing](scenario-unit-testing.md) - Writing effective unit tests
- [Test Runners](test-runners.md) - Language-specific runner details

## See Also

- [Phases Overview](../README.md) - All phases
- [Dependencies Phase](../dependencies/README.md) - Previous phase
- [Integration Phase](../integration/README.md) - Next phase
