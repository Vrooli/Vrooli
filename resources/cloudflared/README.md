# Cloudflare Tunnel connector

The control plane owns this managed connector, its logical credential file and
its checksum-pinned release in `resource.json`. Host requirement setup delegates
to Scenario Dependency Analyzer and the same resource artifact owner. An arbitrary
`cloudflared` on PATH does not satisfy the pinned installation.

Stage the declared release through the dependency owner:

```bash
scenario-dependency-analyzer deps install source/cloudflared@2026.10.7 --scenario cloudflared --surface host-tool --exact
scenario-dependency-analyzer deps install source/cloudflared@2026.10.7 --scenario cloudflared --surface host-tool --exact --apply
vrooli resource status cloudflared --json
```

Staging verifies the release and preserves the running connector. The status
record under `resource` distinguishes `declared_artifact_version`,
`staged_artifact_version` and `running_artifact_version`. A missing target image
appears as `artifact_staging_error` without hiding an independently verified,
healthy retained service. `artifact_update_available` is update intent, not a
service outage.

The Linux and Windows download digests come from the [official release
assets](https://github.com/cloudflare/cloudflared/releases/tag/2026.10.7).
The macOS archive digests authenticate the downloads; separate manifest digests
also authenticate the extracted executables. The client uses the retained
[Apache 2.0 license](LICENSE). Cloudflare account/service terms are separate.
Windows ARM64 has no declared native upstream asset; the desktop route declares
AMD64 only. Artifact integrity does not claim runtime qualification on every OS.

Recoverable rolling promotion, retained launch-contract rollback and native
interruption reconciliation are the target in
[the resource deployment contract](../../docs/resources/deployment-contract.md#target-recoverable-updates-for-stateless-managed-services).
Those operations are still being implemented. Staging success does not claim
that the new release is serving or that remote terminal continuity is proved.

The declared memory and disk footprint is a manually authored allocation estimate;
it is not a live benchmark result.
