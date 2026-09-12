# Integration Hub continuation contract

This scenario is a programmatic interface enabler. The generated Connect
contract in `common.v1.integrations` is the source of truth for lifecycle
operations. Keep provider values inside the credential authority and keep this
scenario's JSON state metadata-only. Add connector-specific drivers only behind
the provider-neutral service; never put provider auth logic in consumers or the
shared component library.

Targeted validation:

```bash
GOWORK=off go test ./...
vrooli scenario restart integration-hub
vrooli scenario status integration-hub --json
```
