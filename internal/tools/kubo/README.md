# Pinned Kubo CLI owner

This owner acquires Kubo0.43.1 on Linux amd64 through the closed SDA
`source/kubo` route. The manifest pins the upstream55,646,028-byte tar.gz and
SHA-256. Native binaryfetch verifies the archive and stages extraction; this
owner checks the retained MIT/Apache notices against reviewed upstream bytes,
limits the extracted tree to256MiB, records its digest, and atomically publishes
an install-ledger-owned selection. It executes no upstream install script.

```sh
scenario-dependency-analyzer deps install source/kubo@0.43.1 --scenario kubo --surface host-tool --exact --json
scenario-dependency-analyzer deps install source/kubo@0.43.1 --scenario kubo --surface host-tool --exact --apply --json
vrooli host install kubo --dry-run --json
```

An unrecorded request is refused until normal SDA governance records the exact
review. Existing unowned or modified roots are refused; there is no silent
reinstall or PATH adoption. The runtime reuses generic manifest inspection but
the owner replaces ambient-command readiness with verified selection and
routes writes through SDA. Go callers obtain the exact executable and receipt
through `kubo.SelectedInstallation(ctx)`. No PATH launcher is installed by this
owner. The managed root is `~/.vrooli/opt/kubo`, with the executable at
`tree/kubo/ipfs` and receipt at `release.json`.

Acquisition has a three-minute context and shared artifact lock. Callers must
obtain fresh host admission and account for temporary/archive/extracted storage;
the256MiB check is post-extraction, not a separate disk-admission mechanism.
The upstream release digest is verified; reproducible-build or attestation
provenance is not claimed. Other platforms are explicitly unqualified.

Provisioning creates no repository, peer identity, daemon, listener, wallet or
public announcement. Caller-owned repositories and content remain separate.
Offline use must set a task-only `IPFS_PATH`, disable provider announcements,
avoid daemons and retain actual network/process/cleanup evidence. A CAR import
can pin its header roots rather than the externally requested directory; import
success alone does not prove a complete or retained recovery path. Inspect pin
errors, actual pins and retrieval after GC. Missing descendants are expected to
prevent an offline recursive pin of an incomplete DAG. Native offline trials also
confirmed that directory-root CAR import can exit zero with a nonempty per-root
`PinErrorMsg`; the initially readable path is then lost after GC. Automation must
check these semantic results before accepting retention. `--local-only` imports
without pinning, so it likewise needs an explicit retention strategy.

Validation: focused owner tests cover verified reuse/tampering, wrong checksum,
changed license, cancellation, unowned roots, unsupported platforms and ambient
command refusal. SDA tests cover the exact owner/version/source closure and
actual installer dispatch guards, including cross-owner mutations. Use the
shared generated-owner preflight and scoped Test Genie phases for consumer
validation; these tests do not establish public hosting or pinning economics.
