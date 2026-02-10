## Tools focus: Landing Page Desktop Upload

Orchestrate end-to-end desktop release delivery: ensure LPBS deployment health, ensure LPBS upload readiness, then run `scenario-to-desktop` deploy pipeline and verify update endpoints.

Required reading:
- `prompt-manager skill read scenario-to-cloud`
- `prompt-manager skill read landing-page-deploy-setup`
- `prompt-manager skill read scenario-to-desktop`

---

### 1. When to Use This Tool

| Goal | Use this skill? | Notes |
|---|---|---|
| End-to-end desktop release to LPBS | Yes | This skill coordinates all required subskills |
| Only deploy/update LPBS infrastructure | No | Use `scenario-to-cloud` directly |
| Only prepare LPBS for uploads | No | Use `landing-page-deploy-setup` directly |
| Only build desktop artifacts | No | Use `scenario-to-desktop` directly |

---

### 2. Scope Boundaries

**In scope:**
- Orchestration across cloud health, LPBS readiness, desktop build+deploy
- Gate-based execution with stop conditions
- Post-release update endpoint verification

**Out of scope:**
- Replacing subskill internals
- LPBS feature/content development
- Non-LPBS deployment targets

---

### 3. Required Inputs

- `{{TARGET}}` scenario to package/deploy
- `{{DOMAIN}}` LPBS domain
- `{{APP_KEY}}` LPBS download app key (stable ID for the specific app release)
- `{{PROFILE_TAG}}` LPBS remote profile tag (example: `prod`)
- `{{PLATFORMS}}` target platforms (example: `linux` or `linux,win`)
- `{{CHANNEL}}` update channel (default: `stable`)

Conditional input:
- `{{APP_NAME}}` only required when `{{APP_KEY}}` does not already exist and must be onboarded

---

### 4. Orchestration Flow

Run stages in order. Each stage has a strict pass condition.

#### Stage 1: LPBS deployment convergence (`scenario-to-cloud`)

Check LPBS deployment by selector:

```bash
scenario-to-cloud deployment health --domain {{DOMAIN}} --scenario landing-page-business-suite --json
```

If missing/unhealthy, converge:

```bash
scenario-to-cloud redeploy \
  --domain {{DOMAIN}} \
  --scenario landing-page-business-suite \
  --if-needed --preflight --wait
```

Freshness rule (fingerprint drift safe-guard):
- If health is `ok=true` but freshness is `outdated`, run a single convergence attempt with a forced bundle rebuild:
  ```bash
  scenario-to-cloud redeploy \
    --domain {{DOMAIN}} \
    --scenario landing-page-business-suite \
    --if-needed --preflight --wait --force-bundle
  ```
- If health remains `healthy` but freshness is still fingerprint-only `outdated` after one `--if-needed` convergence attempt, stop automatic retries and record the drift risk before proceeding.

Pass condition:
- Deployment health is successful for `landing-page-business-suite` on `{{DOMAIN}}`.

#### Stage 2: LPBS upload readiness convergence (`landing-page-deploy-setup`)

Run the setup skill gates (local LPBS started, admin session, storage test, remote profile tested, `LPBS_SERVICE_SECRET` set), then run optional app gate for the release `{{APP_KEY}}`.

Targeting rule (do not skip):
- All `landing-page-business-suite ...` commands in Stage 2 are intended to run against the **local LPBS control-plane instance** (the one `scenario-to-desktop` deploys through).
- The **remote deployed LPBS** at `{{DOMAIN}}` is validated indirectly via the remote profile session (`remote-profiles-*` and `remote-profiles-proxy`).
- Do not point `--api-base` at `https://{{DOMAIN}}` for Stage 2 unless you are intentionally operating the remote instance directly and have an admin session there.

Sequencing rule:
- `landing-page-business-suite admin-login` must succeed before running any `landing-page-business-suite admin-*` or `landing-page-business-suite remote-profiles-*` commands.

Minimal readiness check (preferred; single contract):

```bash
landing-page-business-suite deploy-readiness \
  --profile-tag {{PROFILE_TAG}} \
  --domain {{DOMAIN}} \
  --json
```

Remote readiness gates (required for deploy handoff):
- Remote download storage must be configured and testable (this is the most common deploy blocker).
- Remote app registry must contain `{{APP_KEY}}` (deploy requires the remote LPBS to know this app key).

Remote storage test (via remote profile proxy):
```bash
# Preferred: selector-first by tag (no numeric id required)
landing-page-business-suite remote-profiles-proxy --profile-tag {{PROFILE_TAG}} \
  --method POST --path /admin/download-storage/test --json

# Optional: inspect available profiles (useful for debugging tag/id drift)
landing-page-business-suite remote-profiles-list --json
```

Remote app registry check + optional onboarding:
```bash
landing-page-business-suite remote-profiles-proxy --profile-tag {{PROFILE_TAG}} \
  --method GET --path /admin/download-apps --json
```

If `{{APP_KEY}}` is missing and you intend to onboard it, create it:
```bash
cat > /tmp/lpbs-remote-app.json <<'JSON'
{
  "app_key": "{{APP_KEY}}",
  "name": "{{APP_NAME}}",
  "description": "Desktop application"
}
JSON

landing-page-business-suite remote-profiles-proxy --profile-tag {{PROFILE_TAG}} \
  --method POST --path /admin/download-apps --body @/tmp/lpbs-remote-app.json --json
```

If any readiness check fails:
- Stop release execution
- Follow `landing-page-deploy-setup` to converge

Pass condition:
- Base setup gates pass for `{{PROFILE_TAG}}`.
- Remote storage test passes for `{{PROFILE_TAG}}` (via remote proxy).
- Remote app gate passes for `{{APP_KEY}}` (existing app or newly created app).

#### Stage 3: Build + deploy desktop artifact (`scenario-to-desktop`)

Run deploy pipeline:

```bash
scenario-to-desktop pipeline run {{TARGET}} \
  --platforms {{PLATFORMS}} \
  --clean --wait \
  --deploy-to landing-page-business-suite \
  --remote-profile {{PROFILE_TAG}} \
  --app-key {{APP_KEY}}
```

If this fails, troubleshoot in `scenario-to-desktop`.

Pass condition:
- Pipeline completes successfully through deploy stage.

#### Stage 4: Post-release verification

Verify update manifest endpoint(s) based on platform.

Electron-updater manifest filename mapping:

| Platform | Manifest file |
|---|---|
| `win` | `latest.yml` |
| `mac` | `latest-mac.yml` |
| `linux` | `latest-linux.yml` |

Manifest checks:

```bash
# Linux example
curl -fsS "https://{{DOMAIN}}/api/v1/updates/{{APP_KEY}}/{{CHANNEL}}/latest-linux.yml"
```

Artifact download checks (recommended):
- The manifest only proves update metadata is published.
- Also verify the artifact path itself resolves (LPBS commonly responds `302` to a storage URL, but `200` may occur depending on edge/CDN configuration).

Use the artifact filename from the pipeline output (Stage 3) and check it:

```bash
# Example (artifact filename must match the built output)
curl -sS -o /dev/null -w '%{http_code}\n' --max-redirs 0 \
  "https://{{DOMAIN}}/api/v1/updates/{{APP_KEY}}/{{CHANNEL}}/<artifact-filename>"
```

Pass expectation:
- Returns `302` or `200` for each artifact you shipped (must not be 4xx/5xx).

Optionally verify LPBS artifact records:

```bash
landing-page-business-suite admin-download-artifacts-by-app \
  --query app_key={{APP_KEY}} --json
```

Discovery fallback (when manifest URL returns 404 but Stage 3 succeeded):
- Use LPBS artifact records to discover what the current deployed artifact+version are:
  ```bash
  landing-page-business-suite admin-download-artifacts-by-app \
    --query app_key={{APP_KEY}} --json
  ```
- Then re-run the manifest and artifact checks using the discovered filenames/versions.

Pass condition:
- Manifest request(s) succeed for each platform in `{{PLATFORMS}}`.
- Artifact metadata exists in LPBS.

---

### 5. Decision Table

| Stage failure | Next action |
|---|---|
| Stage 1 fails | Use `scenario-to-cloud` convergence/debug flow; do not continue |
| Stage 2 fails | Use `landing-page-deploy-setup` to fix prerequisites; do not continue |
| Stage 3 fails | Use `scenario-to-desktop` troubleshooting (`preflight`, pipeline diagnostics) |
| Stage 4 fails | Check LPBS app/asset state and update endpoints before declaring release complete |

---

### 6. Guardrails

- Do not skip stages.
- Do not use direct scenario binary execution; use lifecycle commands.
- Do not run deploy flags speculatively; only deploy when user requested release.
- Use `--wait` for pipeline execution.
- Keep command responsibility boundaries: cloud in `scenario-to-cloud`, setup in `landing-page-deploy-setup`, packaging/deploy in `scenario-to-desktop`.

---

### 7. Output Expectations

**Must produce:**
- A released desktop artifact linked to `{{APP_KEY}}`
- Verified update manifest availability at LPBS domain

**May update:**
- LPBS deployment state (via `scenario-to-cloud`)
- LPBS setup state (via `landing-page-deploy-setup`)
- Desktop pipeline/build records (via `scenario-to-desktop`)

**Must not:**
- Bypass prerequisite gates
- Mutate unrelated LPBS business/content configuration

---

### 8. Troubleshooting & Edge Cases

If Stage 3 fails with a generic deploy error (for example `deploy failed` / `INTERNAL_ERROR`) and the stage logs do not include a sub-step cause:

1. Inspect pipeline status:
```bash
scenario-to-desktop pipeline status <pipeline_id> --verbose --json
```

2. Confirm build artifacts exist in the pipeline output:
- `stages.build.details.artifacts` must include at least one platform path for the platforms you deployed.

3. Validate remote prerequisites deterministically (via remote profile proxy):
```bash
landing-page-business-suite admin-session
landing-page-business-suite remote-profiles-proxy --profile-tag {{PROFILE_TAG}} \
  --method POST --path /admin/download-storage/test --json

landing-page-business-suite remote-profiles-proxy --profile-tag {{PROFILE_TAG}} \
  --method GET --path /admin/download-apps --json

# Optional: list profiles for debugging
landing-page-business-suite remote-profiles-list --json
```

If remote storage test fails (common symptom: S3 credential errors), stop and hand off to `landing-page-deploy-setup` remote-storage convergence; do not retry deploy until remote `/admin/download-storage/test` succeeds.
