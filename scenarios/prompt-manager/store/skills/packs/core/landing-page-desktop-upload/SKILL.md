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

If missing/unhealthy/outdated, converge:

```bash
scenario-to-cloud redeploy \
  --domain {{DOMAIN}} \
  --scenario landing-page-business-suite \
  --if-needed --preflight --wait
```

Pass condition:
- Deployment health is successful for `landing-page-business-suite` on `{{DOMAIN}}`.

#### Stage 2: LPBS upload readiness convergence (`landing-page-deploy-setup`)

Run the setup skill gates (local LPBS started, admin session, storage test, remote profile tested, `LPBS_SERVICE_SECRET` set), then run optional app gate for the release `{{APP_KEY}}`.

Minimal readiness re-check commands:

```bash
landing-page-business-suite admin-download-storage-test --json
landing-page-business-suite admin-download-apps-list --json
landing-page-business-suite remote-profiles-list --json
```

If any readiness check fails:
- Stop release execution
- Follow `landing-page-deploy-setup` to converge

Pass condition:
- Base setup gates pass for `{{PROFILE_TAG}}`.
- App gate passes for `{{APP_KEY}}` (existing app or newly created app).

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

Verify update manifest endpoint:

```bash
curl -fsS "https://{{DOMAIN}}/api/v1/updates/{{APP_KEY}}/stable/latest.yml"
```

Optionally verify LPBS artifact records:

```bash
landing-page-business-suite admin-download-artifacts-by-app \
  --query app_key={{APP_KEY}} --json
```

Pass condition:
- Update manifest request succeeds and artifact metadata exists in LPBS.

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
