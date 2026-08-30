# Asset update flow

After changing an asset, the catalog engine resolves that asset's declarations,
recomputes its stale rule set, and reuses current evidence for unaffected
assets. Dependents are invalidated through the asset revision index.

Run one gate for one asset with:

```bash
react-component-library catalog gates types --asset-id controls.button
```

The response includes the inspected result and the rule source and declaring
file for each finding. Run the scenario-owned suite for the complete workflow:
`vrooli scenario test react-component-library`.
