# Error Handling — Secrets Manager

## API Errors

Handlers return actionable status and error payloads for invalid requests, unavailable dependencies, and failed mutations. They must not include secret values or access tokens.

## Resource Errors

Vault resource preflight reports missing host tooling, signer evidence, or broker authorization before initialization. The remediation must name the supported control-plane action.

## Storage Errors

Database initialization failures prevent unsafe metadata access. Desktop SQLite errors retain the private path boundary and must not be repaired by deleting data blindly.

## Cross-References

- [Troubleshooting](../guides/troubleshooting.md)
- [Security](SECURITY.md)
