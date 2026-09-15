# Transport canon

The generated Connect procedures are canonical for application operations. The
legacy REST handlers remain only where they are explicitly listed below or are
needed for compatibility with an existing external caller.

| Concept | Canonical transport | REST status |
| --- | --- | --- |
| Downloads | Connect `DownloadService` | Compatibility surface remains |
| Variants and branding | Connect variant/branding services | Compatibility surface remains |
| Bundles | Connect `BundleAdminService` | Compatibility surface remains |
| AI | Connect `IntelligenceService` | Compatibility surface remains |
| Content and SEO | Connect content/SEO services | Compatibility surface remains |
| Deployment readiness | Connect `DeploymentService.CheckReadiness` for typed callers | JSON endpoint remains for deployment-manager integration |

Deliberate REST exceptions:

- Stripe webhooks require the signed callback shape supplied by Stripe.
- JWKS is a standard discovery document consumed by generic clients.
- Sitemap and robots are crawler documents, not application RPCs.
- Health is the lifecycle probe used by the control plane.
- Asset serving is a byte-oriented delivery surface.
- `GET /api/v1/readiness/state` is the documented deployment-manager JSON
  projection used by Command Center to display readiness state.

The endpoint generator remains the source of the registered route inventory;
this document records transport ownership and does not duplicate that list.
