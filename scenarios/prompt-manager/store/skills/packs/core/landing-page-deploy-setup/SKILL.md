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
- LPBS infrastructure deployment (`scenario-to-cloud`)
- LPBS UI/content customization

---

### 3. Inputs

Always required (base LPBS readiness):
- `{{DOMAIN}}` remote LPBS domain (recommended)
- `{{PROFILE_TAG}}` remote profile tag (example: `prod`)
- Local LPBS admin credentials (`email`, `password`)
- Remote LPBS admin credentials (`email`, `password`)
- `LPBS_SERVICE_SECRET` value used by local LPBS runtime

Optional (only when onboarding or validating a specific app entry):
- `{{APP_KEY}}`: stable machine identifier for one desktop app in LPBS (for example `crm-desktop`)
- `{{APP_NAME}}`: display name, only needed when creating a new app for `{{APP_KEY}}`

---

### 4. Setup Workflow (Idempotent Gates)

Run gates in order. Do not skip a failing gate.

#### Gate A: Ensure local LPBS is running

```bash
vrooli scenario start landing-page-business-suite
landing-page-business-suite status
```

Pass condition:
- `status` command succeeds.

#### Gate B: Resolve remote LPBS API base

Preferred (domain known):

```bash
scenario-to-cloud deployment health --domain {{DOMAIN}} --scenario landing-page-business-suite
```

If this fails, follow `scenario-to-cloud` convergence flow first.

Remote API base for profile setup:
- `https://{{DOMAIN}}/api/v1`

If no domain is available, discover deployment details:

```bash
scenario-to-cloud deployment list --scenario landing-page-business-suite
```

Pass condition:
- A remote LPBS deployment is identified and `deployment health` succeeds.

#### Gate C: Ensure local admin session

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

Resolve profile id by tag:

```bash
landing-page-business-suite remote-profiles-list
REMOTE_PROFILE_ID=<id for {{PROFILE_TAG}} from output above>
test -n "$REMOTE_PROFILE_ID"
```

Login + test:

```bash
landing-page-business-suite remote-profiles-login \
  --email <remote_admin_email> \
  --password @/path/to/remote-admin-password.txt \
  "$REMOTE_PROFILE_ID"

landing-page-business-suite remote-profiles-test "$REMOTE_PROFILE_ID"
```

Pass condition:
- `remote-profiles-test` succeeds for the profile id tied to `{{PROFILE_TAG}}`.

#### Gate G: Ensure `LPBS_SERVICE_SECRET` is set for deploy stage auth

```bash
export LPBS_SERVICE_SECRET='<shared-secret>'
test -n "${LPBS_SERVICE_SECRET:-}"
```

Pass condition:
- `LPBS_SERVICE_SECRET` is non-empty in the shell that runs `scenario-to-desktop pipeline run ... --deploy-*`.

---

### 5. Readiness Verification Bundle

Run this before handing off to deployment:

```bash
landing-page-business-suite admin-session
landing-page-business-suite admin-download-storage-test
landing-page-business-suite remote-profiles-test "$REMOTE_PROFILE_ID"
```

Optional app-level check:

```bash
landing-page-business-suite admin-download-apps-list
```

All executed commands must succeed.

---

### 6. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `admin session not configured` | Missing/expired local admin session | Re-run `admin-login` |
| `download storage not configured` | Storage row missing/invalid | Run `admin-download-storage-update`, then `admin-download-storage-test` |
| `download app not found` for target release | `{{APP_KEY}}` not registered yet | Create app via `admin-download-apps-create` (requires `{{APP_NAME}}`) |
| `api_base must end with /api/v1` | Invalid remote profile API base | Recreate/update profile using full `/api/v1` suffix |
| `Remote profile is not logged in` / session expired | Remote cookie missing/expired | Re-run `remote-profiles-login`, then `remote-profiles-test` |
| Deploy later fails with service auth 401/403 | `LPBS_SERVICE_SECRET` unset/mismatch | Set env var and ensure it matches LPBS runtime token |

---

### 7. Guardrails

- Use lifecycle commands only (`vrooli scenario start`, Makefile lifecycle).
- Do not embed credentials directly in shell history; prefer `--password @file`.
- Do not run desktop deploy commands in this skill.
- Do not use this skill to deploy LPBS infrastructure.

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
- Redeploy LPBS infrastructure
