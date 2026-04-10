# Remote Profiles (API-First) Plan

## Why we are doing this

We need a reliable way to upload desktop installers (built by scenario-to-desktop) into a
landing-page-business-suite (LPBS) instance that is deployed remotely (via scenario-to-cloud
on a VPS). The current admin auth model in LPBS is session-cookie based (admin_session), while
CLI tooling only supports Bearer tokens and the UI is same-origin with credentials included.
This makes direct remote administration from the UI or CLI impractical and insecure.

By moving "remote profiles" into the LPBS API (instead of the CLI), we get a single
source of truth for remote connections that both the UI and CLI can use. Admins can
view, test, and revoke remote profiles in the UI, and the CLI can reuse the same API
surface without duplicating auth logic. This also avoids cross-domain cookie/CORS issues
by introducing a server-side proxy that uses stored admin sessions.

## Goals

- Centralize remote profile management in the LPBS API.
- Allow admins to view, test, and revoke remote profiles in the UI.
- Provide a server-side proxy that can call remote admin endpoints using stored
  admin_session cookies.
- Enable automation of the installer upload flow (presign -> upload -> commit -> apply)
  without embedding passwords in scripts.

## Non-goals (for this phase)

- No new scenario-to-cloud automation changes yet.
- No changes to the remote LPBS instance itself.
- No attempt to replace cookie auth with bearer tokens (that is a later design option).
- No public exposure of remote profiles or credentials (admin-only).

## Key constraints and context

- LPBS admin endpoints require the admin_session cookie.
- UI fetch calls are same-origin and use credentials: include.
- CLI core only supports Authorization: Bearer today.
- Admin endpoints are under /api/v1/admin/*.
- Remote API base must be the full base URL including /api/v1.

## High-level approach

1) Store remote profile metadata and encrypted admin_session cookies in the local LPBS
   database.
2) Provide admin-only API endpoints to create, login, test, and revoke remote profiles.
3) Add a server-side proxy endpoint that forwards a limited set of admin requests to the
   remote API using the stored session cookie.
4) Update UI (later) to use these endpoints to view and manage profiles.
5) Update CLI (later) to use these endpoints for remote execution.

---

## Prioritized implementation checklist

### P0 - Data model and security baseline (must be done first)

- [x] Define DB schema for remote profiles.
  - Table: remote_profiles
  - Fields (suggested):
    - id (serial / uuid)
    - tag (unique, human-readable handle)
    - label (display name)
    - api_base (full URL including /api/v1)
    - status (active / expired / error)
    - encrypted_session (encrypted admin_session cookie value)
    - session_expires_at (nullable)
    - last_login_at, last_used_at
    - created_by (admin user id), created_at, updated_at
- [x] Add encryption for stored session cookies.
  - Prefer a dedicated env var like LPBS_REMOTE_PROFILE_ENCRYPTION_KEY.
  - If not available, reuse LPBS_API_KEY_ENCRYPTION_KEY with clear warnings.
- [x] Add validation helpers.
  - api_base must be a valid https URL (allow http in development).
  - must include /api/v1 (explicit validation).
  - tag must be unique and safe for URL paths.

### P1 - Core RemoteProfileService

- [x] Implement RemoteProfileService in API layer.
  - create profile (tag, label, api_base)
  - update profile metadata
  - delete profile
  - login(profile, email, password)
    - POST remote /api/v1/admin/login
    - capture admin_session cookie from Set-Cookie
    - GET remote /api/v1/admin/session to verify
    - store encrypted cookie and expiry
  - logout(profile)
    - POST remote /api/v1/admin/logout if cookie present
    - clear stored cookie
  - test(profile)
    - GET remote /api/v1/health or /api/v1/admin/session
- [x] Add structured error handling with clear user messages.
- [x] Add audit logging (profile created, login, logout, proxy usage).

### P2 - Admin API endpoints

- [x] Add admin routes (requireAdmin):
  - GET    /api/v1/admin/remote-profiles
  - POST   /api/v1/admin/remote-profiles
  - PUT    /api/v1/admin/remote-profiles/{id}
  - DELETE /api/v1/admin/remote-profiles/{id}
  - POST   /api/v1/admin/remote-profiles/{id}/login
  - POST   /api/v1/admin/remote-profiles/{id}/logout
  - POST   /api/v1/admin/remote-profiles/{id}/test
- [x] Return sanitized profile data (never return the session cookie itself).
- [ ] Add pagination and filtering (optional, low effort).

### P3 - Server-side proxy endpoint (remote admin calls)

- [x] Add admin-only proxy route:
  - POST /api/v1/admin/remote-profiles/{id}/proxy
- [x] Proxy request shape:
  - method, path, query, body, headers (allowlist headers)
- [x] Allowlist only required admin endpoints (downloads, storage, artifacts, bundles).
  - Example allowlist:
    - /admin/download-storage
    - /admin/download-artifacts
    - /admin/download-assets
    - /admin/download-apps
- [x] Attach stored admin_session cookie to outbound request.
- [x] Return raw status + body so UI and CLI can handle validation.
- [x] Enforce timeouts and size limits to prevent abuse.

### P4 - UI integration (admin portal)

- [x] Add Remote Profiles page under Admin settings.
- [x] UI features:
  - list profiles (status, last login, last used)
  - create/edit
  - login (email + password input)
  - test connection
  - revoke session (logout)
  - delete profile
- [x] Update UI API client to call local /admin/remote-profiles endpoints.
- [x] Add UI warnings for expired sessions.

### P5 - CLI integration (later)

- [x] CLI uses local LPBS API, not remote directly.
- [x] New command group: remote-profiles (list/create/login/test/delete).
- [x] Any future "remote upload" commands should call the proxy endpoint.

### P6 - Tests and validation

- [x] Unit tests for validation and encryption helpers.
- [x] API tests for profile CRUD, login, and proxy allowlist.
- [x] Mock remote server in tests to verify cookie capture and proxy forwarding.
- [x] Add regression tests for expired session handling.

### P7 - Operational docs

- [x] Document the new endpoints in docs/reference/api/admin.md.
- [x] Add a brief admin guide for remote profiles.
- [x] Add a troubleshooting section for common errors (expired session, invalid base URL).

---

## Risks and mitigations

- **Storing session cookies**: mitigate with encryption at rest and admin-only access.
- **Session expiry**: surface clear errors and re-login CTA.
- **Proxy abuse**: allowlist endpoints and restrict headers/body size.
- **Misconfigured api_base**: validation + test endpoint to confirm health.

## Implementation notes / references

- Admin auth uses admin_session cookie and requireAdmin middleware.
- All API routes are under /api/v1; remote api_base must include /api/v1.
- UI fetch helper uses credentials: include and is same-origin.
- Existing download flows already expose all required admin endpoints.
