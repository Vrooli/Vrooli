### Scenarios reviewed
Reviewed 3 eligible queued scenarios with fresh GCT completeness runs:
- `knowledge-observatory`: functional_incomplete, score 52, calculated `2026-05-17T04:01:54Z`
- `scenario-to-cloud`: functional_incomplete, score 41, calculated `2026-05-17T04:02:02Z`
- `browser-automation-studio`: foundation_laid, score 38, calculated `2026-05-17T04:02:10Z`

Skipped `web-console` as same-day cooldown because it had a prior `reviewed-scenario/web-console` production_ready score 98 from `2026-05-16T16:02:13Z`.

### Findings converted to backlog
Created 6 Swarm Manager fixes with `notes.md` evidence:
- `fix/qa-knowledge-observatory-gct-requirement-target-readiness-20260517`
- `fix/qa-knowledge-observatory-gct-test-depth-20260517`
- `fix/qa-scenario-to-cloud-gct-requirement-target-readiness-20260517`
- `fix/qa-scenario-to-cloud-gct-test-depth-20260517`
- `fix/qa-browser-automation-studio-gct-requirement-target-pass-rate-20260517`
- `fix/qa-browser-automation-studio-gct-test-decomposition-20260517`

### Dependencies wired
Wired:
- KO QA fixes onto `execute/project-wiki-ko-integration`.
- Scenario-to-cloud QA fixes onto `execute/tool-authoring-migrate-existing`.
- BAS QA fixes onto `research/bas-screenshot-api-audit`, `research/bas-grounded-mockup-flow`, and `execute/bas-grounded-mockup-integration`.

### Skipped scenarios
- `web-console`: cooldown/duplicate churn avoidance.
- No selected scenario was missing; all three reviewed scenarios had visible files through `swarm-manager scenarios files`.

### Bugs filed (via report-bug)
Filed:
- `bug-inbox/prompt-confusion/report-friction-writer-team-mismatch`: `knw-1778990924577888480`

Reason: `report-friction` says cross-team friction writes to `meta-optimization` are universal, but `prompt-manager team knowledge-add meta-optimization ...` failed with `team_mismatch`, and the skill’s documented `--by` flag is removed.

### Knowledge entries written
- `qa-run/knowledge-observatory`: `knw-1778990811196572327`
- `qa-run/scenario-to-cloud`: `knw-1778990811196845087`
- `qa-run/browser-automation-studio`: `knw-1778990811346415327`
- `reviewed-scenario/knowledge-observatory`: `knw-1778990832260086757`
- `reviewed-scenario/scenario-to-cloud`: `knw-1778990832261087950`
- `reviewed-scenario/browser-automation-studio`: `knw-1778990832432716235`
- `dependency-wiring/2026-05-17-gct-completeness-queued-scenarios`: `knw-1778990832433553207`

Friction not filed due the bug above: `swarm-manager scenarios files` is still too noisy for existence checks; BAS emitted 172,888 lines / roughly 1.9M transcript tokens before truncation.