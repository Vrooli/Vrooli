# Playbook-Requirement Linkage Validation

## Overview

The playbook linkage validator ensures **every workflow playbook is tied to an existing requirement**, preventing orphaned test files and maintaining bidirectional traceability.

This validation runs automatically during the **structure phase** of scenario tests.

## What It Detects

The validator catches **three critical failure cases**:

### ❌ **CASE 1: Requirements Reference Missing Playbook Files**

**Problem**: A requirement declares an automation validation pointing to a playbook file that doesn't exist.

**Example**:
```json
// requirements/ui/user-flows.json
{
  "requirements": [{
    "id": "BAS-PROJECT-CREATE",
    "validation": [{
      "type": "automation",
      "ref": "test/playbooks/ui/projects/project-create.json",  // ❌ File doesn't exist!
      "phase": "integration"
    }]
  }]
}
```

**Error Output**:
```
❌ CASE 1: Requirements Reference Missing Playbook Files

   ❌ Requirement: BAS-PROJECT-CREATE
      File: test/playbooks/ui/projects/project-create.json
      → File does not exist
```

**Fix**:
- Create the missing playbook file, OR
- Remove the automation validation entry from the requirement

---

### ❌ **CASE 2: Playbooks Reference Non-Existent Requirements**

**Problem**: A playbook declares a requirement ID in its metadata that doesn't exist in any `requirements/*.json` file.

**Example**:
```json
// test/playbooks/ui/projects/project-search.json
{
  "metadata": {
    "requirement": "BAS-PROJECT-SEARCH-FILTER",  // ❌ Requirement doesn't exist!
    "description": "Tests search filtering",
    "version": 1
  },
  "nodes": [...],
  "edges": [...]
}
```

**Error Output**:
```
❌ CASE 2: Playbooks Reference Non-Existent Requirements

   ❌ Playbook: test/playbooks/ui/projects/project-search.json
      Declares: BAS-PROJECT-SEARCH-FILTER (not found in requirements/)
```

**Fix**:
- Create a requirement entry in the appropriate `requirements/*.json` file, OR
- Update `metadata.requirement` to reference an existing requirement, OR
- Delete the playbook if it's no longer needed

---

### ❌ **CASE 3: Playbooks Missing Requirement Metadata**

**Problem**: A playbook file exists but doesn't declare which requirement it validates.

**Example**:
```json
// test/playbooks/ui/workflows/workflow-test.json
{
  "nodes": [...],
  "edges": [...]
  // ❌ No metadata section!
}
```

**Error Output**:
```
❌ CASE 3: Playbooks Missing Requirement Metadata

   ❌ Playbook: test/playbooks/ui/workflows/workflow-test.json
      Issue: Missing .metadata section entirely
```

**Fix**:
- Add metadata section with requirement field:
  ```json
  {
    "metadata": {
      "requirement": "BAS-WORKFLOW-TEST",
      "description": "Tests workflow loading",
      "version": 1
    },
    "nodes": [...],
    "edges": [...]
  }
  ```

---

### ⚠️ **Orphaned Playbooks** (Warning, not failure)

**Problem**: A playbook declares a valid requirement, but that requirement doesn't reference the playbook back.

**Example**:
```json
// test/playbooks/ui/builder/builder-add-node.json
{
  "metadata": {
    "requirement": "BAS-BUILDER-ADD-NODE"  // ✅ Requirement exists
  }
}

// requirements/workflow-builder/core.json
{
  "requirements": [{
    "id": "BAS-BUILDER-ADD-NODE",
    "validation": []  // ⚠️ Doesn't reference the playbook!
  }]
}
```

**Warning Output**:
```
⚠️  Playbooks Not Referenced by Their Declared Requirements

   ⚠️  Playbook: test/playbooks/ui/builder/builder-add-node.json
      Declares: BAS-BUILDER-ADD-NODE (requirement exists but doesn't link back)
```

**Fix**:
- Add automation validation entry to the requirement:
  ```json
  {
    "id": "BAS-BUILDER-ADD-NODE",
    "validation": [{
      "type": "automation",
      "ref": "test/playbooks/ui/builder/builder-add-node.json",
      "phase": "integration",
      "status": "implemented"
    }]
  }
  ```

---

## How It Works

### Automatic Execution

The validator runs during the **structure phase**:

```bash
cd scenarios/browser-automation-studio
bash test/phases/test-structure.sh
```

### Manual Execution

Run standalone validation:

```bash
bash scripts/scenarios/testing/shell/validate-playbook-linkage.sh \
  scenarios/browser-automation-studio
```

### Validation Process

1. **Scan Requirements**: Reads all `requirements/**/*.json` files via `requirements/index.json` imports
2. **Extract References**: Finds all `"type": "automation"` validation entries
3. **Scan Playbooks**: Finds all `test/playbooks/**/*.json` files (excluding `__*` test files)
4. **Cross-Validate**: Checks bidirectional linkage between playbooks and requirements

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All playbooks properly linked |
| 1 | Orphaned playbooks found (not referenced) |
| 2 | Requirements reference missing playbook files |
| 3 | Playbooks reference non-existent requirements |
| 4 | Requirements directory not found |

---

## Best Practices

### Creating New Playbooks

**Always follow this workflow**:

1. **Create requirement entry first**:
   ```json
   // requirements/ui/user-flows.json
   {
     "id": "BAS-NEW-FEATURE",
     "category": "ui",
     "title": "New feature description",
     "status": "in_progress",
     "criticality": "P0",
     "validation": [{
       "type": "automation",
       "ref": "test/playbooks/ui/new-feature.json",
       "phase": "integration",
       "status": "pending"
     }]
   }
   ```

2. **Create playbook with matching metadata**:
   ```json
   // test/playbooks/ui/new-feature.json
   {
     "metadata": {
       "requirement": "BAS-NEW-FEATURE",  // ✅ Matches requirement ID
       "description": "Tests new feature user flow",
       "version": 1
     },
     "nodes": [...],
     "edges": [...]
   }
   ```

3. **Run validation** before committing:
   ```bash
   bash test/phases/test-structure.sh
   ```

### Requirement File Organization

| File | Purpose | Example IDs |
|------|---------|-------------|
| `ui/user-flows.json` | UI interaction flows | `BAS-PROJECT-*`, `BAS-WORKFLOW-*` |
| `workflow-builder/core.json` | Canvas interactions | `BAS-BUILDER-*` |
| `ai/generation.json` | AI features | `BAS-AI-*` |
| `execution/telemetry.json` | Execution monitoring | `BAS-EXEC-*` |
| `persistence/version-history.json` | Data persistence | `BAS-VERSION-*` |

### Naming Conventions

**Requirement IDs**:
- Format: `BAS-{CATEGORY}-{ACTION}`
- Examples: `BAS-PROJECT-CREATE`, `BAS-WORKFLOW-SELECT`, `BAS-EXEC-HISTORY-VIEW`

**Playbook Paths**:
- Format: `test/playbooks/{category}/{feature}-{action}.json`
- Examples:
  - `test/playbooks/ui/projects/project-create.json`
  - `test/playbooks/ui/workflows/workflow-select-from-list.json`
  - `test/playbooks/ui/executions/execution-history-view.json`

---

## Troubleshooting

### "Found 0 unique requirement IDs"

**Cause**: `requirements/index.json` not found or malformed

**Fix**:
```json
// requirements/index.json
{
  "imports": [
    "ui/user-flows.json",
    "workflow-builder/core.json",
    "execution/telemetry.json"
  ]
}
```

### "jq not available"

**Cause**: `jq` not installed

**Fix**:
```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq
```

### Validation Passes But Playbooks Don't Run

**Cause**: Playbook might be linked but marked `"status": "pending"` in requirement

**Check**:
```bash
# Find all pending validations
jq '.requirements[].validation[] | select(.status == "pending")' requirements/**/*.json
```

**Fix**: Update status to `"implemented"` when playbook is ready

---

## Integration with CI/CD

### Pre-Commit Hook

Prevent orphaned playbooks from being committed:

```bash
#!/bin/bash
# .git/hooks/pre-commit

if git diff --cached --name-only | grep -q "^scenarios/.*/test/playbooks/.*\.json$"; then
  echo "🔍 Validating playbook-requirement linkage..."

  for scenario in scenarios/*/; do
    if [ -d "$scenario/test/playbooks" ]; then
      bash scripts/scenarios/testing/shell/validate-playbook-linkage.sh "$scenario" || {
        echo "❌ Playbook validation failed for $scenario"
        echo "💡 Run: bash test/phases/test-structure.sh"
        exit 1
      }
    fi
  done
fi
```

### GitHub Actions

```yaml
- name: Validate Playbook Linkage
  run: |
    for scenario in scenarios/*/; do
      if [ -d "$scenario/test/playbooks" ]; then
        bash scripts/scenarios/testing/shell/validate-playbook-linkage.sh "$scenario"
      fi
    done
```

---

## Related Documentation

- [Requirements Tracking System](/scripts/requirements/lib/README.md)
- [Integration Test Guide](/docs/testing/integration.md)
- [Playbook Writing Guide](/scenarios/browser-automation-studio/test/playbooks/README.md)

---

## Example Output

### ✅ Success Case

```
🔍 Validating Playbook-Requirement Linkage for browser-automation-studio
==============================================================

📋 Step 1: Scanning requirements registry...

   Found 33 unique requirement IDs
   Found 25 playbook references in requirements

📂 Step 2: Checking for missing playbook files...

✅ All referenced playbook files exist

📝 Step 3: Scanning playbook files...

   Found 25 playbook file(s)

🔗 Step 4: Validating playbook metadata...

==============================================================
📊 Validation Results
==============================================================

==============================================================

✅ All playbooks are properly linked to requirements!

   📊 Summary:
      • Total playbooks: 25
      • Total requirements: 33
      • Playbook references in requirements: 25
      • All have valid requirement metadata
      • All requirements reference existing files
```

### ❌ Failure Case

```
🔍 Validating Playbook-Requirement Linkage for browser-automation-studio
==============================================================

...

❌ CASE 2: Playbooks Reference Non-Existent Requirements

   ❌ Playbook: test/playbooks/ui/builder/builder-add-node.json
      Declares: BAS-BUILDER-ADD-NODE (not found in requirements/)

   Action Required:
   • Create requirement entry in appropriate requirements/*.json file

==============================================================

❌ Validation Failed: 1 issue(s) found

   📊 Summary:
      • Total playbooks: 25
      • Total requirements: 32
      • Invalid requirement refs: 1
```

---

## FAQ

**Q: Can I skip this validation for a specific scenario?**

A: Not recommended, but you can configure it in `.vrooli/testing.json`:
```json
{
  "structure": {
    "validations": {
      "playbook_linkage": false
    }
  }
}
```

**Q: What if I have experimental playbooks?**

A: Prefix with `__` (e.g., `__test-experiment.json`) - these are automatically excluded

**Q: Can requirements have multiple playbooks?**

A: Yes! Add multiple automation validation entries:
```json
{
  "id": "BAS-PROJECT-CRUD",
  "validation": [
    {"type": "automation", "ref": "test/playbooks/ui/projects/project-create.json"},
    {"type": "automation", "ref": "test/playbooks/ui/projects/project-edit.json"},
    {"type": "automation", "ref": "test/playbooks/ui/projects/project-delete.json"}
  ]
}
```

**Q: Can playbooks reference multiple requirements?**

A: No. Each playbook should validate a single requirement. If needed, create composite requirements with children.
