# Integration Guide - Scenario Dependency Analyzer

This guide shows how to integrate the Scenario Dependency Analyzer into your scenarios, scripts, and workflows.

---

## Quick Start

### From Shell Scripts

```bash
#!/bin/bash

# Get the analyzer API URL
API_URL=$(vrooli scenario port scenario-dependency-analyzer API_PORT)
API_URL="http://localhost:${API_URL}"

# Analyze a scenario
curl -s "${API_URL}/api/v1/analyze/my-scenario" | jq .
```

### From CLI

```bash
# Ensure scenario-dependency-analyzer is in PATH
which scenario-dependency-analyzer

# Use the CLI
scenario-dependency-analyzer analyze my-scenario --json
scenario-dependency-analyzer dag export my-scenario --output deps.json
```

### From Go Code

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type DeploymentReport struct {
    Scenario     string                   `json:"scenario"`
    Dependencies []DependencyNode         `json:"dependencies"`
    MetadataGaps *DeploymentMetadataGaps  `json:"metadata_gaps,omitempty"`
}

func getDeploymentReport(scenario string) (*DeploymentReport, error) {
    // Get API port
    apiPort := getAPIPort("scenario-dependency-analyzer")
    url := fmt.Sprintf("http://localhost:%s/api/v1/scenarios/%s/deployment", apiPort, scenario)

    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var report DeploymentReport
    if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
        return nil, err
    }

    return &report, nil
}
```

---

## Integration Patterns

### Pattern 1: Pre-Deployment Validation

Use the analyzer to validate deployment readiness before packaging.

**Scenario**: `deployment-manager`

```bash
#!/bin/bash
# In deployment-manager/scripts/pre-package.sh

SCENARIO="$1"

# Get deployment report
REPORT=$(scenario-dependency-analyzer deployment "$SCENARIO" --json)

# Check for blockers
BLOCKERS=$(echo "$REPORT" | jq -r '.aggregates.desktop.blocking_dependencies | length')

if [[ "$BLOCKERS" -gt 0 ]]; then
    echo "❌ Deployment blocked for desktop tier"
    echo "$REPORT" | jq '.aggregates.desktop.blocking_dependencies'
    exit 1
fi

# Check metadata gaps
GAPS=$(echo "$REPORT" | jq -r '.metadata_gaps.total_gaps // 0')

if [[ "$GAPS" -gt 0 ]]; then
    echo "⚠️  Warning: $GAPS metadata gaps detected"
    echo "$REPORT" | jq '.metadata_gaps.recommendations'
fi

echo "✅ Deployment validation passed"
```

### Pattern 2: Recursive Bundle Generation

Export the full recursive DAG for packaging.

**Scenario**: `scenario-to-desktop`, `scenario-to-docker`

```bash
#!/bin/bash
# In scenario-to-desktop/scripts/build-bundle.sh

SCENARIO="$1"
BUNDLE_DIR="./bundles/$SCENARIO"

# Export recursive DAG
scenario-dependency-analyzer dag export "$SCENARIO" \
    --recursive \
    --output "$BUNDLE_DIR/dependencies.json"

# Parse and download dependencies
jq -r '.dag[] | select(.type == "resource") | .name' \
    "$BUNDLE_DIR/dependencies.json" \
    | while read -r resource; do
        echo "Packaging resource: $resource"
        package_resource "$resource" "$BUNDLE_DIR/resources/"
    done

# Package nested scenarios
jq -r '.dag[] | select(.type == "scenario") | .name' \
    "$BUNDLE_DIR/dependencies.json" \
    | while read -r nested_scenario; do
        echo "Including scenario: $nested_scenario"
        # Recursively call this script for nested scenarios
        ./build-bundle.sh "$nested_scenario"
    done
```

### Pattern 3: CI/CD Drift Detection

Detect dependency drift in CI pipelines.

**Scenario**: CI/CD workflows

```yaml
# .github/workflows/dependency-check.yml
name: Dependency Drift Check

on: [pull_request]

jobs:
  check-drift:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Start Dependency Analyzer
        run: vrooli scenario run scenario-dependency-analyzer

      - name: Scan for drift
        run: |
          SCENARIOS=$(find scenarios -maxdepth 1 -type d -name "scenario-*")

          for scenario in $SCENARIOS; do
            name=$(basename "$scenario")

            # Scan and check for drift
            RESULT=$(scenario-dependency-analyzer scan "$name" --json)

            MISSING_RESOURCES=$(echo "$RESULT" | jq -r '.analysis.resource_diff.missing | length')
            MISSING_SCENARIOS=$(echo "$RESULT" | jq -r '.analysis.scenario_diff.missing | length')

            if [[ "$MISSING_RESOURCES" -gt 0 || "$MISSING_SCENARIOS" -gt 0 ]]; then
              echo "::error::Dependency drift detected in $name"
              echo "$RESULT" | jq '.analysis.resource_diff, .analysis.scenario_diff'
              exit 1
            fi
          done
```

### Pattern 4: Secrets Strategy Detection

Integrate with secrets-manager to flag secret requirements.

**Scenario**: `secrets-manager`

```bash
#!/bin/bash
# In secrets-manager/scripts/analyze-scenario.sh

SCENARIO="$1"

# Get deployment report
REPORT=$(scenario-dependency-analyzer deployment "$SCENARIO" --json)

# Extract resources that likely need secrets
echo "$REPORT" | jq -r '.dependencies[] | select(.type == "resource") | .name' \
    | while read -r resource; do
        case "$resource" in
            postgres|mysql|mongodb)
                echo "Resource $resource requires: database credentials"
                ;;
            stripe|paypal)
                echo "Resource $resource requires: API keys"
                ;;
            smtp|sendgrid)
                echo "Resource $resource requires: email credentials"
                ;;
        esac
    done

# Check for tier-specific secrets
jq -r '.dependencies[] | select(.tier_support.saas) | .name' <<<"$REPORT" \
    | while read -r dep; do
        echo "Dependency $dep needs SaaS secret strategy"
    done
```

### Pattern 5: Impact Analysis Before Removal

Check impact before removing a dependency.

**Scenario**: Any refactoring workflow

```bash
#!/bin/bash
# Before removing a resource/scenario

DEPENDENCY="$1"

# Analyze impact
IMPACT=$(scenario-dependency-analyzer impact "$DEPENDENCY" --json)

SEVERITY=$(echo "$IMPACT" | jq -r '.severity')
AFFECTED=$(echo "$IMPACT" | jq -r '.total_affected')

if [[ "$SEVERITY" == "critical" ]]; then
    echo "❌ Cannot remove $DEPENDENCY - critical dependency"
    echo "Affects $AFFECTED scenarios:"
    echo "$IMPACT" | jq '.direct_dependents[].scenario_name'
    exit 1
fi

echo "⚠️  Removing $DEPENDENCY will affect $AFFECTED scenarios"
echo "$IMPACT" | jq '.recommendations'
```

---

## Advanced Integration

### Programmatic API Usage

#### From Node.js/TypeScript

```typescript
import axios from 'axios';

interface DeploymentReport {
  scenario: string;
  dependencies: DependencyNode[];
  metadata_gaps?: MetadataGaps;
}

async function getDeploymentReport(scenario: string): Promise<DeploymentReport> {
  const apiPort = process.env.SCENARIO_DEPENDENCY_ANALYZER_API_PORT || '20400';
  const url = `http://localhost:${apiPort}/api/v1/scenarios/${scenario}/deployment`;

  const { data } = await axios.get<DeploymentReport>(url);
  return data;
}

// Usage in deployment-manager UI
async function validateDeployment(scenario: string, tier: string) {
  const report = await getDeploymentReport(scenario);

  const tierData = report.aggregates?.[tier];
  if (!tierData) {
    throw new Error(`No tier data for ${tier}`);
  }

  if (tierData.blocking_dependencies && tierData.blocking_dependencies.length > 0) {
    throw new Error(`Deployment blocked: ${tierData.blocking_dependencies.join(', ')}`);
  }

  return {
    ready: true,
    requirements: tierData.estimated_requirements,
    warnings: report.metadata_gaps?.recommendations || []
  };
}
```

#### From Python

```python
import requests
import os

class DependencyAnalyzer:
    def __init__(self):
        api_port = os.getenv('SCENARIO_DEPENDENCY_ANALYZER_API_PORT', '20400')
        self.base_url = f"http://localhost:{api_port}/api/v1"

    def get_deployment_report(self, scenario: str) -> dict:
        response = requests.get(f"{self.base_url}/scenarios/{scenario}/deployment")
        response.raise_for_status()
        return response.json()

    def export_dag(self, scenario: str, recursive: bool = True) -> dict:
        params = {'recursive': str(recursive).lower(), 'format': 'json'}
        response = requests.get(
            f"{self.base_url}/scenarios/{scenario}/dag/export",
            params=params
        )
        response.raise_for_status()
        return response.json()

    def check_metadata_gaps(self, scenario: str) -> list:
        report = self.get_deployment_report(scenario)
        gaps = report.get('metadata_gaps', {})
        return gaps.get('recommendations', [])

# Usage
analyzer = DependencyAnalyzer()
report = analyzer.get_deployment_report('my-scenario')

if report['metadata_gaps']['total_gaps'] > 0:
    print("Metadata gaps found:")
    for recommendation in report['metadata_gaps']['recommendations']:
        print(f"  - {recommendation}")
```

---

## Event-Driven Integration (Future)

When webhook support is added, you can listen for events:

```javascript
// Express.js webhook receiver
app.post('/webhooks/dependency-analyzer', (req, res) => {
  const { event, payload } = req.body;

  switch (event) {
    case 'dependency.drift.detected':
      notifyTeam(`Drift detected in ${payload.scenario}`);
      break;

    case 'metadata.gap.found':
      createJiraTicket({
        title: `Metadata gaps in ${payload.scenario}`,
        description: payload.recommendations.join('\n')
      });
      break;

    case 'scenario.analyzed':
      updateDashboard(payload);
      break;
  }

  res.sendStatus(200);
});
```

---

## Best Practices

### 1. Cache API Responses

The deployment report is expensive to generate. Cache it when possible:

```bash
# Cache for 5 minutes
CACHE_FILE="/tmp/dep-report-${SCENARIO}.json"
CACHE_AGE=$((5 * 60)) # 5 minutes

if [[ -f "$CACHE_FILE" ]]; then
    AGE=$(($(date +%s) - $(stat -c %Y "$CACHE_FILE")))
    if [[ $AGE -lt $CACHE_AGE ]]; then
        cat "$CACHE_FILE"
        exit 0
    fi
fi

# Fetch fresh data
scenario-dependency-analyzer deployment "$SCENARIO" --json > "$CACHE_FILE"
cat "$CACHE_FILE"
```

### 2. Validate Before Committing

Add a git pre-commit hook:

```bash
#!/bin/bash
# .git/hooks/pre-commit

changed_scenarios=$(git diff --cached --name-only | grep 'scenarios/.*/\.vrooli/service\.json' | cut -d/ -f2 | sort -u)

for scenario in $changed_scenarios; do
    echo "Validating $scenario..."

    # Scan for drift
    RESULT=$(scenario-dependency-analyzer scan "$scenario" --json 2>&1)

    if [[ $? -ne 0 ]]; then
        echo "❌ Scan failed for $scenario"
        exit 1
    fi

    # Check if there's drift
    DRIFT=$(echo "$RESULT" | jq -r '.analysis.resource_diff.missing | length')

    if [[ "$DRIFT" -gt 0 ]]; then
        echo "⚠️  Warning: Undeclared dependencies detected"
        echo "Run: scenario-dependency-analyzer scan $scenario --apply"
    fi
done
```

### 3. Document Dependencies

Auto-generate dependency documentation:

```bash
#!/bin/bash
# scripts/generate-dep-docs.sh

SCENARIO="$1"
OUTPUT="docs/dependencies/${SCENARIO}.md"

cat > "$OUTPUT" << EOF
# Dependencies for $SCENARIO

Last updated: $(date)

## Direct Dependencies

EOF

scenario-dependency-analyzer deployment "$SCENARIO" --json \
    | jq -r '.dependencies[] | "- **\(.type)**: \(.name)\(if .resource_type then " (\(.resource_type))" else "" end)"' \
    >> "$OUTPUT"

cat >> "$OUTPUT" << EOF

## Deployment Readiness

EOF

scenario-dependency-analyzer deployment "$SCENARIO" --json \
    | jq -r '.aggregates | to_entries[] | "### \(.key)\n\n- Fitness: \(.value.fitness_score * 100)%\n- Dependencies: \(.value.dependency_count)\n"' \
    >> "$OUTPUT"
```

### 4. Monitor for Gaps

Set up periodic scans to catch metadata gaps:

```bash
#!/bin/bash
# cron: 0 2 * * * /path/to/monitor-gaps.sh

SCENARIOS=$(scenario-dependency-analyzer list --json | jq -r '.[].name')

GAPS_FOUND=0

for scenario in $SCENARIOS; do
    REPORT=$(scenario-dependency-analyzer deployment "$scenario" --json)
    GAPS=$(echo "$REPORT" | jq -r '.metadata_gaps.total_gaps // 0')

    if [[ "$GAPS" -gt 0 ]]; then
        GAPS_FOUND=$((GAPS_FOUND + 1))
        echo "Scenario $scenario has $GAPS metadata gaps"
        echo "$REPORT" | jq '.metadata_gaps.recommendations'
    fi
done

if [[ "$GAPS_FOUND" -gt 0 ]]; then
    # Send notification
    send_slack_message "🔍 $GAPS_FOUND scenarios have metadata gaps"
fi
```

---

## Troubleshooting

### Analyzer Not Running

```bash
# Check if running
vrooli scenario status scenario-dependency-analyzer

# Start if needed
vrooli scenario run scenario-dependency-analyzer

# Check logs
vrooli scenario logs scenario-dependency-analyzer
```

### Port Discovery Issues

```bash
# Get the port manually
API_PORT=$(vrooli scenario port scenario-dependency-analyzer API_PORT)
echo "API available at: http://localhost:$API_PORT"

# Test connectivity
curl -s "http://localhost:$API_PORT/health" | jq .
```

### Stale Cache

```bash
# Clear analysis cache
rm -f /tmp/dependency-analysis-*

# Force re-analysis
scenario-dependency-analyzer analyze my-scenario --force
```

---

## Examples Repository

See `scenarios/scenario-dependency-analyzer/examples/` for:
- Full integration examples
- Sample scripts
- CI/CD workflows
- Test scenarios

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/Vrooli/Vrooli/issues
- Documentation: `/docs/README.md`
- CLI Help: `scenario-dependency-analyzer --help`
