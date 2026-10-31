# DNS Setup for VPS Deployments

This guide covers two concerns that are often confused:

- **Web DNS** — the records that make the site answer. Sections below, Options A and B.
- **Mail DNS** — the records that authorize a provider to send email *as* the
  domain. See [Mail DNS](#mail-dns) at the end. A deployment can serve pages
  perfectly while being unable to send a single email, and the web checks will
  not notice.

Scenario-to-cloud supports two DNS patterns when deploying to a VPS:

1) **Cloudflare Worker proxy (OG meta injection)**: Cloudflare sits in front of the VPS and a Worker injects Open Graph metadata for crawlers.
2) **Basic DNS (no Worker)**: DNS points directly to the VPS with no Cloudflare Worker layer.

This doc explains both setups and how they relate to scenario-to-cloud preflight checks.

---

## DNS policy and certificate issuance

- `edge.dns_policy` in the manifest decides whether the plan requires DNS to
  resolve to the target (`required`, the default), warns (`warn`) or skips the
  check (`skip`). Under `required` a mismatch blocks the plan before any
  target effect.
- If the apex/`www` records are proxied through Cloudflare (orange cloud),
  HTTP-01 issuance cannot reach the origin and DNS-01 is required. Supply a
  Cloudflare API token as a deployment secret (`CLOUDFLARE_API_TOKEN`) so
  Caddy can complete DNS-01.
- DNS-only during issuance (the simpler path): set apex/`www` and the edge
  domain to DNS-only until certificates are issued, confirm the A/AAAA
  records point at the target, make sure inbound 80/443 are open at the
  firewall and the provider security group, then re-enable proxying if you
  want it. `scenario-to-cloud edge dns-check <deployment_id>` and
  `edge tls <deployment_id>` report the state at each step.

## Option A: Cloudflare Worker proxy (recommended when you need dynamic OG tags)

**When to use:** You want the Cloudflare Worker in `platforms/og-worker/` to intercept crawler requests and inject OG meta tags before serving content from your VPS.

**How it works (high level):**
- Public traffic goes through Cloudflare (orange cloud).
- The Worker runs on matching routes (apex and subdomains).
- The Worker fetches the origin from a **different hostname** that is DNS-only (grey cloud) to avoid loops.

**Example DNS records**

| Type | Name | Value | Proxy | Purpose |
|------|------|-------|-------|---------|
| A | @ | 203.0.113.10 | Proxied (orange) | Public site (Worker intercepts) |
| A | do-origin | 203.0.113.10 | DNS-only (grey) | Origin host the Worker fetches |
| A | api | 203.0.113.10 | Proxied (orange) | API subdomain |
| CNAME | www | @ | Proxied (orange) | WWW redirect |

**Critical notes**
- The apex (`@`) **must be proxied** so the Worker runs in front of traffic.
- The origin host (example: `do-origin.example.com`) **must be DNS-only** so the Worker can call it without recursion.
- Your Worker config must route your public domain and set a different `ORIGIN_HOST`.

**Worker config alignment (wrangler.jsonc)**
- Routes: `example.com/*` and `*.example.com/*`
- `ORIGIN_HOST`: `do-origin.example.com` (or another DNS-only hostname)

---

## Option B: Basic DNS (no Worker)

**When to use:** You do not need dynamic OG tag injection and want a direct path to the VPS.

**How it works (high level):**
- DNS points directly to the VPS.
- Your reverse proxy (Caddy) handles HTTPS and routing.

**Example DNS records**

| Type | Name | Value | TTL | Purpose |
|------|------|-------|-----|---------|
| A | @ | 203.0.113.10 | 300 | Public site (direct to VPS) |
| A | api | 203.0.113.10 | 300 | API subdomain |
| CNAME | www | @ | 300 | WWW redirect |

---

## How scenario-to-cloud validates DNS

Scenario-to-cloud preflight checks include:
- DNS for the public domain resolves to the VPS.
- Ports 80/443 are reachable for Caddy/Let's Encrypt.

In **Option A**, DNS still points to the VPS, but through Cloudflare for the public hostnames. The Worker-origin hostname (DNS-only) must also resolve directly to the VPS so the Worker can fetch it. In **Option B**, the public domain records point directly to the VPS with no Cloudflare proxy layer.

---

## Quick decision guide

- Need OG meta injection for crawlers? Use **Option A** and the Cloudflare Worker.
- No dynamic OG requirements? Use **Option B** for the simplest DNS setup.

---

## Mail DNS

Web DNS decides whether the site answers. Mail DNS decides whether the domain's
email is accepted. They are independent: every web check can pass while every
email is rejected.

A deployment that sends email declares which providers it uses. Preflight
verifies, for each of them, that the domain publishes the records that provider
requires. Under `edge.dns_policy: required` a missing mail record blocks the
deployment, in the same way a missing A record does.

### The three records and what each one does

| Record | Answers | Failure mode if wrong |
| --- | --- | --- |
| **SPF** (TXT at the sending domain) | Which servers may send for this envelope domain | Mail from an unlisted server fails SPF |
| **DKIM** (TXT or CNAME at `<selector>._domainkey.<domain>`) | The public key that verifies the provider's signature | Signature cannot be verified; DKIM fails |
| **DMARC** (TXT at `_dmarc.<domain>`) | What receivers should do when neither SPF nor DKIM aligns | With `p=reject`, unauthorized mail is refused outright |

### Rules that are easy to get wrong

**Exactly one SPF record per hostname.** A second `v=spf1` record does not add
authorization — it breaks SPF evaluation entirely for that domain. Adding a
provider means *merging* its mechanism into the existing record.

**Ten DNS lookups per SPF record, including nested ones.** Exceeding it makes
SPF fail. The limit is per record, so a provider that uses its own return-path
subdomain carries a separate record with a separate budget. Typical costs:
Amazon SES 1, SMTP2GO 1, SendGrid 2, Mailgun 3 (its include expands to two more).

**Never delete an SPF record to recreate it.** The gap is a total mail outage
under a `p=reject` policy. Use an update that replaces the value in place.

**DKIM selectors are published as TXT by some providers and CNAME by others.**
A verifier that checks only one record type reports a false failure on a
correctly configured domain. Check the type the provider actually specifies.

**Alignment is what DMARC evaluates, not merely a pass.** With `aspf=s` (strict),
SPF aligns only when the envelope domain matches the visible From domain exactly.
A provider using its own return-path subdomain will not align on SPF and must
therefore have a working DKIM signature. Do not add a provider on the strength of
an SPF entry alone.

**Do not add the VPS IP to SPF because the app calls an email API over HTTPS.**
The receiving server sees the *provider's* sending host. Authorize the VPS only
if it originates SMTP delivery itself.

**Adding an outbound provider must not disturb MX records.** MX decides where
*incoming* mail goes. Replacing it to satisfy an outbound provider silently
breaks reply handling.

### Who writes mail DNS

Writes go through `tunnel-manager`, which holds the Cloudflare credential, and
are initiated from a local machine by an operator who reviews the dry-run diff
first. A deployed scenario never holds a DNS-editing credential: a token that can
edit DNS can take the whole domain offline, and a public-facing VPS is the wrong
place for it.

Verification is the opposite — it is read-only, needs no credential, and runs
both at deploy time and continuously inside the running deployment. Deploy-time
asks "is it right?"; the running deployment asks "is it *still* right?", and uses
the answer to decide which providers it may send through.

### Verify by hand

```bash
dig +short TXT example.com                        # SPF — expect exactly one v=spf1 record
dig +short TXT _dmarc.example.com                 # DMARC policy
dig +short TXT <selector>._domainkey.example.com  # DKIM, when published as TXT
dig +short CNAME <selector>._domainkey.example.com # DKIM, when published as CNAME
dig +short MX  example.com                        # inbound routing — record before changing anything
```

A non-empty answer is not a passing check. Compare the record type, the owner
name and the value against what the provider requires, and follow any alias.

For a worked example of a live configuration, the providers it authorizes, and
how a deployment consumes these verdicts, see
`scenarios/landing-page-business-suite/docs/reference/EMAIL_DELIVERY.md`.
