# PRD Control Tower E2E Test Playbooks

This directory contains end-to-end test playbooks for validating PRD Control Tower functionality using browser-automation-studio.

## Directory Structure

```
test/playbooks/
├── capabilities/          # Organized by operational target
│   ├── 01-catalog/       # Catalog view and navigation
│   │   └── e2e/
│   ├── 02-drafts/        # Draft lifecycle management
│   │   └── e2e/
│   ├── 03-backlog/       # Backlog intake and conversion
│   │   └── e2e/
│   ├── 04-requirements/  # Requirements registry and linkage
│   │   └── e2e/
│   └── 05-ai/            # AI-assisted generation
│       └── e2e/
└── journeys/             # User journey workflows
    └── 01-new-user/      # New user onboarding flows
```

## Test Coverage

### Capabilities (E2E)

#### 01-catalog (PCT-FUNC-001)
- **catalog-view-and-navigate.json** - Verifies catalog displays all scenarios/resources and navigation works

#### 02-drafts (PCT-FUNC-002)
- **draft-create-edit-publish.json** - Complete draft lifecycle: create, edit, save, validate, publish

#### 03-backlog (PCT-FUNC-003)
- **backlog-intake-and-convert.json** - Backlog intake workflow and conversion to drafts

#### 04-requirements (PCT-FUNC-004)
- **requirements-view-and-linkage.json** - Requirements registry display and operational target linkage

#### 05-ai (PCT-FUNC-005)
- **ai-section-generation.json** - AI-powered PRD content generation

## Running Tests

### Individual Playbook
```bash
vrooli scenario test prd-control-tower e2e --playbook test/playbooks/capabilities/01-catalog/e2e/catalog-view-and-navigate.json
```

### All E2E Tests
```bash
vrooli scenario test prd-control-tower e2e
```

### Full Test Suite
```bash
vrooli scenario test prd-control-tower all
```

## Selector Manifest

UI selectors are defined in `ui/src/consts/selectors.manifest.json` and referenced in playbooks using `@selector/path.to.element` syntax.

Example:
```json
{
  "selector": "@selector/catalog.card"
}
```

Maps to:
```json
{
  "selectors": {
    "catalog": {
      "card": "[role='link'][data-testid='catalog-card']"
    }
  }
}
```

## Adding New Tests

1. Create playbook JSON in appropriate `capabilities/<target>/e2e/` directory
2. Reference selectors from manifest using `@selector/` prefix
3. Link requirements in metadata: `"requirements": ["PCT-FUNC-XXX"]`
4. Update this README with test description
5. Rebuild registry: `test-genie registry build` (run from scenario directory)

## Requirements Linkage

Each playbook includes `requirements` array in metadata to link E2E tests to requirement IDs. These links enable multi-layer validation tracking in completeness scoring.

Example:
```json
{
  "metadata": {
    "requirements": ["PCT-FUNC-001", "PCT-CATALOG-ENUMERATE", "PCT-CATALOG-STATUS"]
  }
}
```

## Test Data Management

- Use `Date.now()` for unique identifiers to avoid conflicts
- Set `"reset": "full"` in metadata for clean test runs
- Store generated values with `"storeResult": "variableName"`
- Reference stored values with `{{variableName}}` syntax

## Troubleshooting

### Selector Not Found
1. Check selector manifest at `ui/src/consts/selectors.manifest.json`
2. Verify selector syntax: `@selector/category.elementName`
3. Inspect UI element in browser devtools for correct data-testid attribute

### Playbook Fails
1. Check scenario is running: `vrooli scenario status prd-control-tower`
2. Verify browserless is available: `vrooli resource status browserless`
3. Review test output logs for specific assertion failures
4. Run with verbose mode: `vrooli scenario test prd-control-tower e2e --verbose`

## Related Documentation

- [Phased Testing Architecture](/docs/testing/architecture/PHASED_TESTING.md)
- [Vrooli Ascension](/scenarios/browser-automation-studio/README.md)
- [Selector Manifest Schema](/schemas/selector-manifest.schema.json)
- [Requirements Tracking](/scenarios/prd-control-tower/requirements/README.md)
