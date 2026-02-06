## Tools focus: Landing Page Desktop Deploy Setup

Prepare a landing-page-business-suite (LPBS) instance to receive desktop application deployments from the scenario-to-desktop deploy stage.

Required reading:
- `prompt-manager skill read scenario-to-desktop`

---

### 1. When to Use This Tool

| Goal | Use this skill? | Notes |
|---|---|---|
| Set up LPBS for desktop deployments | Yes | Prerequisites before s2d deploy can work |
| Build + deploy desktop apps in one step | Use scenario-to-desktop skill | `pipeline run ... --deploy-to ... --app-key ...` |
| Build desktop apps without deploying | Use scenario-to-desktop skill only | No deploy flags needed |
| Deploy LPBS itself to a VPS | No (use scenario-to-cloud) | -- |

---

### 2. Scope Boundaries

**In scope:**
- LPBS admin login and session management
- Download storage configuration (S3-compatible)
- Download app creation and management
- Remote profile setup and session validation
- Setting `LPBS_SERVICE_SECRET` for service-to-service auth
- Verifying deployment readiness

**Out of scope:**
- Building desktop artifacts (scenario-to-desktop skill)
- The actual upload/deploy flow (handled by s2d deploy stage)
- Deploying LPBS itself (scenario-to-cloud)
- Landing page design/content changes

---

### 3. Manual Inputs Checklist

You will need to supply:
- `{{APP_KEY}}` download app key for LPBS
- `{{APP_NAME}}` human-readable app name
- Local LPBS admin credentials
- Remote LPBS admin credentials (for remote profile login)
- Remote LPBS API base URL (from scenario-to-cloud deployment)

---

### 4. Prerequisites Setup

#### A) Start LPBS locally

```bash
vrooli scenario start landing-page-business-suite
```

#### B) Admin login

```bash
landing-page-business-suite admin-login \
  --email <local_admin> --password @/path/to/password.txt
```

#### C) Configure download storage

```bash
# Check current config
landing-page-business-suite admin-download-storage-get --json

# Test connectivity
landing-page-business-suite admin-download-storage-test
```

Storage must be S3-compatible and configured before any uploads can work.

#### D) Create download app

```bash
# List existing apps
landing-page-business-suite admin-download-apps-list --json

# Create app (if not exists)
cat > /tmp/download-app.json <<'JSON'
{
  "app_key": "{{APP_KEY}}",
  "name": "{{APP_NAME}}",
  "description": "{{APP_DESCRIPTION}}"
}
JSON
landing-page-business-suite admin-download-apps-create --body @/tmp/download-app.json
```

#### E) Discover remote LPBS URL

```bash
# Human-friendly listing (preferred)
scenario-to-cloud deployment list --scenario landing-page-business-suite
```

Use the **DOMAIN** column when present (API base: `https://<domain>/api/v1`).
If DOMAIN is empty, use **HOST** with the API port (default `3001`): `http://<host>:3001/api/v1`.

#### F) Create and login to remote profile

```bash
# Create remote profile (API base must include /api/v1)
landing-page-business-suite remote-profiles-create \
  --tag prod \
  --api-base https://example.com/api/v1

# Find the profile ID
landing-page-business-suite remote-profiles-list --json

# Login to remote profile
landing-page-business-suite remote-profiles-login <REMOTE_PROFILE_ID> \
  --email <remote_admin> \
  --password @/path/to/remote-password.txt
```

#### G) Set service secret

```bash
export LPBS_SERVICE_SECRET="<shared_secret>"
```

This enables s2d's deploy stage to authenticate with the local LPBS via Bearer token. The value must match the secret configured on the LPBS instance.

---

### 5. Verify Deployment Readiness

After completing prerequisites, validate the chain:

```bash
# Test remote profile session (via LPBS CLI)
landing-page-business-suite remote-profiles-test <REMOTE_PROFILE_ID>

# Or test via s2d deploy-target (if saved)
scenario-to-desktop deploy-target test <target_name>
```

---

### 6. Deploy via s2d

With prerequisites complete, deploy is a single pipeline command:

```bash
scenario-to-desktop pipeline run {{TARGET}} --platforms linux --clean --wait \
  --bump-version patch --version-source auto \
  --deploy-to landing-page-business-suite --remote-profile prod --app-key {{APP_KEY}}
```

Or with a saved deploy target:

```bash
scenario-to-desktop deploy-target add prod \
  --scenario landing-page-business-suite --profile prod --label "Production"

scenario-to-desktop pipeline run {{TARGET}} --platforms linux --clean --wait \
  --bump-version patch --version-source auto \
  --deploy-target prod --app-key {{APP_KEY}}
```

---

### 7. Manual Upload Fallback

If the s2d deploy stage is unavailable, you can upload artifacts manually using the LPBS CLI:

```bash
# Build artifacts first (see scenario-to-desktop skill)
scenario-to-desktop pipeline run {{TARGET}} --platforms {{PLATFORM}} --clean --wait \
  --bump-version patch --version-source auto

# Upload to local LPBS
landing-page-business-suite admin-downloads-upload-managed \
  --file /path/to/artifact \
  --app-key {{APP_KEY}} \
  --platform {{PLATFORM}} \
  --release-version <VERSION_FROM_PIPELINE_OUTPUT>

# Or upload to remote LPBS via remote profile
landing-page-business-suite admin-downloads-upload-managed \
  --remote-profile <REMOTE_PROFILE_ID> \
  --file /path/to/artifact \
  --app-key {{APP_KEY}} \
  --platform {{PLATFORM}} \
  --release-version <VERSION_FROM_PIPELINE_OUTPUT>
```

Multiple platforms: repeat the upload for each platform artifact.

---

### 8. Verification

```bash
# Check artifacts for app/platform
landing-page-business-suite admin-download-artifacts-by-app \
  --query app_key={{APP_KEY}} \
  --query platform={{PLATFORM}} \
  --json

# Check current download app assets
landing-page-business-suite admin-download-apps-list --json

# Check update manifest
curl https://<lpbs-domain>/api/v1/updates/{{APP_KEY}}/stable/latest.yml

# Check binary redirect
curl -v https://<lpbs-domain>/api/v1/updates/{{APP_KEY}}/stable/<artifact-filename>
```

If `update_api_key` is set on the download app, include `-H "X-Update-Key: <key>"` in requests.

---

### 9. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `LPBS_SERVICE_SECRET not set` | Missing env var | `export LPBS_SERVICE_SECRET=...` |
| `admin session not configured` | No local admin login | Run `admin-login` |
| `Remote session expired` | Remote profile cookie expired | Re-run `remote-profiles-login` |
| `download storage not configured` | No storage settings | Configure + test storage |
| `download app not found` | Missing app key | Create app via `admin-download-apps-create` |
| Upload `403/Signature` error | Storage creds mismatch | Re-test storage + retry |
| `platform is required` | Missing/invalid platform | Use `win`, `mac`, or `linux` |

---

### 10. Guardrails

- **Do not bypass lifecycle tooling** (`make start`, `vrooli scenario start`).
- **Do not embed credentials** in command history; use `@file` for passwords.
- **Use `--wait`** for scripted scenario-to-desktop pipeline runs.
- **Remote profile API base must include `/api/v1`.**
- **Uploads require download storage settings** (S3-compatible) to be configured.
- **Platform values:** `win`, `mac`, `linux`. LPBS normalizes `win` to `windows`.
- **Prefer the s2d deploy stage** over manual uploads for all new deployments.
- **Avoid concurrent pipeline runs** when using `--set-version` or `--bump-version`.

---

### 11. Output Expectations

**May create/update:**
- Remote profiles and stored sessions on LPBS
- Download apps, artifacts, and assets in LPBS
- Deploy targets in `.vrooli/deploy-targets.json`

**Must not:**
- Install dependencies without permission
- Modify scenario source code
- Deploy LPBS itself (use scenario-to-cloud)
