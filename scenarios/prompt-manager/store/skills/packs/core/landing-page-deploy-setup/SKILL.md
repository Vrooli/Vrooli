## Tools focus: Landing Page Deploy Setup

Prepare a local `landing-page-business-suite` (LPBS) instance so `scenario-to-desktop` can deploy desktop artifacts through LPBS to a remote LPBS deployment.

Required reading:
- `prompt-manager skill read scenario-to-cloud`

Optional reading:
- `prompt-manager skill read scenario-to-desktop`

---

### 1. When to Use This Tool

| Goal | Use this skill? | Notes |
|---|---|---|
| Prepare LPBS for desktop artifact deployment | Yes | This is the canonical setup flow |
| Build/package desktop binaries | No | Use `scenario-to-desktop` |
| Deploy LPBS to VPS | No | Use `scenario-to-cloud` |
| Run end-to-end release orchestration | No | Use `landing-page-desktop-upload` |

---

### 2. Scope Boundaries

**In scope:**
- Local LPBS lifecycle + local admin session
- Download storage readiness (configured + connectivity test)
- Optional download app onboarding (create/verify per-app `app_key`)
- Remote profile creation/login/test against deployed LPBS
- `LPBS_SERVICE_SECRET` requirement for inter-scenario deploy calls

**Out of scope:**
- Desktop build and upload execution (`scenario-to-desktop`)
- New LPBS infrastructure provisioning or first-time VPS/domain/TLS setup (`scenario-to-cloud`)
- LPBS UI/content customization

Clarification:
- Selector-based health checks and convergence of an existing LPBS deployment (for example `scenario-to-cloud deployment health ...` and `scenario-to-cloud redeploy ... --if-needed --preflight --wait`) are in scope for readiness.

---

### 3. Inputs

Required (base LPBS readiness):
- `{{PROFILE_TAG}}` remote profile tag (example: `prod`)
- Local LPBS admin credentials (`email`, `password`)
- Remote LPBS admin credentials (`email`, `password`)
- `LPBS_SERVICE_SECRET` value used by local LPBS runtime

`LPBS_SERVICE_SECRET` source-of-truth rule:
- Use the canonical secret configured by the local LPBS runtime.
- Do not invent ad-hoc per-shell values.
- If the canonical runtime value is unknown, stop and hand off to the LPBS runtime owner before continuing.

One deployment selector is required for Gate B health checks:
- `{{DOMAIN}}` remote LPBS domain (preferred), or
- `{{VPS_HOST}}` remote VPS host (fallback for health/discovery only)

Additional requirement for Gate F (remote profile setup):
- A routable HTTPS API base with `/api/v1` is required.
- Canonical form: `https://{{DOMAIN}}/api/v1`.
- If only host is available and no domain can be resolved, stop and hand off to `scenario-to-cloud` domain/TLS convergence first.

Optional (only when onboarding or validating a specific app entry):
- `{{APP_KEY}}`: stable machine identifier for one desktop app in LPBS (for example `crm-desktop`)
- `{{APP_NAME}}`: display name, only needed when creating a new app for `{{APP_KEY}}`

---

### 4. Setup Workflow (Idempotent Gates)

Run gates in order. Do not skip a failing gate.

Primary-path rule:
- Run Gate A and Gate B first.
- Then run the readiness-first check:
  ```bash
  landing-page-business-suite deploy-readiness \
    --profile-tag {{PROFILE_TAG}} \
    --domain {{DOMAIN}}
  ```
- If `deploy-readiness` reports `READY`, continue directly to Gate G.1 and Gate 5.
- If `deploy-readiness` fails, use Gates C/D/F as remediation gates based on the reported failing checks.

#### Gate A: Ensure local LPBS is running

```bash
landing-page-business-suite --auto-start status
```

Pass condition:
- `status` command succeeds.

Fallback lifecycle path (if needed):

```bash
vrooli scenario start landing-page-business-suite
landing-page-business-suite status
```

If `vrooli scenario start ...` exits non-zero during an in-progress restart, re-run `landing-page-business-suite --auto-start status` and continue only when it succeeds.

#### Gate B: Resolve remote LPBS API base

Preferred (domain known):

```bash
scenario-to-cloud deployment health --domain {{DOMAIN}} --scenario landing-page-business-suite
```

If this fails, follow `scenario-to-cloud` convergence flow first.

Agent handoff rule (when SSH bootstrap is interactive):
- If `scenario-to-cloud` reports that non-interactive bootstrap cannot proceed and asks for a password prompt, stop and hand off the exact interactive command to a human.
- Resume Gate B only after key authorization is configured.

Remote API base for profile setup:
- `https://{{DOMAIN}}/api/v1`

If no domain is available, discover deployment details:

```bash
scenario-to-cloud deployment list --scenario landing-page-business-suite
```

No-domain fallback convergence:
1. From `deployment list`, pick one row (prefer `status=deployed`).
2. If the row has a domain, set `{{DOMAIN}}` and run:
   ```bash
   scenario-to-cloud deployment health --domain {{DOMAIN}} --scenario landing-page-business-suite
   ```
3. If the row has no domain but has a host, set `{{VPS_HOST}}` and run:
   ```bash
   scenario-to-cloud deployment health --host {{VPS_HOST}} --scenario landing-page-business-suite
   ```
4. Do not continue to Gate F until an HTTPS API base is known (`https://<domain>/api/v1`).

Pass condition:
- A remote LPBS deployment is identified and `deployment health` succeeds.

Quality rule:
- If health reports `failed`/`unhealthy`, stop and follow `scenario-to-cloud` convergence flow.
- If health reports `degraded`/warnings, setup may continue but record the risk before any deploy handoff.
- If health reports freshness `outdated`, do not treat Gate B as passed for deploy handoff; converge first:
  ```bash
  scenario-to-cloud redeploy \
    --domain {{DOMAIN}} \
    --scenario landing-page-business-suite \
    --if-needed --preflight --wait
  ```
  Then re-run `deployment health` and continue only when freshness is `current`, or record explicit approved risk for known deterministic fingerprint drift.
  Drift stop condition:
  - If health remains `healthy`, endpoint checks are non-5xx, and freshness is still fingerprint-only `outdated` after one `--if-needed` redeploy, stop automatic retries.
  - Record deterministic fingerprint drift risk and proceed/handoff per `scenario-to-cloud` guidance.

Operator override (only when a human explicitly requests immediate redeploy):

```bash
scenario-to-cloud redeploy \
  --domain {{DOMAIN}} \
  --scenario landing-page-business-suite \
  --force-run --preflight --wait
```

#### Gate C: Ensure local admin session

Run this gate when `deploy-readiness` reports admin session failure, or when you need to perform admin-only mutations.

```bash
landing-page-business-suite admin-session
```

If session is missing/expired:

```bash
landing-page-business-suite admin-login \
  --email <local_admin_email> \
  --password @/path/to/local-admin-password.txt
```

Pass condition:
- `admin-session` returns authenticated admin session data.

#### Gate D: Ensure download storage is configured and reachable

Run this gate when `deploy-readiness` reports download storage failure, or when onboarding storage for the first time.

Inspect current settings:

```bash
landing-page-business-suite admin-download-storage-get
```

Validate connectivity:

```bash
landing-page-business-suite admin-download-storage-test
```

If not configured, update settings first:

```bash
cat > /tmp/lpbs-download-storage.json <<'JSON'
{
  "provider": "s3",
  "bucket": "<bucket>",
  "region": "<region>",
  "endpoint": "<optional-s3-endpoint>",
  "force_path_style": false,
  "default_prefix": "downloads",
  "signed_url_ttl_seconds": 900,
  "public_base_url": ""
}
JSON

landing-page-business-suite admin-download-storage-update \
  --body @/tmp/lpbs-download-storage.json
landing-page-business-suite admin-download-storage-test
```

Pass condition:
- `admin-download-storage-test` succeeds.

#### Gate E (Optional): Ensure a specific download app exists

Run this gate only when:
- onboarding a new app into the bundle, or
- validating a known `{{APP_KEY}}` before a release.

```bash
landing-page-business-suite admin-download-apps-list
```

If `{{APP_KEY}}` is missing and you intend to onboard it, create it:

```bash
cat > /tmp/lpbs-download-app.json <<'JSON'
{
  "app_key": "{{APP_KEY}}",
  "name": "{{APP_NAME}}",
  "description": "Desktop application"
}
JSON

landing-page-business-suite admin-download-apps-create --body @/tmp/lpbs-download-app.json
```

Pass condition:
- If Gate E was executed: `admin-download-apps-list` includes `{{APP_KEY}}`.
- If Gate E was skipped: no app-level assertion is required for base readiness.

#### Gate F: Ensure remote profile exists, is logged in, and passes test

Run this gate when `deploy-readiness` reports remote profile/session failure, or when onboarding a new profile.

List profiles:

```bash
landing-page-business-suite remote-profiles-list
```

Create profile if missing:

```bash
landing-page-business-suite remote-profiles-create \
  --tag={{PROFILE_TAG}} \
  --label=Production \
  --api-base=https://{{DOMAIN}}/api/v1
```

Selector-first rule:
- Prefer `--tag {{PROFILE_TAG}}` over numeric IDs.
- If a numeric ID must be used, it must map to `tag == {{PROFILE_TAG}}`.
- Never assume a fixed ID (for example `1`).

Login + test:

```bash
landing-page-business-suite remote-profiles-login \
  --tag {{PROFILE_TAG}} \
  --email <remote_admin_email> \
  --password @/path/to/remote-admin-password.txt

landing-page-business-suite remote-profiles-test --tag {{PROFILE_TAG}}
```

Pass condition:
- `remote-profiles-test --tag {{PROFILE_TAG}}` succeeds.
- Do not treat `scenario-to-desktop deploy-target test <name>` as a replacement for Gate F pass/fail.

#### Gate G: Ensure `LPBS_SERVICE_SECRET` is set for deploy stage auth

Default safety rule:
- Do not rotate `LPBS_SERVICE_SECRET` during setup when both scopes already verify.
- Verify first, then sync only if missing/mismatched.
- Treat LPBS runtime value as canonical source of truth.

Verify current state first:

```bash
scenario-to-cloud secrets verify LPBS_SERVICE_SECRET \
  --scenario landing-page-business-suite \
  --targets scenario,deployment \
  --domain {{DOMAIN}}

scenario-to-cloud secrets verify LPBS_SERVICE_SECRET \
  --scenario scenario-to-desktop \
  --targets scenario
```

If verify indicates missing/mismatch in either scope, sync using the canonical LPBS runtime value:

```bash
scenario-to-cloud secrets set LPBS_SERVICE_SECRET \
  --scenario landing-page-business-suite \
  --value <canonical_lpbs_service_secret> \
  --targets scenario,deployment \
  --domain {{DOMAIN}} \
  --restart

scenario-to-cloud secrets set LPBS_SERVICE_SECRET \
  --scenario scenario-to-desktop \
  --value <canonical_lpbs_service_secret> \
  --targets scenario
```

Pass condition:
- `LPBS_SERVICE_SECRET` exists for LPBS runtime surfaces (scenario + deployment targets).
- `LPBS_SERVICE_SECRET` also exists in `scenario-to-desktop` scenario scope (same canonical value).
- If deploy-auth probes fail with 401/403 or `LPBS_SERVICE_SECRET is not set`, treat as scope mismatch and re-sync both sides before handoff.

Gate G.1 (required for unattended runs): verify auth gates (and re-verify secrets only if sync was applied)

```bash
landing-page-business-suite service-auth-status --require-enabled
scenario-to-desktop deploy-target test {{PROFILE_TAG}} --require-service-auth
```

If no deploy target exists for `{{PROFILE_TAG}}`, create one first:

```bash
scenario-to-desktop deploy-target add {{PROFILE_TAG}} \
  --scenario landing-page-business-suite \
  --profile {{PROFILE_TAG}} \
  --label Production
```

Pass condition:
- If Gate G performed a sync, both `secrets verify` commands succeed after sync.
- `service-auth-status --require-enabled` succeeds.
- `deploy-target test {{PROFILE_TAG}} --require-service-auth` succeeds.

Optional one-shot diagnosis for the same gate:

```bash
scenario-to-desktop deploy-target doctor {{PROFILE_TAG}}
```

---

### 5. Readiness Verification Bundle

Run this before handing off to deployment.

Path A (recommended when `deploy-readiness` already passed in the current run):

```bash
scenario-to-desktop deploy-target test {{PROFILE_TAG}} --require-service-auth
```

Path B (when `deploy-readiness` has not been run in the current run):

```bash
landing-page-business-suite deploy-readiness \
  --profile-tag {{PROFILE_TAG}} \
  --domain {{DOMAIN}}

scenario-to-desktop deploy-target test {{PROFILE_TAG}} --require-service-auth
```

Optional app-level check:

```bash
landing-page-business-suite admin-download-apps-list
```

All executed commands must succeed.

### 5.1 Cleanup (required if temp files were created)

```bash
rm -f /tmp/lpbs-download-storage.json
rm -f /tmp/lpbs-download-app.json
```

---

### 6. Troubleshooting & Edge Cases

Use this section for long-tail operational failures and recovery paths.

| Symptom | Likely cause | Fix |
|---|---|---|
| `admin session not configured` | Missing/expired local admin session | Re-run `admin-login` |
| `download storage not configured` | Storage row missing/invalid | Run `admin-download-storage-update`, then `admin-download-storage-test` |
| `download app not found` for target release | `{{APP_KEY}}` not registered yet | Create app via `admin-download-apps-create` (requires `{{APP_NAME}}`) |
| `remote-profiles-create` returns `409` `Remote profile tag already exists` | `{{PROFILE_TAG}}` already exists | Run `remote-profiles-list`, select existing profile where `tag == {{PROFILE_TAG}}`, and continue with login/test using that ID |
| `remote-profiles-list` returns `500` | Legacy/invalid remote profile rows (for example `connector_id` is `NULL`) or server-side schema drift | Run `vrooli scenario logs landing-page-business-suite --runtime`; if logs show `list_remote_profiles_failed`, stop Gate F and hand off a data/code fix task before retrying |
| `remote-profiles-create` fails input validation for API base | `api_base` malformed or missing `/api/v1` suffix | Re-run with `--api-base=https://{{DOMAIN}}/api/v1` and follow the command's emitted `Next steps` guidance |
| `Remote profile is not logged in` / session expired | Remote cookie missing/expired | Re-run `remote-profiles-login --tag {{PROFILE_TAG}}`, then `remote-profiles-test --tag {{PROFILE_TAG}}` |
| `remote-profiles-login` returns `401` and later admin commands fail with `admin session not configured` | Remote login failure invalidated local admin session | Re-run `admin-login`, then retry `remote-profiles-login` and `remote-profiles-test` |
| `deployment health` is healthy but freshness is `outdated` | Deployed bundle fingerprint drift vs local scenario state | Run `deployment health` and follow the emitted `Next step` command first. If drift persists with healthy endpoint after one `--if-needed` convergence, record risk and hand off per `scenario-to-cloud` guidance |
| `scenario-to-desktop deploy-target test {{PROFILE_TAG}} --require-service-auth` fails OR deploy later fails with service auth 401/403 | `LPBS_SERVICE_SECRET` unset/mismatch in LPBS runtime scope or in `scenario-to-desktop` scope | 1) Run `scenario-to-desktop deploy-target test {{PROFILE_TAG}} --require-service-auth` and capture exact error. 2) Verify LPBS scope: `scenario-to-cloud secrets verify LPBS_SERVICE_SECRET --scenario landing-page-business-suite --targets scenario,deployment --domain {{DOMAIN}}`. 3) Verify `scenario-to-desktop` scope: `scenario-to-cloud secrets verify LPBS_SERVICE_SECRET --scenario scenario-to-desktop --targets scenario`. 4) Re-sync failing scope, then rerun Gate G.1 and continue only after both checks pass. |

Command timing note:
- `scenario-to-cloud deployment health ...` can take ~40s+ in normal conditions. Use long enough timeouts in scripted runs.

---

### 7. Guardrails

- Use lifecycle commands only (`vrooli scenario start`, Makefile lifecycle).
- Do not embed credentials directly in shell history; prefer `--password @file`.
- Do not run desktop deploy commands in this skill.
- Do not use this skill for first-time LPBS infrastructure provisioning/bootstrap.
- Remove temporary files created under `/tmp` when setup is complete.

---

### 8. Output Expectations

**May create/update:**
- Local admin session state
- Download storage settings
- Download app records
- Remote profiles and stored remote sessions

**Must not:**
- Build desktop artifacts
- Upload/apply desktop artifacts
- Provision or bootstrap brand-new LPBS infrastructure
