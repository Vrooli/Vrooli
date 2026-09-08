# Integrations — Secrets Manager

## Purpose Of This Document

This document describes the external contracts Secrets Manager relies on.

## Dependency Inventory

Postgres is required for shared metadata. Ordinary credential storage and
resolution use the canonical credential authority backed by the host key
service or encrypted authority storage. Claude Code is optional for remediation
workflows. Vault is not an implicit scenario dependency.

## Vrooli Resources

The credential authority validates and provisions secret storage through
stdin-only control-plane commands. The managed-service contract requires
metadata-safe status, recovery-bundle support, and no plaintext API response.
Postgres stores shared metadata.

## Scenario Dependencies

Deployment Manager and scenario-to-desktop consume deployment strategy outputs. Scenario Dependency Analyzer supplies deployment metadata used during manifest generation.

## Third-Party Services

No remote secret service is a runtime dependency. Vault may be selected only by
an explicitly governed Vault-specific capability such as Transit signing.

## Failure Modes

Missing native key-service/authority support fails credential operations with
actionable remediation. Recovery-bundle import/export remains operator
controlled. Database failures produce degraded metadata posture rather than
secret disclosure. External provider metadata is retained with its declared
capabilities, but source health is reported as `unavailable` until that
provider's owner adapter produces a capability receipt; the native vault is
never consulted as an implicit fallback.

The browser native-messaging protocol is maintained under
`platforms/native-host`. Its default host is fail-closed until an authority
adapter is configured. When configured, the adapter calls the four narrow
`/api/v1/native-host/*` endpoints with both the installed-host transport token
and the authenticated owner token. Enrollment is exact-origin, unlock checks
the selected grant and current item revision, and fill returns only the
requested allowlisted field. Revocation removes the enrollment while leaving
vault custody intact. Its Linux, macOS, and Windows registration files are
installation templates rather than platform support claims.

## Cross-References

- [Configuration](../reference/configuration.md)
- [Deployment](../operations/DEPLOYMENT.md)
- [Architecture](ARCHITECTURE.md)
