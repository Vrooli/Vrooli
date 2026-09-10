# Release Identity

A release is one immutable artifact set: the deterministic mini-Vrooli bundle,
the native control-plane binary for the target platform, the dependency
closure digest and the nonsecret configuration digest, bound by one
`release_digest`. The artifact tested is the artifact promoted: nothing is
compiled or fetched at deploy time, and the target verifies the same manifest
before it extracts anything.

> [CODE: packages/cloudrelease] — manifest shape, canonical JSON, digest rule (shared with `internal/cloudtarget`)
> [CODE: api/release] — builder, verifier, store, provenance, target-delivery resolver
> [CODE: api/releasesvc] — Connect `ReleasesService` + REST adapter
> [CODE: api/handlers_release.go] — `POST /api/v1/releases/build`, `GET /api/v1/releases/{digest}`, `POST /api/v1/releases/{digest}/verify`
> [CODE: api/vps/native_cli.go] — delivery of the release's native binary (no deploy-time build)
> [PROTO: packages/proto/schemas/scenario-to-cloud/v1/releases/releases.proto]

## Identity

```
release_digest = hex(sha256(canonical_json({
  "bundle_sha256":        <hex>,
  "closure_digest":       <closure.Digest, "sha256:…">,
  "configuration_digest": <hex>,
  "native_cli": {"goarch": …, "goos": …, "sha256": <hex>}
})))
```

Canonical JSON is sorted-key, compact `encoding/json` output (the same bytes as
Python's `json.dumps(v, sort_keys=True, separators=(",", ":"))`). Provenance and
archive limits are deliberately outside the identity: they describe how the
release was produced and how it may be unpacked, not what it is. The one
implementation lives in `packages/cloudrelease`; `internal/cloudtarget`
(`vrooli cloud-target release verify|stage`) aliases it, so the cloud builder
and the target verifier cannot drift.

| Input | Digest | Rule |
|---|---|---|
| Bundle | `bundle_sha256` | sha256 of the deterministic tar.gz (`api/bundle`: pinned headers, sorted paths, secrets stripped) |
| Native control plane | `native_cli.sha256` + `goos`/`goarch` | `go build -trimpath -buildvcs=false -ldflags=-buildid=` with `CGO_ENABLED=0 GOOS GOARCH GOWORK=off GOTOOLCHAIN=local`; version and flags recorded |
| Dependencies | `closure_digest` | `closure.Digest` of the resolved closure (or `manifest.dependencies.closure_digest`) |
| Configuration | `configuration_digest` | sha256 over canonical JSON of the manifest with `secrets` nulled and the target locator (`target.vps.host`, `port`, `user`, `workdir`) cleared |

The configuration digest is therefore identical for the same scenario,
dependencies, bundle selection, ports, edge and preserve paths regardless of
which machine the deployment lives on, and it never covers a secret.

## Layout

Beneath the bundle store (`bundle.GetLocalBundlesDir()`):

```
releases/<release_digest>/bundle.tar.gz
releases/<release_digest>/vrooli-<goos>-<goarch>
releases/<release_digest>/release-manifest.json     # schema_version 1, the identity above + provenance + limits
releases/<release_digest>/inputs.json               # every material input, references only
releases/<release_digest>/provenance/…              # signed releases: resource-deployment stage + detached signature
releases/<release_digest>/.complete                 # written last; absent = not a release
releases/<release_digest>.lease.json                # artifact ownership (packages/artifactlease)
```

A build assembles the set in `releases/.building-<nonce>/`, renames it into
place, then writes `.complete`. `Store.Load`, `Verify` and the delivery
resolver all refuse a directory without the marker, so an interrupted build is
never activatable. Building identical inputs into a store that already holds
the release returns that release and renews its lease.

## inputs.json

`schema_version 1`. Fields: `builder{identity,node,user}`,
`source{commit,dirty,dirty_paths,snapshot:"working-tree",content_manifest_sha256,file_count,include_roots,excludes}`,
`bundle{file_name,sha256,size_bytes}`,
`dependencies{closure_digest,analyzer_*,scenarios,resources}`,
`native_cli{file_name,sha256,goos,goarch,size_bytes,package,module_dir,go_version,args,env}`,
`resource_artifacts[{component,platform,name,digest,mode,eligibility,license_refs}]`,
`toolchain{go_version,host_goos,host_goarch,go_flags}`,
`configuration{digest,schema,rule}`,
`credentials[{id,class,required,descriptor{logical_id,field},target_type,target_name}]`,
`provenance{policy,trust_mode,signer_key_id,signature_file}`,
`reproducibility{bundle,native_cli,reason}`, `limitations[]`.

Honesty rules the record follows:

- `source.snapshot` is always `working-tree`; the builder never claims an
  isolated snapshot. A dirty tree is recorded (`dirty`, `dirty_paths`, a
  limitation line) and the content manifest digest is the source identity.
- `reproducibility.native_cli` is `reproducible` only when the build was run
  twice and the bytes matched (`verify_reproducible`); otherwise `unverified`,
  or `not_reproducible` with the reason. The release always binds the first
  build's sha256.
- Credentials are descriptor references. Generators, prompts, formats and
  values are dropped; the canary test proves nothing leaks into any emitted
  file, the bundle's embedded manifest included.
- Every input that could not be pinned appears in `limitations`.

## Provenance and trust modes

`SCENARIO_TO_CLOUD_ARTIFACT_TRUST` selects `development-local` (default when
unset) or `production`; an unknown value is refused. The policy a release was
built under is recorded in `release-manifest.json` `provenance.policy` and in
`inputs.json`:

| Trust mode | Build | Verify |
|---|---|---|
| `development-local` | policy `development-local-unsigned`, explicitly recorded as a limitation | accepts an unsigned release that declares that policy; a signed release is still verified |
| `production` | policy `production-signed`; the manifest bytes are copied into `provenance/cloud-release-manifest.json`, pinned by a resource-deployment `release-manifest.json`, signed through the control-plane release authority (`vrooli release-authority sign --stage …`, argv seam `release.ArgvSigner`, injectable `release.Signer`), and verified against the trust anchor immediately; a signer failure leaves no release | requires the signed policy, a valid signature by `install/vrooli-release.pub` (or the configured key), and a signed copy byte-identical to `release-manifest.json` |

Signing the pin signs, transitively, every digest the manifest carries. No key
material ever enters this module.

## Verification

`release.Verify` runs, in order and without writing: `complete`, `manifest`,
`release_digest`, `bundle_sha256`, `platform` (when a target platform is
given), `native_cli`, `archive_bounds` (dry pass through
`binaryfetch.InspectArchiveBounded` with the release's declared `limits`), and
`provenance`. Failures are typed:

| Code | When | `details` |
|---|---|---|
| `release_verification_failed` (422) | any tamper or mismatch | `check` (step), `reason` (`release_incomplete`, `release_not_found`, `release_manifest_invalid`, `release_digest_mismatch`, `bundle_sha256_mismatch`, `native_cli_mismatch`, `archive_traversal`, `archive_symlink_escape`, `archive_too_large`, `archive_entry_limit`, `archive_unsupported_entry`, `archive_deadline`, `untrusted_provenance`, `signing_failed`, `untrusted_signer`), plus `declared`/`observed`/`artifact` where they apply |
| `unsupported_capability` (501) | the release's native control plane is not the target platform, or no control plane can be built for the requested platform | `artifact` (`vrooli-<goos>-<goarch>` or `native_cli`), `declared`, `required` |

The target repeats the digest, native-CLI and archive checks with the same
manifest before extraction (`vrooli cloud-target release verify`).

## Delivery

`vps/native_cli.go` no longer compiles anything. `release.NativeCLIForTarget`
resolves the binary beside the local bundle, checks the release is complete,
the platform matches the detected target, and the local bundle and binary
bytes match the manifest; delivery then copies the binary and compares the
remote `sha256sum` output with `native_cli.sha256` before `chmod`.

## Ownership and retention

Every built release holds an artifact lease (`releases/<digest>.lease.json`,
30 days, renewed by rebuilds and activations through `Store.Protect`).
`bundle.GCVPSBundles` merges the bundle digests of every leased release into
its protected set (`bundle.ReleaseProtectedBundleSHA256s`), so an active or
rollback-predecessor release is never reclaimed by the keep-latest rule; an
unreadable lease directory refuses GC instead of protecting nothing.

## API

REST bodies use the typed error envelope. `POST /releases/build` body:

```json
{"manifest": {…cloud manifest…}, "closure_digest": "sha256:…", "goos": "linux", "goarch": "amd64", "trust_mode": "", "verify_reproducible": false}
```

Responses carry `schema_version: "1"` and the stored `release`
(`release_digest`, `dir`, `complete`, `manifest`, `inputs`).
`POST /releases/{digest}/verify` accepts `{trust_mode, goos, goarch}` and
returns the report (`verified`, `checks[]`, `policy`, `trust_mode`,
`signer_key_id`); a refusal also carries the partial report in
`details.report`. The Connect `ReleasesService` exposes the same three
operations with proto field names.
