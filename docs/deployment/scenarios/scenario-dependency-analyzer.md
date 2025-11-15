# Scenario: scenario-dependency-analyzer

## Current Role

Scans a scenario to determine which resources/scenarios it depends on and writes the result back to `service.json`. We already use it to avoid rescanning every run.

## Needed Enhancements

1. **Recursive DAG Export** — Include transitive dependencies (resources of dependencies, etc.) with aggregation.
2. **Resource Requirements** — Sum RAM/CPU/disk/network requirements across the DAG per tier.
3. **Fitness Aggregation** — Read deployment metadata and compute composite scores per tier.
4. **Alternatives Discovery** — When a dependency defines `alternatives`, surface them in the report.
5. **Bundle Manifest Skeleton** — Produce the initial `bundle.json` with the exact binaries/files/config required.

## Proposed Output

```json
{
  "scenario": "picker-wheel",
  "generated_at": "2025-03-06T12:00:00Z",
  "dependencies": [
    {
      "name": "postgres",
      "type": "resource",
      "requirements": {"ram_mb": 512, "disk_mb": 1024},
      "deployment": {"platforms": {"desktop": {"fitness_score": 0.3, "alternatives": ["sqlite"]}}}
    }
  ],
  "aggregates": {
    "desktop": {
      "fitness_score": 0.6,
      "blocking_dependencies": ["postgres", "ollama"],
      "estimated_ram_mb": 2048,
      "estimated_disk_mb": 1500
    }
  }
}
```

## Integration Points

- Feed reports into deployment-manager for visualization.
- Allow CLI usage (`vrooli scenario dependencies picker-wheel --format json`) so agents can script checks.
- Cache reports in `.vrooli/deployment/<scenario>.json` with versioning.

## Open Questions

- How do we tag dependencies that cannot be swapped (regulatory, licensing)? Add `locked: true` metadata.
- Where do we store manual notes (performance quirks, manual tuning)? Possibly `deployment.notes` arrays.
