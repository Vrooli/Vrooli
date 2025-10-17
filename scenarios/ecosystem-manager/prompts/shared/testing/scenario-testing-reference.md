# 🧪 Scenario Testing Reference

## Purpose
Single source of truth for validation commands, checklists, and troubleshooting when exercising a scenario.

## Quick Command Reference

| Command | When to use | Notes |
| --- | --- | --- |
| `make run` | Start scenario through lifecycle system | Preferred entry point before any tests |
| `make status` | Confirm lifecycle status | Mirrors `vrooli scenario status [name]` |
| `make test` | Run full suite | Use `TIMEOUT=` override for slow suites |
| `make logs` | Inspect current logs | Pipe/grep as needed |
| `make stop` | Stop scenario cleanly | Required before switching tasks |
| `make help` | List scenario-specific commands | Documents custom targets |
| `vrooli scenario run [name]` | When Makefile unavailable | Same lifecycle stack, bypasses custom targets |
| `vrooli scenario test [name]` | Remote trigger for CI parity | Uses scenario defaults |
| `vrooli scenario logs [name]` | Retrieve lifecycle-managed logs | Works outside scenario repo |

### Validation Commands

| Flow | Command | Tip |
| --- | --- | --- |
| API health | `curl -sf http://localhost:[PORT]/api/health` | `time` it to confirm <500 ms target |
| API endpoint | `curl -X GET/POST http://localhost:[PORT]/api/[endpoint]` | Use realistic payloads from PRD |
| UI snapshot | `vrooli resource browserless screenshot --scenario [name] /tmp/[name]-ui.png` | Attach screenshot evidence in summary |
| CLI scenario tool | `./cli/[scenario-name] --help` | Ensure help output documents new commands |

## Quick Validation Checklists

- **Full pass**: `make run` → wait for ready logs → API health curl → `make test` → `make stop`
- **Health-only**: `make status` → API health curl → record result
- **UI snapshot** (if applicable): `make run` → screenshot via browserless → verify rendering → `make stop`

## Structure & Readiness Validation
- Run `scenario-auditor audit <scenario-name> --timeout 240` and review the structure and standards results (captures required files, Make targets, and contract violations).
- Address findings before manual spot checks; link the JSON output in handoff notes when major issues remain.

## Testing Obligations Matrix

| Role | Minimum checks | Extra focus | Evidence to capture |
| --- | --- | --- | --- |
| Generator | `make run` + health curl + one P0 flow | Uniqueness proof (`grep -r "[scenario-name]"`) | Screenshot or logs demonstrating working P0 |
| Improver | Prior baseline tests + updated feature tests | Regression guard (`make test`, UI snapshot if applicable) | Commands run + PRD updates referencing verification |

Cross-reference detailed PRD accuracy rules in the `progress-verification` section when updating checklists.

## Troubleshooting Cues

| Symptom | First check | Follow-up |
| --- | --- | --- |
| Port already in use | `lsof -i :[PORT]` | Update config or stop conflicting process |
| Missing dependency resource | `make status | grep -i resource` | `vrooli resource [name] manage start` |
| Build failures or stale artifacts | `make clean` | Re-run build target defined in scenario Makefile |
| UI fails to load | API health curl | Rebuild UI via documented Make target (see scenario README) |
| Tests timing out | Extend `TIMEOUT=` in `make test` | Profile slow test; consider test-specific timeout in script |

For structural gaps, prefer `scenario-auditor` feedback over ad-hoc shell exploration so the remediation path stays consistent with security requirements.

## Performance & Execution Order
- Performance thresholds and validation gate sequencing live in `scenario-validation-gates`; review that section before sign-off.
- In summaries, note both user-value outcomes and any deviations from the target response times.
