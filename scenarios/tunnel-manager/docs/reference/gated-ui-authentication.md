# Gated UI authentication ownership

Tunnel Manager owns the external exposure and Access application metadata for
scenario UIs. It does not provision a private authentication bypass inside
the primary application.

The existing bounded exception is the explicit `/public` asset path. It is
read-only, ownership-guarded, and used for public tunnel/bootstrap assets. The
primary UI and API routes remain subject to the operator-configured trust
boundary. Tunnel Manager may observe or report that boundary, but it does not
repair host state or silently change an Access policy.

For each human-gated application, record the route hostname and ownership in
Tunnel Manager. Tunnel Manager reads the primary application's audience and
the account's Access team domain for lifecycle binding. If the Cloudflare
credential is intentionally least-privilege and cannot read organization
metadata, configure `VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN` once on
Tunnel Manager. Keep application audiences separate; an audience valid for
one UI must not be accepted by another.

The application verifies the origin assertion and owns domain authorization.
Tunnel Manager owns the external route, Access policy lifecycle, and recovery
coordination. Changes to those external resources require the tunnel-manager
owner and are not performed by scenario startup.
