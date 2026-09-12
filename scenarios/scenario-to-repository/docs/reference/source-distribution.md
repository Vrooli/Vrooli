# Source distribution contract

## Supported path

1. Analyze the source root through Scenario Dependency Analyzer.
2. Capture a stable source digest and versioned `ExportRecipe`.
3. Apply the publishability policy and refuse unresolved or unsafe inputs.
4. Assemble a deterministic `tar_gzip` archive.
5. Verify the exact archive digest and `SOURCE-MANIFEST.json`.
6. Run clean verification against the exported directory or archive.
7. Prepare a human-only publication handoff. Publication is not automated by
   this scenario and is not verified without destination read-back.

The CLI equivalents are `source analyze`, `source assemble`, `source verify`,
and `source prepare-publication`. API consumers use the generated Connect
service in `v1/source/source.proto`. Read-only consumers can list and inspect
durable distributions with `ListDistributions`, `GetDistribution`,
`GetDistributionContents`, `GetPublicationHandoff`, and `GetDistributionDrift`.

Each distribution carries the source, closure, recipe, policy, artifact, and
verification identities, plus the Deployment Manager decision/reference,
destination read-back evidence, publication standing, freshness, and drift
state. Contents classify admitted application/shared/generated/documentation
files. Exclusions carry safe reasons; unresolved obligations and runtime
requirements remain explicit instead of being silently copied into the archive.

## Included and external inputs

The archive includes the admitted source closure and records file paths,
SHA-256 digests, sizes, and modes in the manifest. Git history, `.git`
metadata, private plan artifacts, private key material, and secret-like
content are not export inputs. Runtime resources and host credentials are
external requirements and must be documented in the closure rather than
silently copied.

## Honest readiness states

`assembled` means bytes exist. `verified` means those bytes match the exact
artifact and manifest. `ready_for_publication` means the evidence tuple is
complete for human review. None of these states means that a remote repository
was created, credentials exist, or the exported application is independently
deployable.

## Configuration and support

The API uses `VROOLI_DATA_DIR` for generated artifacts when managed by the
scenario lifecycle. A caller may provide a read-only source root and an
explicit output path. Supported archive format is currently deterministic
`tar_gzip`; unsupported recipes fail closed. The exported package must use its
own declared toolchain and resources; ambient monorepo paths are not a
supported build input.

## Security and contributions

Changes to closure, policy, recipe, archive, or publication behavior require
adversarial tests and a refreshed manifest/verification receipt. Contributions
must preserve the one-way monorepo contract and must not add Git mutation to
lifecycle or test setup.
