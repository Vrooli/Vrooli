---
name: "landing-page-desktop-upload"
description: "Orchestrate desktop release delivery by sequencing scenario-to-cloud, landing-page-deploy-setup, and scenario-to-desktop with explicit gate checks"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["landing-page-business-suite","desktop","upload","deployment","orchestration","scenario-to-desktop","scenario-to-cloud"]
  icon: "upload"
  status: "active"
  revision: 6
  createdAt: "2026-02-04T23:30:00Z"
  updatedAt: "2026-02-08T00:20:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Tools focus: Landing Page Desktop Upload

Drive an end-to-end LPBS desktop release through `deployment-manager`. DM owns the release pipeline: cloud-health gate, LPBS readiness gate, release-id allocation, build/sign/package, publish to LPBS, and post-release verification. This skill is a thin operator wrapper around `deployment-manager releases start`.

Required reading:
- `deployment-manager --help` — the release-pipeline CLI this skill wraps (cloud-health gate, LPBS readiness, release-id allocation, build/sign/package, publish, verify)
- `prompt-manager skill read landing-page-deploy-setup` — for setup-time gates A–G (LPBS storage, app registration, remote profile)

---

### 1. When to Use This Tool

| Goal | Use this skill? | Notes |
|---|---|---|
| End-to-end desktop release to LPBS | Yes | Drives DM, which drives the full pipeline |
| Configure LPBS app/storage/profile for the first time | No | Use `landing-page-deploy-setup` |
| Re-verify an already-published release | No | Call `deployment-manager releases verify <id>` directly |
| Build a desktop bundle without publishing | No | Use `scenario-to-desktop pipeline run` directly |

---

### 2. Scope Boundaries

**In scope:**
- One operator-intent command (`releases start`) that DM resolves into the full pipeline
- Polling for terminal status and surfacing per-platform verification evidence

**Out of scope:**
- Direct multi-stage CLI relays across `scenario-to-cloud`, `landing-page-business-suite`, and `scenario-to-desktop` — DM orchestrates these
- LPBS app onboarding / storage configuration — use `landing-page-deploy-setup`
- Approval gating — use `deployment-manager approvals`

---

### 3. Required Inputs

- `{{PROFILE_ID}}` — deployment-manager profile. Must have an LPBS release config set via DM UI or `PUT /api/v1/profiles/{id}/lpbs-config`.
- `{{COMMIT}}` — git commit hash to release. Must be approved for every required platform.
- `{{VERSION}}` — release version string passed to `scenario-to-desktop` and recorded on LPBS.

Optional:
- `{{CHANNEL}}` — defaults to the profile's `default_channel` (typically `stable`). Free-text; `beta`, `nightly`, etc. are all accepted.
- `{{PLATFORMS}}` — comma-separated. Defaults to `linux-x64,darwin-arm64,win-x64`.
- `{{NOTES}}` — release notes recorded on the release row.

---

### 4. Orchestration Flow

#### Step 1: Sanity check — DM reachable, approval gate green

```bash
deployment-manager --auto-start status
deployment-manager approvals gate {{PROFILE_ID}} --commit {{COMMIT}}
```

Stop condition: if the gate is `BLOCKED`, fix approvals before continuing. DM rejects `releases start` for an un-gated commit.

#### Step 2: Start the release

```bash
deployment-manager releases start {{PROFILE_ID}} \
  --commit {{COMMIT}} \
  --version {{VERSION}} \
  --channel {{CHANNEL:-stable}} \
  --platforms {{PLATFORMS:-linux-x64,darwin-arm64,win-x64}} \
  --notes "{{NOTES}}"
```

What DM does internally (informational — do not script around it):
1. Acquires a profile-scoped Postgres advisory lock so concurrent starts 409 cleanly
2. Probes `scenario-to-cloud` for LPBS deployment health
3. Calls LPBS `POST /api/v1/deploy-readiness` with the service secret
4. Allocates a `release_id` (UUID) and inserts a `releases` row + one `release_platforms` row per target
5. Drives `scenario-to-desktop pipeline run` with `release_id`, `channel`, `app_key`, `remote_profile`, `update_url` injected
6. Calls LPBS `GET /api/v1/updates/{app_key}/verify` per platform. Mismatches mark the release `verify_failed` and return HTTP 502.

Capture the `release_id` from the response.

#### Step 3: Surface the result

```bash
deployment-manager releases get {{RELEASE_ID}}
```

Output shows terminal status (`published` / `verify_failed` / `failed`), per-platform state, and verification evidence (expected vs observed version, sha512 match).

If status is `verify_failed`, retry once after ~30s for CDN/cache propagation:

```bash
deployment-manager releases verify {{RELEASE_ID}}
```

If still failing, escalate. **Do not** manually mark the release published — the mismatch is intentional for operator decision (typically: re-publish or roll forward).

---

### 5. Decision Table

| Symptom | Likely cause | Action |
|---|---|---|
| `releases start` returns 409 `release_in_flight` | Another release mid-flight for this profile | Wait for termination, then retry |
| Cloud-health step fails | LPBS runtime down on `scenario-to-cloud` | `scenario-to-cloud deployment status landing-page-business-suite`; restore health before retry |
| Readiness `download_storage` gate fails | LPBS S3 storage not configured | Run `landing-page-deploy-setup` gates A–C |
| Readiness `app_registered` gate fails | `app_key` from the profile's LPBS config doesn't exist on LPBS | Onboard via `landing-page-deploy-setup` or fix `lpbs_app_key` on the profile |
| Verify step fails every platform | LPBS serving prior version | Wait ~30s for cache, retry `releases verify`; if still failing, inspect evidence |
| 412 "lpbs config missing app_key" | No `profile_lpbs_release_config` row | `PUT /api/v1/profiles/{id}/lpbs-config` with `lpbs_app_key`, `lpbs_remote_profile`, `lpbs_domain` |

---

### 6. Guardrails

- Never call `scenario-to-desktop pipeline run` directly for a publishing release — it bypasses release-id allocation and the post-release verify gate
- Never manually mark a `verify_failed` release as `published`; re-verify or re-publish. Verification is a hard gate
- Channel is free-text; arbitrary values (`beta`, `nightly`) are accepted. To release the same commit on a different channel, run `releases start` again with `--channel <name>`
- Dry-run and skip-packaging orchestrator runs do not create release records. If you want a recorded release, use `releases start`

---

### **7. Output Expectations**

After `releases start`, surface:
- `release_id`
- Terminal `status`
- Per-platform `version` and verify outcome (table or JSON)
- A pointer to `deployment-manager releases get <id>` for re-fetching evidence

For automation: use `--format json` on every CLI call so callers can parse evidence reliably.

---

### 8. Fallback (DM unavailable) — emergency only

Use only when `deployment-manager` is genuinely unreachable. This path bypasses release-id allocation, advisory locking, and the unified verify step.

```bash
# 1. Cloud convergence
scenario-to-cloud deployment status landing-page-business-suite

# 2. LPBS readiness gate
landing-page-business-suite deploy-readiness --profile-tag {{PROFILE_TAG}} --app-key {{APP_KEY}}

# 3. Build + publish via S2D
scenario-to-desktop pipeline run \
  --scenario {{TARGET}} \
  --remote-profile {{PROFILE_TAG}} \
  --app-key {{APP_KEY}} \
  --channel {{CHANNEL:-stable}} \
  --platforms {{PLATFORMS}}

# 4. Verify
landing-page-business-suite remote-profiles-update-verify \
  --profile-tag {{PROFILE_TAG}} \
  --app-key {{APP_KEY}} \
  --channel {{CHANNEL:-stable}} \
  --expected-version {{VERSION}}
```

Once DM is reachable again, call `releases start` (or insert a recovery `releases` row manually) so the release appears in DM's history.
