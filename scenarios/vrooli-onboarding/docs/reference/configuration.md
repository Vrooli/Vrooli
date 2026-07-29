# Configuration

Scenario and resource manifests declare dependencies, credential descriptors,
and host requirements. `.vrooli/operator-state.json` records only operator
choices and is written atomically by the API. It is reloaded on every entry.

Ordinary credentials use the native credential authority; Vault is only a
declared capability or optional mirror, never an implicit fallback.
