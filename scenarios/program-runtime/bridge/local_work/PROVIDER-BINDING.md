# Concrete provider and immutable binding candidate

7 November 2026. Implementation continues; activation is not approved or attempted. Main remains `Enabled: false` with no runner/provider. No authority values were retrieved, credentials created, grants expanded, listeners opened, mounts changed, subscriptions created, or source/cloud calls made.

## Selected delivery boundary

The candidate `localworkbridge.SealedProvider` consumes already-delivered anonymous Linux memfds from a trusted lifecycle launcher. It has no credential-store client, discovery, environment lookup, filesystem credential path, network call, credential minting, refresh, or fallback. A launcher must first obtain the specific action-time approval and use Vrooli's credential authority to deliver only the reviewed capabilities.

The constructor requires originals to be close-on-exec, duplicates descriptors with close-on-exec and validates their bounded size, anonymous tmpfs identity, four permanent kernel seals, distinct inode, exact purpose set, authority identity/version and scope-digest metadata. After every descriptor validates, successful construction consumes ownership and closes all originals; failure leaves originals with the caller. It never reads credential bytes. Disabled construction does not inspect inputs. Resolve returns a deep copy of non-secret config only. Credential reads are lazy, purpose-specific, at most 8 KiB, refused after expiry or provider closure, and returned as a fresh caller-owned buffer. The runner's separate operation allowlist still applies. Every delivery rechecks current binding and confinement; no generic credential API is exposed to Python.

- `native_webhook_verify`: one dedicated 32-byte local subscription HMAC key.
- `events_reader`: one bounded Bearer capability restricted at the owner to selected event identity/type/source reads.
- `runtime_collector`: a distinct Bearer capability restricted at the owner to the pinned snapshot declaration, fixed inputs, admission observation and that principal's admitted handle.
- `flywheel_submit`: one dedicated 32-byte cloud signing key, present only when export is explicitly enabled.

The descriptor's Identity, Version and ScopeSHA256 are required launcher metadata, not an independent validation of the underlying token's claims or server-side grant. A broad token with bridge-specific metadata remains broad and is forbidden. A trusted launcher must verify the active authority version, owner principal, audience, exact operation/target scope, expiry and revocation before delivery. No production launcher or live scope receipt exists yet.

## Why the existing authority is not called directly

`packages/credential-authority-go` is the canonical Vrooli custody boundary. Its direct Resolve method has no context/deadline and its backing keyring may block. `credentialclient-go` checks context before hydration, but resolves synchronously, uses caller environment maps and requires separate lease cleanup. Those are useful existing authority facilities, but are not a cancellation-cooperative implementation of this bridge's private-pipe provider. The Secrets Manager broker keeps values within an origin/operation-bound executor, but does not currently expose this native webhook-HMAC verification + local collector + cloud envelope-signing operation as one supported contract.

The selected boundary therefore separates authority delivery from the finite bridge worker. This is not an authorization to create another secret store. The launcher must use the existing authority, keep delivery in memory, revoke its delivery lease on stop/failure/restart, and retain no extra original copies after successful transfer. The constructor consumes/closes supplied originals and the bridge owns its duplicates and closes them after joined Host.Stop. Expiry prevents further reads; it does not erase bytes already delivered to a running process. The owner must stop and join the child before claiming revocation of that exposure. Authority revocation is not independently polled or detected by this provider; the lifecycle owner must propagate it to Host.Stop, join, provider Close and authority lease cleanup.

## Immutable execution and private storage

The selected target is a dedicated lifecycle-owned bridge process in a reviewed, read-only SquashFS or EROFS whole-root image, with a dedicated non-root UID, isolated PID namespace, recursively private mounts, no capabilities (including bounding/ambient sets), NoNewPrivs and a non-dumpable Go parent. Parent non-dumpability must be established in the running post-exec Go process before credential delivery: setting it only before exec is insufficient because exec can reset it. The provider checks PR_GET_DUMPABLE==0 and never changes that setting. The launcher verifies the exact image digest before launch and supplies the observed root/state device+inode identities directly in approved configuration. The provider compares those local identities to its current namespace. ImageSHA256 records the reviewed image identity; the provider does not independently hash the backing image or prove the launcher checked it.

The complete interpreter, standard library, dynamic loader/libraries, CA/DNS configuration, seven bridge modules and release metadata must come from that image. A venv symlink into the user's uv cache is not acceptable. Fixed paths:

- `/runtime/bin/python`, remaining runtime closure under the image, prepared lock at `/runtime/requirements.lock`, sentinel under `/runtime`.
- `/app/local_work/ipc_entry.py` and the exact existing seven-module map.
- `/state/local-work-outbox.sqlite3`, existing private directory owned by the dedicated UID with mode 0700.

The only additional mounts accepted are read-only proc at `/proc` with nosuid/nodev/noexec, and read-write `/state` with nosuid/nodev/noexec on local ext4/xfs/btrfs/tmpfs. Additional mounts (including a writable application/stdlib submount), shared/master/other optional propagation tags, network/FUSE state storage, wrong UID, capabilities, writable root, changed mount identity, a dumpable parent, or interpreter symlinks outside `/runtime` fail closed. Mount metadata is validated before touching the state path. The state mount is identity-bound and writable, not immutable; its mountpoint parent is the immutable root. SQLite file protections and owner/permission checks remain in place.

This checks an already-created isolation boundary. It does not create mounts, namespaces, UIDs, permissions or a sandbox. Root/image provenance, dedicated UID/PID-namespace and post-exec non-dumpability setup, recursively private mounts, and server-side credential scope still belong to the trusted launcher. Linux file operations and memfd page faults cannot be forcibly interrupted by a Go context; pathological kernel/storage stalls are an explicit availability limitation. The owner must not launch replacements while the previous call is unjoined. No claim of hard-real-time cancellation is made.

The current shared Program Runtime host is not such a process. Its mutable checkout and user-owned uv Python tree cannot satisfy this provider, and this provider must not be attached there and described as immutable. Production integration requires either a dedicated bridge helper/sidecar under the existing lifecycle owner, with the callback/admission boundary preserved, or a separately reviewed isolation adapter. The existing generic Program Runtime API must not be silently migrated into a minimal filesystem or given extra mount exceptions to force this candidate to pass.

## Source authorization gate, verified in code

The current Python owner client sends Authorization Bearer. Shared api-core authentication supports verified principals and operation grants, but `RequireCapability` is explicitly a coarse check and domain object/operation scope remains the owner's job. Program Runtime's declared admission ownership uses verified agent provenance; it does not by itself prove a Bearer-only service can call only this declaration and these inputs. Events' query path is likewise not evidence of a dedicated selected-event reader capability. A source-side grant adapter using the canonical evaluator, or an existing equivalent owner contract, must be implemented and negatively tested before minting any token.

Required negative tests: wrong event source/type/ID, unselected query/list range, other collector name/digest/inputs, another principal's handle, unrelated runtime RPC, expired/revoked capability, and attempted mutation. Do not substitute a user session, fleet key, incidental MCP token, arbitrary live agent token or broad scenario write scope.

An inactive source-policy core and canonical exclusive-audience verifier mode now implement and test these decision checks; see [source policy and remaining owner gates](SOURCE-POLICY.md). The core is not registered or connected to a live binding authority. It does not close generic/anonymous source-route bypasses, persist service ownership, or provide continuous authenticated event selection. Source capability issuance remains blocked on that production integration, not merely on approval.

Evidence inspected: `packages/credential-authority-go/authority.go`, `packages/credentialclient-go/inprocess.go`, `scenarios/secrets-manager/docs/reference/agent-runtime-contract.md`, `packages/api-core/authn/operation.go`, `scenarios/program-runtime/api/internal/programs/admission.go`, Program Runtime main and library handlers, Events main/routes, and bridge owners.py.

## Alternatives

1. Extend the existing Secrets Manager trusted broker with the exact bridge operations, keeping raw values wholly inside the broker. This reduces material exposure but requires an owned broker/transport change and the same scope proof.
2. Supply a lifecycle-prepared immutable worker bundle through a reviewed launcher adapter while retaining the existing Program Runtime host. It must pin the full runtime closure and private storage across child execution; chmod or hashes against the shared checkout alone do not qualify.
3. Use a full digest-pinned OCI image under an approved runtime with read-only root, dedicated UID and scoped mounts. Overlayfs is not accepted by this candidate verifier; supporting it needs a separately reviewed immutable-image attestation, not relaxing the check to any read-only overlay.

None of these alternatives requires activating the present candidate to continue development.

## Exact remaining work and action-time approval

Before asking for activation, implement and offline-test the source-side least-privilege grant adapter, trusted authority-to-sealed-FD launcher, immutable image assembly/manifest and image proof, isolated worker routing/lifecycle, storage preparation/checkpoint migration and revocation/recovery behavior. No provider, launch service or authority capability is installed by this work.

Then obtain action-time approval naming the real launcher/service owner and callback, actual image/config/lock/module digests and dedicated UID/storage path, source instance `swarminator` and a commissioned epoch, the three selected agent-manager events (`agent_manager.run.notification.v1`, `agent_manager.run.decision_requested.v1`, `agent_manager.run.decision_withdrawn.v1`), fixed collector digest/inputs, exact authority identity+version+server-enforced scopes and expiry no more than seven days, local webhook key placement, cloud signing key placement and private Site access, the exact destination `https://decision-inbox-experiment.mdhalloran.chatgpt.site/api/bridge/local-work-events`, allowed metadata, subscription, service enablement/deployment and named synthetic/real transmissions. Explain the credential/persistent-access expansion and relevant security settings. Scope/access setup, sandbox provisioning and credential creation remain gated; continuing code is not that approval.

After approval, qualify the joined real runtime/provider and source → queue → cloud receipt → journal → handler pickup → source-backed many-to-many reconciliation → readback chain. Require two separated later authorized changes without manual pickup. Preserve permission/coverage/checkpoint holds, pause/mute, and valid local-only/cloud-only work. Component tests and a 204 receipt are not acceptance of sustained reconciliation.

## Qualification scope

New tests cover disabled no-I/O, constructor/Resolve aliasing, descriptor duplication and caller close, sealed/anonymous/bounded/distinct descriptors, missing/extra purposes, invalid metadata/config/layout, expiry/cancellation/provider closure, lazy malformed material refusal, explicit export, mount closure and process confinement. They use public RAM-only fixture bytes. The isolation verifier is replaced by a package-private fake in positive credential tests; parser tests cover confinement metadata, not an actual provisioned namespace. A real finite child-exec regression checks descriptor noninheritance, not proc reopening or actual namespace provisioning. No credential authority, real mount, live source or cloud endpoint is used. Retained results and exact source hashes are in `.vrooli/evidence/local-work-bridge-20261106/provider-binding-20261107/`.

## Confinement continuation

The selected next architecture is a trusted finite source broker with a network-denied, credential-free worker; see [the confinement decision and owner gates](BROKER-CONFINEMENT.md). The source broker core is inert and unregistered. The existing sealed provider and Python credential pipe are not connected to it and are not proof of this boundary. Real launcher/fence/transport, trusted native selection and atomic service admission ownership remain required before source capability issuance or activation.
