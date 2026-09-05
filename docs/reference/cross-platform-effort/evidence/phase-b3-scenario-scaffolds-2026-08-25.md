# Phase B3 — Scenario scaffold cleanup

Date: 2026-08-25

## Evidence

- `rg -l -i 'generated from the .*react-vite.*template' scenarios/*/README.md`
  returns no files after rewriting 44 affected READMEs (the baseline was 43;
  the additional match was the newly present RCL fixture).
- `find scenarios -type d -name notes` returns no directories.
- `find scenarios -path '*/docs/START-HERE.md' -type f` returns 71 files.
  They remain because no machine-readable completion marker was found for the
  generated onboarding protocol. The 19 legacy scenario-owned guides without
  `.vrooli/orientation.json` are:

  `scenario-dependency-analyzer`, `structure-health`,
  `scenario-completeness-scoring`, `code-facts`, `offer-desk`, `api-health`,
  `business-health`, `template-manager`, `secrets-manager`, `quality-health`,
  `search-hub`, `fall-foliage-explorer`, `experience-manager`,
  `browser-automation-studio`, `tidiness-manager`, `meta-optimization-manager`,
  `vrooli-onboarding`, `money-ledger`, and `cli-health`.

  Their docs manifests still reference the guides, so deleting them without a
  completed initialization record would make the documentation contract less
  truthful. The remaining 52 have active orientation metadata and are
  explicitly incomplete until their finalize protocol runs.

## Enforcement

Structure Health now owns `SCENARIO_README_POLICY`, an enforced scenario rule
that rejects the template scaffold sentence in a scenario README. Its unit
tests cover both the negative scratch case and a scenario-specific README.

Validation:

```text
go test ./internal/packs/structurepack/requiredlayout ./internal/packs
vrooli hygiene --fail-on error
```

The hygiene command exited 0 with no blocking issues.
