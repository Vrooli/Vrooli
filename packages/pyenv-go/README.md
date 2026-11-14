# pyenv-go

Provisions and validates a private, lock-pinned Python virtualenv built by uv and invoked by
absolute interpreter path (Go package `pyenv`). Consumers own their dependency manifests; this
package owns only the reproducible environment lifecycle and takes the lock file path as input.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
