---
name: "scenario-authenticator"
description: "Accounts, login and tokens for a scenario or the fleet: register a user, sign in and get access and refresh tokens, validate or refresh a JWT and see its claims, manage sessions, scopes, roles and TOTP multi-factor authentication."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["scenario-authenticator", "login", "auth", "jwt", "tokens", "accounts", "sessions", "mfa", "totp", "identity-provider", "oauth"]
  requires:
    scenarios: ["scenario-authenticator"]
    commands: ["scenario-authenticator auth register"]
  origin:
    kind: "authored"
---

## scenario-authenticator

A multi-tenant identity provider issuing RS256 tokens with JWKS.

| You want | Command |
|---|---|
| Create an account | `scenario-authenticator auth register` |
| Sign in | `scenario-authenticator auth login` |
| Check or refresh a token | `scenario-authenticator auth validate`, `scenario-authenticator auth refresh` |
| Sign out / revoke | `scenario-authenticator auth logout`, `scenario-authenticator sessions revoke-all` |
| See sessions | `scenario-authenticator sessions list` |
| Scopes and roles | `scenario-authenticator auth grant-scope`, `scenario-authenticator auth list-scopes`, `scenario-authenticator auth set-roles` |
| Multi-factor | `scenario-authenticator mfa begin-enrollment`, `scenario-authenticator mfa confirm-enrollment` |

Run `scenario-authenticator <group> <command> --help` for flags.
