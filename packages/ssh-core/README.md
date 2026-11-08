# ssh-core

Security-sensitive SSH host-key and transport primitives shared by Bridge and
scenario-to-cloud (Go package `sshcore`). Consumers supply their own state path; the package
never silently writes to an operator's `~/.ssh` directory.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
