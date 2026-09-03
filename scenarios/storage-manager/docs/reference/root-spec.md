# Governed root specification

The repository contract and owner `storage.entries` declarations are the
source of truth for storage roots. A root declaration names its path, class,
safety tier, age limit, byte limit, protected patterns, lease check, and
supported platforms. Adding a root is a declaration plus validation; it does
not require a provider-specific filesystem walker.

The repository roots are in `.vrooli/repo-contract.json` under `storage.roots`.
Examples include the Go build cache, Go module cache, uv cache, Hugging Face
cache, Playwright cache, scenario-to-desktop staging, browser evidence,
web-console sessions, test runs, and the governed Go work directory.

A root is eligible for recovery only when its declared tier and proof authorize
the action. `regenerable` means that the bytes derive from other inputs, the
owning tool can recreate them, deletion is contained to the exact root, and no
active lease protects the data. Durable models, scenario databases, and active
resources remain outside this class.

Use the storage-manager provider and recovery read surfaces to inspect the
resolved declaration. Do not hardcode a physical path in a new provider.
