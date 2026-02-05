## Tools focus: Landing Page Desktop Uploads

Use scenario-to-desktop build artifacts and the landing-page-business-suite (LPBS) CLI to upload desktop installers into LPBS downloads, locally or via remote profiles.

Required reading:
- `prompt-manager skill read scenario-to-desktop`

---

### 1. When to Use This Tool

| Goal | Use this skill? | Primary command |
|---|---|---|
| Upload desktop installers into LPBS downloads | Yes | `landing-page-business-suite admin-downloads-upload-managed` |
| Build installers for a scenario | Use scenario-to-desktop skill | `scenario-to-desktop pipeline run ...` |
| Deploy LPBS to a VPS | No (use scenario-to-cloud) | — |
| Update landing page copy/branding | No (use LPBS admin/branding tools) | — |

---

### 2. Scope Boundaries

**In scope:**
- Uploading scenario-to-desktop artifacts into LPBS downloads
- Local LPBS uploads and remote LPBS uploads via remote profiles
- Verifying uploads and current download assets

**Out of scope:**
- Deploying LPBS itself (scenario-to-cloud)
- Building desktop artifacts (handled by scenario-to-desktop skill)
- Landing page design/content changes

---

### 3. Manual Inputs Checklist

You will need to supply:
- `{{TARGET}}` scenario name (for scenario-to-desktop)
- `{{APP_KEY}}` (download app key in LPBS)
- `{{PLATFORM}}` (`windows`, `mac`, or `linux`)
- `{{RELEASE_VERSION}}` (e.g., `1.2.3`)
- Artifact file path (from scenario-to-desktop)
- Local LPBS admin credentials (for `admin-login`)
- Remote LPBS credentials + API base (if using remote profiles)

---

### 4. Prerequisites

1) **LPBS must be running locally** (use lifecycle tooling only):
```bash
cd scenarios/landing-page-business-suite
make start
# or: vrooli scenario start landing-page-business-suite
```

2) **Local admin session stored**:
```bash
landing-page-business-suite admin-login --email <local_admin> --password @/path/to/password.txt
```

3) **Download storage configured** (remote or local):
```bash
landing-page-business-suite admin-download-storage-get --json
landing-page-business-suite admin-download-storage-test
```

4) **Download app exists** (create if missing):
```bash
# list apps
landing-page-business-suite admin-download-apps-list --json

# create app (minimal example)
cat > /tmp/download-app.json <<'JSON'
{
  "app_key": "{{APP_KEY}}",
  "name": "{{APP_NAME}}",
  "description": "{{APP_DESCRIPTION}}"
}
JSON
landing-page-business-suite admin-download-apps-create --body @/tmp/download-app.json
```

5) **Desktop artifacts built** (follow scenario-to-desktop skill). Preferred artifact path options:
- `scenario-to-desktop download {{TARGET}} {{PLATFORM}} --output /tmp/{{TARGET}}-{{PLATFORM}}.bin`
- `scenario-to-desktop pipeline status <id> --json` (inspect `final_artifacts`)

---

### 5. Core Workflow

#### A) Upload to local LPBS

```bash
# Build artifacts (see scenario-to-desktop skill)
scenario-to-desktop pipeline run {{TARGET}} --platforms {{PLATFORM}} --clean --wait

# Upload + apply
landing-page-business-suite admin-downloads-upload-managed \
  --file /path/to/artifact \
  --app-key {{APP_KEY}} \
  --platform {{PLATFORM}} \
  --release-version {{RELEASE_VERSION}}
```

#### B) Upload to remote LPBS via remote profiles

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

# Upload + apply through proxy
landing-page-business-suite admin-downloads-upload-managed \
  --remote-profile <REMOTE_PROFILE_ID> \
  --file /path/to/artifact \
  --app-key {{APP_KEY}} \
  --platform {{PLATFORM}} \
  --release-version {{RELEASE_VERSION}}
```

**Multiple platforms:** repeat the upload for each platform artifact.

---

### 6. Convergence Patterns

| Situation | Preferred action | Avoid |
|---|---|---|
| Remote LPBS upload | Use `--remote-profile` | Direct remote API calls from CLI |
| App key missing | Create download app first | Uploading without app (FK failure) |
| Multiple platforms | Upload each platform artifact | Reusing one artifact across platforms |
| Need clean build | Use `--clean --wait` (scenario-to-desktop) | Reusing stale artifacts |

---

### 7. Guardrails

- **Do not bypass lifecycle tooling** (`make start`, `vrooli scenario start`).
- **Do not embed credentials** in command history; use `@file` for passwords.
- **Use `--wait`** for scripted scenario-to-desktop pipeline runs.
- **Remote profile API base must include `/api/v1`.**
- **Uploads require download storage settings** (S3-compatible) to be configured.

---

### 8. Troubleshooting

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `admin session not configured` | No local admin login | `admin-login` | Run `admin-login` again |
| `Remote session expired` | Remote profile cookie expired | `remote-profiles-test` | Re-run `remote-profiles-login` |
| `download storage not configured` | No storage settings | `admin-download-storage-get` | Configure + test storage |
| `download app not found` | Missing app key | `admin-download-apps-list` | Create app via `admin-download-apps-create` |
| Upload `403/Signature` error | Storage creds/headers mismatch | presign response headers | Re-test storage + retry upload |
| `platform is required` | Missing/invalid platform | command flags | Use `windows`, `mac`, or `linux` |

---

### 9. Verification

```bash
# Check artifacts for app/platform
landing-page-business-suite admin-download-artifacts-by-app \
  --query app_key={{APP_KEY}} \
  --query platform={{PLATFORM}} \
  --json

# Check current download app assets
landing-page-business-suite admin-download-apps-list --json
```

For remote profiles, add `--remote-profile <REMOTE_PROFILE_ID>` to the admin commands above.

---

### 10. Output Expectations

**May create/update:**
- Remote profiles and stored sessions
- Download artifacts in storage
- Download assets for app/platform in LPBS

**Must not:**
- Install dependencies without permission
- Modify scenario source code
- Deploy LPBS itself (use scenario-to-cloud)
