# Pinned Akash CLI

This owner acquires `akt` 1.0.4 on Linux amd64 through Scenario Dependency
Analyzer. It downloads the exact release ZIP in `tool.json`, verifies the
archive SHA-256 through `binaryfetch`, stages a private candidate, retains the
upstream license, and records the selected tree and its native install ownership.
An existing unowned or changed selection is refused. No upstream installer runs.

```sh
scenario-dependency-analyzer deps install source/akt@1.0.4 --scenario akt --surface host-tool --exact --json
scenario-dependency-analyzer deps install source/akt@1.0.4 --scenario akt --surface host-tool --exact --apply --json
vrooli host install akt --dry-run --json
```

The SDA preview does not approve an unrecorded dependency. Any approval must be
reviewed and written through SDA before applying. `vrooli host install akt`
delegates to the same governed route. This profile does not install a PATH
wrapper: Go callers obtain the exact executable and receipt through
`akt.SelectedInstallation(ctx)`, which rechecks ownership and tree integrity.
The managed root is `~/.vrooli/opt/akt`; `release.json` records the archive and
tree digests. Acquisition has a three-minute context and one shared artifact lock.

The upstream source is [Apache-2.0 licensed](https://github.com/akash-network/akt/blob/v1.0.4/LICENSE).
The reviewed release asset is 35,111,376 bytes. Selected trees are limited to
256 MiB after extraction; archive parsing uses the existing native decoder's
limits. This is a post-extraction profile check, not a new disk admission system.
The invoking workflow must preview temporary/extracted storage and host capacity.
The profile verifies a reviewed digest; no GitHub build-attestation verification
is claimed. Linux arm64, macOS, and Windows remain explicitly unqualified.

The owner provisions no context, wallet, key, account, certificate, chain
transaction, billing arrangement, or public deployment. Offline `sdl` commands
are upstream's no-context/no-key surface; network and funded operations retain
their own custody and authorization requirements. Callers should use a dedicated
`--home` for their task and retain the selected release receipt with results.

Focused validation:

```sh
go test -v ./internal/tools/akt
```

SDA resolver checks additionally reject changes to the owner, ecosystem,
package, version, source manifest, argument vector, policy or install paths
before acquisition. Use the repository's generated-owner preflight and scoped
scenario validation for SDA changes. A passing unit check does not establish
funded Akash provider acceptance, remote recovery, or unattended operation.
