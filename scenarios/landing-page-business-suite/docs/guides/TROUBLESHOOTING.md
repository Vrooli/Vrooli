---
title: "Troubleshooting"
description: "Solutions for common landing page issues"
category: "operational"
order: 10
audience: ["users", "developers"]
---

# Troubleshooting

## Everyone was signed out after upgrading

LPBS uses `__Host-access_token`, `__Secure-refresh_token`, and
`__Host-lpbs_session_hint` cookie names on HTTPS. The rename intentionally
ends existing browser sessions once so that an old cookie name is never
accepted as authority. Sign in again after the deployment; native and desktop
clients that send tokens in request bodies are unaffected.

Solutions for common issues with your landing page.

---

## Quick Diagnostics

Run these commands to diagnose most issues:

```bash
# Check scenario status
vrooli scenario status "<your-scenario>"

# Check PostgreSQL
resource-postgres status

# View logs
vrooli scenario logs "<your-scenario>" --tail 50

# Check health endpoint
curl http://localhost:<port>/health
```

---

## Startup Issues

### "Port already in use"

**Symptoms**: Scenario fails to start with port binding error

**Solutions**:

1. **Find what's using the port**:
   ```bash
   lsof -i :<port>
   # or
   ss -tlnp | grep <port>
   ```

2. **Stop the conflicting process**:
   ```bash
   kill <pid>
   ```

3. **Let Vrooli reallocate**:
   ```bash
   make stop && make start
   ```

### "Database connection failed"

**Solutions**:

1. **Start PostgreSQL**:
   ```bash
   resource-postgres start
   ```

2. **Check connection parameters**:
   ```bash
   resource-postgres status
   ```

3. **Verify database exists**:
   ```bash
   psql -h localhost -U postgres -c "\l" | grep <scenario-slug>
   ```

4. **Run schema setup**:
   ```bash
   cd <scenario-path>
   make setup
   ```

### "Go build failed"

**Solutions**:

1. **Check Go is installed**:
   ```bash
   go version
   ```

2. **Check for syntax errors** in the logs

3. **Run manually to see errors**:
   ```bash
   cd <scenario-path>/api
   go build -o <slug>-api .
   ```

### UI won't start

**Solutions**:

1. **Check node_modules**:
   ```bash
   cd <scenario-path>/ui
   pnpm install
   ```

2. **Check Vite config**:
   ```bash
   pnpm run dev
   ```

3. **Check for port conflicts** on UI port

---

## Admin Portal Issues

### "Session expired" / Can't log in

**Solutions**:

1. **Clear cookies** for the domain
2. **Check API is running** at `/api/v1/health`
3. **Verify credentials**:
   - There is no default password; provision `admin-default-password` and restart (see [Admin Credentials](ADMIN_GUIDE.md#admin-credentials))

### Changes not saving

**Solutions**:

1. **Check browser console** for API errors
2. **Verify API health**:
   ```bash
   curl http://localhost:<port>/api/v1/health
   ```
3. **Check database connection**

### Live preview not updating

**Solutions**:

1. **Check for JavaScript errors** in browser console
2. **Ensure debounce isn't stuck** - wait 500ms after typing
3. **Refresh the page**

### Analytics showing no data

**Solutions**:

1. **Check time range filter** - may be set to empty period
2. **Verify events are being tracked**:
   ```bash
   curl http://localhost:<port>/api/v1/metrics/summary
   ```
3. **Check database has events**:
   ```sql
   SELECT COUNT(*) FROM events;
   ```

---

## A/B Testing Issues

### Same variant always shown

**Possible causes**:

1. **URL override active** - check for `?variant=` in URL
2. **localStorage stuck** - clear localStorage
3. **Only one active variant** - check variant status

**Solutions**:
```javascript
// Clear variant from localStorage (run in browser console)
localStorage.removeItem('variant_id');
```

### Variant weights not working

**Cause**: Weights are normalized - they don't need to equal 100

**Verify normalization**:
```
If weights are [50, 30, 20]:
Total = 100
Control: 50/100 = 50%
Variant A: 30/100 = 30%
Variant B: 20/100 = 20%
```

### Can't create new variant

**Solutions**:

1. **Check slug is unique** - no duplicate slugs
2. **Check Control variant exists** - new variants copy from Control
3. **Check API logs** for validation errors

---

## Payment Issues

### Stripe checkout not working

**Solutions**:

1. **Verify Stripe keys are set**:
   - Admin -> Billing -> Stripe (`/admin/billing`)
   - Check both publishable and restricted keys

2. **Check Stripe Dashboard** for errors

3. **Verify webhook endpoint** is accessible:
   ```bash
   curl -X POST http://localhost:<port>/api/v1/webhooks/stripe
   # Should return 400 (no signature), not 404
   ```

### Webhooks not received

**Solutions**:

1. **Check webhook secret** matches Stripe Dashboard

2. **Verify endpoint URL** in Stripe:
   ```
   https://your-domain.com/api/v1/webhooks/stripe
   ```

3. **Check Stripe Dashboard -> Webhooks** for delivery failures

4. **For local testing**, use Stripe CLI:
   ```bash
   stripe listen --forward-to localhost:<port>/api/v1/webhooks/stripe
   ```

### Subscription status not updating

**Solutions**:

1. **Check webhook events** are being received
2. **Verify database** has subscription records:
   ```sql
   SELECT * FROM subscriptions WHERE user_email = 'user@example.com';
   ```
3. **Check cache** - status is cached for up to 60 seconds

---

## Performance Issues

### Slow page load

**Solutions**:

1. **Check Lighthouse score**:
   ```bash
   npx lighthouse http://localhost:<port> --view
   ```

2. **Optimize images** - use WebP, compress below 200KB

3. **Check for unnecessary re-renders** in React DevTools

4. **Enable production build**:
   ```bash
   cd ui && pnpm run build && pnpm run preview
   ```

### API slow responses

**Solutions**:

1. **Check database indexes**:
   ```sql
   EXPLAIN ANALYZE SELECT * FROM sections WHERE variant_id = 1;
   ```

2. **Add missing indexes**:
   ```sql
   CREATE INDEX idx_sections_variant_id ON sections(variant_id);
   ```

3. **Check for N+1 queries** in API logs

---

## Deployment Issues

### "502 Bad Gateway"

The API isn't responding:
```bash
# Check if API is running
ps aux | grep landing-api

# Check logs
journalctl -u landing -n 100
```

### "Database connection refused"

```bash
# Verify DATABASE_URL
echo $DATABASE_URL

# Test connection
psql $DATABASE_URL -c "SELECT 1"

# Check PostgreSQL is running
systemctl status postgresql
```

### SSL certificate errors

```bash
# Check certificate validity
openssl s_client -connect landing.yourdomain.com:443 -servername landing.yourdomain.com

# Renew Let's Encrypt
sudo certbot renew --dry-run
```

### Webhook failures in production

```bash
# Check webhook endpoint is accessible
curl -X POST https://landing.yourdomain.com/api/v1/webhooks/stripe

# Should return 400 (no signature), not 404 or 5xx
```

---

## Getting More Help

### Collecting Debug Information

```bash
# Scenario status
vrooli scenario status "<name>"

# Recent logs
vrooli scenario logs "<name>" --tail 200

# Port allocations
./scripts/resources/port-registry.sh list
```

### Reporting Issues

When reporting issues, include:

1. **Exact error message**
2. **Steps to reproduce**
3. **Output of diagnostic commands** above
4. **Relevant log snippets**

---

## Sign-in email delivery

- **Bounced or rejected:** run `admin-email-readiness`, confirm the address and From-domain alignment, then inspect the admin delivery report. Do not expose provider response text to the person signing in.
- **Deferred:** the receiving mail server is delaying the message. Wait and retry delivery; if it persists, inspect provider status and mailbox quotas.
- **No email:** confirm SendGrid or SMTP credentials, run readiness, and verify the Signed Event Webhook public key and event URL. `admin-sign-in-delivery` reports provider acceptance, while `admin-email-readiness` checks configuration.
- **Lost mailbox access:** use the approved account recovery/address-change process. Never issue an admin-created sign-in link.

## See Also

- [FAQ](faq.md) - Frequently asked questions
- [Concepts](../concepts/CONCEPTS.md) - Architecture understanding
- [Deployment Guide](DEPLOYMENT.md) - Deployment troubleshooting

## Isolated buyer journey audit and evidence replay

Use this protocol for an owner-authorized buyer audit. The audit record from 2026-10-02 is a diagnostic report, not production certification. It retained two synthetic identities and left the managed LPBS shadow available for review. Preserve unrelated working-tree changes and original evidence.

### Prove the target before a customer write

1. Discover the owning scenario and current governed operations with `search-hub query "isolated buyer journey landing page business suite" --type library,skill,command`. Read the combined result and current repository testing/lifecycle guidance. A BAS shadow selection isolates BAS infrastructure; it does not isolate the website under test.
2. Read managed scenario status and UI-proxied/API health. Match the actual API build and process identity. Independently join the API's connected socket client ports to `pg_stat_activity` for the intended database. Record exact origin, ports, database, build digest, time and baseline counts. A badge, environment name, header or environment variable alone is insufficient. Test headers may fall back to primary storage.
3. Prove outbound email composition from the owning source, nonsecret settings and startup readiness. Require absent live transports plus a controlled recorder/sink, or an independently established test sink. Confirm the first synthetic message's provider/delivery record against that sink. Do not export older live recorder rows, cookies, codes or bearer links. Do not read complete process environments to obtain credentials.
4. Prove payment mode, remote test account, actual products/prices/currency/period and every webhook destination before creating a session or opening the portal. Join callbacks to the isolated API/database. Valid checkout navigation itself can create a provider session automatically. A test catalog and connected credential health do not establish remote callback safety. Missing proof blocks provider writes; it is not permission to exercise production.
5. Use disposable `example.invalid` identities and a stated fixture budget. Initial audit budget: two users, no subscriptions/provider sessions. Preserve fixtures for review. Expiry cases may update only an identified synthetic token/session, with before/after evidence and a clear distinction from natural elapsed-time expiry. Do not weaken authentication, create persistent passkeys/provider credentials, expose a local service or change production lifecycle state.

### Execute visible buyer cases and preserve failures

- Use the current public site read-only. No production signup, checkout, contact submission, installer preparation or account mutation. Reverify prices, currency, interval, credit conversion, availability and build; do not inherit historical pricing assumptions.
- Assert scoped visible controls, alerts and destinations. Body/CSS/hidden pending text cannot establish success. Check every BAS node and assertion after the run; BAS may continue after a failed step. Terminal completion alone is not a passing journey.
- Test the real authentication contract. Consumer access is passwordless; password reset is inapplicable, while lost-mailbox support remains a separate approved recovery process. Handle fixture codes/links privately and only on the proven local origin, even when the generated message contains a canonical public URL.
- Inspect direct navigation and SPA navigation separately. Verify account persistence with independent session/API/database evidence. Settings navigation can initialize state. Record exactly what changed, including expiry, logout and duplicate normalization cases.
- Bound each write. After a timeout, interruption or duplicate click, inspect the original request/session/event state before retrying. Preserve uncertain outcomes and failed capture/setup commands. Do not turn a source hypothesis into a reproduced charge or entitlement defect.
- Mark each requested journey passed, partial, blocked, unsupported/inapplicable or untested. Provider cases require real test-provider receipts: checkout, signed webhook reconciliation, entitlements, credits/metering, proration, cancellation/end-of-access, renewal failure/grace, paid downloads and interruption/idempotency. Documentation is not a receipt.

### Capture and deliver with identity intact

Keep original screenshots and recordings under their exact execution identities. Copy bytes into the report package and compare SHA-256 digests; retain original paths, node/time metadata, route/build identity and an explicit failure inventory. Export only synthetic/redacted observations, not raw authenticated traces, HAR files or email spools. Verify the actual video codec and full decode. MP4 playback copies are derivatives with separate hashes; local decoding does not prove destination-player playback.

Use the current supported Library transfer helper for upload/materialization, preserve complete returned IDs/version/xattrs, and verify local file existence and hashes after transfer. Local host paths are not cloud paths. The consuming reviewer owns destination materialization, full manifest verification and playback. Keep owner/independent review pending until accepted. Report measured times only; do not invent labor, token or acquisition cost.

The 2026-10-02 audit used LPBS UI 22738/API 15627/database `vrooli_landing_page_business_suite_shadow`, with BAS API 15372 as capture infrastructure. Reverify these identities on every replay. Planner UI 24816/API 17778 is unrelated and must never receive buyer fixtures.



### Buyer-entry auth handoff regression (approved A v1, 2026-10-06)

The browser must refresh the cookie-backed shared user session before displaying successful sign-in or navigating to a protected account page. A presence-hint cookie starts a loading check; it never authenticates a user by itself. Code completion and known successful magic-link completion bind the authoritative session to the verified email. If a verification response is lost, inspect the existing session before another proof write. A masked magic-link preview cannot identify an existing session, so uncertain recovery must show the current account and require explicit inspection rather than claim link success.

Focused owner/reviewer check:

```bash
cd scenarios/landing-page-business-suite/ui
pnpm exec vitest run src/app/providers/UserAuthProvider.test.tsx src/surfaces/user-auth/routes/UserLogin.test.tsx src/surfaces/user-auth/routes/VerifyMagicLink.test.tsx src/app/routes/userAuthRoutes.test.tsx --reporter=verbose
```

Retain exact test names and source hashes. The historical 19:30 October 6 checkpoint independently passed 47 tests in these four files; the expanded desired-behavior tests below now expose remaining failures. Customer route fixtures need the actual shared `UserAuthProvider`. The final compiler diagnostics match the same 27 entries in a copied pre-edit baseline; touched lint retains the same two conditional-passkey cancellation-guard errors. The existing EmailDelivery fixture, declarative runtime-schema test and canonical-render projection drift also fail on unchanged source. Coverage was below its gate in the scoped unit run and has no matched baseline receipt; do not waive it. Test Genie run `20261006-181525-1b6ebd55` remains failed and is not a final exact-source certification.

At 18:14 UTC, `vrooli scenario start landing-page-business-suite@buyer-entry-a --path /tmp/lpbs-buyer-entry-20261006 --json` was rejected by `admitVariantBuild` while primary and shadow instances were active. The guard checks active instances of the scenario identity even with a custom path. Its suggested primary restart was outside this authorization. The exact action stopped. Do not rename the fixture, run a scenario binary directly, force startup or replace shared served assets. Managed candidate qualification still needs the control-plane owner to provide a governed output layout/admission contract if that route is required.

The supported alternative for bounded auth composition is an opt-in **test-owned** runtime, following `api/presentation_browser_evidence_test.go` and `scripts/capture-integrated-fixture.mjs`: a Go test owns an `httptest` API and calls the actual UI server factory with a separate Vite output directory. This is a test lifecycle, not another attempt to start the refused managed scenario. Do not call `setupTestDB` for it: that helper can select an ambient `TEST_DATABASE_URL` and drop its schema. Create an explicit disposable PostgreSQL testcontainer from an already verified cached image, inspect its exact ID/image and loopback-only port binding, and check `current_database()` before schema writes. Join the direct API preflight to the same preflight through the UI proxy before auth writes.

Use the real auth service, JWT/session validation, throttles, same-origin checks and account-security SQL. Inject a nonnetworking sender limited to bounded disposable `@buyer-entry.invalid` recipients. Install no payment routes or provider workers. Restrict browser requests to the exact test origin and prohibit commerce/enrollment writes. Record whether service workers are blocked. Persist the container ID and unique ownership labels immediately; use bounded test cleanup plus enclosing cleanup that validates those exact labels. Disabling the optional testcontainers reaper for this private fixture requires that explicit owner cleanup; never disable product security controls.

Historical fixture `runtime-20261006T191000Z` passed seven authored checks at 19:09:09–19:09:26 UTC, using UI `127.0.0.1:34459`, API `127.0.0.1:41425` and container `c1af465c5e2941153adde21d8f2f44fe12bd258deedc57418c84332cbac85c6c` (database `buyer_entry_fixture`, loopback port 32772). It served the candidate index SHA-256 `700b52c03b3ddd93fd38416c5b62e81732c631c66b5bf9fe4042fd5861790e09`. Code sign-in reached requested `/account/security` after “Not now”; magic preview stayed unconsumed until explicit confirmation; a failed session read retried only the session; consumed-link access was denied. Reopen preserved each synthetic API/SQL identity. Logout changed each session from one active to zero active/one revoked, and the formerly valid access token returned `401 session_revoked`. The testcontainer was removed by the owning test.

Read authoritative session activity with `NOT revoked AND expires_at > NOW() AND absolute_expires_at > NOW()`. The legacy `revoked_at` field is not the production validator's predicate; a former fixture assertion using it failed and remains retained as a harness error. Likewise retain failed selector/import attempts. Scope visible assertions to the intended page content, and wait for ancestor opacity/filter transitions to settle before capture; `isVisible` alone can capture faded content. Keep the screenshot that deliberately demonstrates “Checking…” separate from success evidence.

These receipts qualify the local auth composition only. They do not qualify managed deployment, mobile behavior, expiry/cross-browser binding, passkey authentication/enrollment, native callbacks, provider delivery, commercial account reads, payments, entitlements or downloads. The account fixture deliberately omits commercial APIs, so its “plan unavailable” message is not evidence of a commerce defect. The Security page is visibly unstyled in the candidate bundle; record that separately and propose a focused UI acceptance scope rather than repairing it under auth authority.

Provider diagnosis belongs to the existing email and Stripe executions. Public health reported Mailgun ineligible on October 6 even though host credential metadata showed its key present. Stripe test fields were present and canonical callback GET returned 405; those observations do not prove production mode, remote test-account identity, a safe isolated callback, signed reconciliation or entitlement. Never enter a valid checkout URL under read-only/provider-diagnosis authorization: navigation can create a provider session.

Private replay sources, exact build/source manifests, all failed attempts, screenshots, recordings and the final joint qualification receipt accompany the BUYER-ENTRY A v1 report. Preserve original Library identities and hashes during transfer; an executor host path is not a cloud path. All work remains owner-unreviewed.


#### Remaining A cases: negative qualification and exact stopping boundary

The additional `runtime-20261006T200500` run actually executed at 20:03:47–20:04:21 UTC (owner lifecycle 44.821 seconds), with UI 45513/API 41483 and verified disposable container `c088c9a19b50d96892c67da66042959ea952e1fb6e8a12e335e22d1e8ed06660`. It used the same exact historical candidate build, four disposable recipients, six controlled sink deliveries and three bounded expiry mutations on already-issued synthetic rows. The fixture's expiry seam is private to the test-owned API and checks its exact isolated container/database identity; it changes synthetic proof/idle/absolute expiry only, never validator clocks or product guards. Preserve expired unused proofs and independent API/storage reads. A successful code invalidates both outstanding proofs for that identity; do not require only one used row when two codes were issued.

The independent matrix is seven passed and five failed. Passed: expired code and link denial, invalid/missing link denial, differently bound code denial, cross-browser website magic link with requested Security destination and session-only retry, expired-session reopen denial, and lost-response uncertain state with explicit current-account inspection. Failed: delayed code overriding “Use a different email,” delayed magic completion navigating after route departure, idle and absolute expiry leaving mounted private views, and another tab retaining the profile after logout. Old access returned 401/session_revoked and expired sessions returned 401/session_expired; stale content is browser state, not evidence that server authorization remained granted. Disabled duplicate confirmation/code submissions each held one request. Commercial APIs remain absent from the fixture; do not diagnose commerce from its account error messages.

The first two additional attempts contained independently identified fixture defects (empty UUID storage query, pending-state transition assertion, then an overstrict used-proof count); originals remain preserved. The later 19:59 run combines requested destination and cross-tab logout into one failed check; the 20:05 run separates them and proves the requested destination itself works. Continue only independent safe cases after a failure, never repeat a consumed proof blindly, and always make the aggregate receipt fail if any desired assertion fails.

Expanded shared-source regressions reproduce six auth-state/navigation failures and one real AccountSecurity identity-cache failure. Product behavior has not been changed during this continuation. The additional four test files intentionally describe desired behavior and remain red until a permitted fix is applied. Conditional/explicit passkey tests are synthetic proposed additions: existing behavior passes five of six; top-level mediation/AbortSignal placement fails the conditional contract. A private two-line fix passed six of six before the shared-source stopping boundary; this does not establish real authenticator behavior or authorize enrollment. Preserve stopped/abort cancellation guards.

Prepared auth and passkey patches remain unapplied. Automatic approval review rejected the exact shared-product edit twice, treating the original audit no-repair instruction as controlling despite the current native brief recording later A approval. The exact action stopped; no indirect application, route change, lifecycle workaround or credential operation followed. A direct exact-patch authorization is pending. Proposal review must cover pending resend and explicit passkey completion as well as code/magic route cancellation, superseded authoritative reads, passive focus and cross-tab logout, and identity-owned private cache reset. Replay the desired tests and visible/API/SQL matrix against the exact final source/build after any permitted repair; a proposed patch is not a fix.

Existing provider read routes were resolved independently. The available local LPBS service identity read remote-profile metadata and redacted local Stripe settings with HTTP200 at 19:49:04 UTC. CLI admin commands pre-gate on stored browser cookies even where the server admits a service reader; a CLI pre-call error is not a server denial. The established production service profile12 Stripe metadata relay returned actual HTTP401 at 19:50:32 UTC, then stopped without alternate target or identity. Email readiness and cached provider verdicts require an admin cookie and are excluded from that relay allowlist; no usable exact public admin session was found and neither read occurred. Do not infer missing keys, disconnection, current remote mode/account or provider usability from those distinct results. Append-only email note `7d585105-f435-461c-bcc8-8d40d8cfa8e2` and Stripe note `fbf50514-ba9f-4c6c-b515-74292d44db2d` preserve the prior 18:22 notes and partial statuses.

Current TypeScript still has exactly the same 27 diagnostic signatures. Coverage remains specifically unverified for exact final source; the earlier below-threshold result has no comparable baseline and must not be called baseline or waived. The latest original captures are Library `libfile_95e124a68a188191982f47efb6b7664e`; the auth proposal is `libfile_42f2b087cbd88191a83ca6c190cbce44`. Verify version, original hashes and all archive members after supported materialization. All work remains owner-unreviewed and A acceptance remains partial/failed on the named cases.
