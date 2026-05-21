### Scenario audited
`vrooli-events`

### Skill applied
`cognitive-load-reduction`

### Findings
Material local-readability drift in the UI selector/testing surface. `ui/src/consts/selectors.ts` is 373 lines and implements a generic recursive selector framework for a small manifest with only two dynamic selectors. Comments claim `selectors.manifest.json` is generated during tests and reference Vrooli Ascension workflows, but no generated manifest file exists; tests import `selectorsManifest` directly. Related doc drift: `docs/internal/PROBLEMS.md` still says runtime rate limit counters are missing, while `internal/middleware/policy_ratelimit.go` and tests implement them.

Tidiness-manager scans could not run. `tidiness-manager scan` failed due missing API base, and `--auto-start` failed with the known runtime registry open error.

### Backlog item created
`execute/vrooli-events-selector-registry-cognitive-load`

### Bugs filed (via report-bug)
None. The tidiness-manager auto-start failure matches existing `bug-investigation-report/vrooli-runtime-registry-open-fails`, so no duplicate was filed.

### Knowledge entries written
`knw-1779347036346849745` under `quality-audit/vrooli-events/cognitive-load-reduction`.

Next run should continue rotation after `cognitive-load-reduction`, likely `decision-boundary-extraction`. Avoid recent pairs inside the seven-day window: `web-console/boundary-of-responsibility-enforcement`, `vrooli-events/seam-discovery-and-enforcement`, `web-console/invariant-discovery-and-enforcement`, and `vrooli-events/cognitive-load-reduction`.