# Routed Test Storage

Mutating browser workflows run against a live scenario process only when its
leased test storage is installed and verified. This applies to both database
and filesystem writes; restarting a scenario is not an isolation mechanism.

The test-mode request header routes eligible persistence operations to the
lease. When either storage leg is unavailable, workflow validation refuses the
mutating case rather than falling back to primary storage. The lease teardown retains the counters its owner exposes as durable run evidence.
Unavailable counters remain unavailable; a cleaned lease does not account for
requests that never carried the routing header.

Use `storage-manager validate scenario <name>` before enabling mutating E2E for
a scenario. It verifies the routed database/file seams and namespaced external
stores. Use `vrooli scenario test <name>` for the server-owned suite; Test
Genie records whether the scenario source remained stable.

For workflow authors, use explicit execution-mode and reset labels. Observer
workflows must contain only read-only nodes; mutating workflows require the
routed isolation contract and are rejected if it cannot be proven.

Direct CLI, Connect and REST clients must propagate the header too. A test-mode
environment variable or routed-eligibility response alone does not prove that
the actual client routes its requests. Before its first mutating journey, verify
the current client transport forwards the header, using a captured request or
a fixture-only read through that same transport. An empty list shared by both
stores is not a routing discriminator. Repair an unsupported client before
writing; an untagged request follows the ordinary primary-store path.

Async work must retain the admitted operation's pinned context through queued
jobs, runner callbacks, event sinks, terminal persistence and finalization.
Detach producer cancellation with `context.WithoutCancel` and give the owning
job its own cancellation and timeout. Starting an admitted effect from
`context.Background()` drops its storage pins; copying only the test-mode flag
also loses the exact lease generation. After clear, replacement or expiry,
the old operation must fail against its original pin rather than select a new
lease or primary storage.

At HTTP entry, register and wrap the handler with the same
`devrouting.Service`: use `RegisterWithFileRootsService` and that service's
`PinnedTestModeMiddleware`. The shared owner serializes pin capture with lease
install, clear and restart operations. The lower-level
`apihttp.TestModeMiddleware` only marks requests; it does not establish an
exact paired generation or refuse a missing lease.

Qualify those handoffs with actual leased SQL/file effects and an independent
primary-storage check. Routing counters cannot count an async effect whose
context lost the test-mode marker. A static storage-analysis pass does not
prove the runtime handoff. Notification delivery needs its own isolation proof;
correct persistence alone does not partition WebSocket subscribers.

Chromium does not forward Playwright's `extraHTTPHeaders` on browser WebSocket
upgrades. BAS therefore adds `__vrooli_test_mode=1` to new WebSocket URLs in
fresh browser contexts explicitly configured with `X-Vrooli-Test-Mode: 1`.
The shared API marker accepts this query only for GET WebSocket upgrades under
the same development/explicit CI guard as the header. It grants no authority;
the paired routing owner must still pin the exact live SQL/file lease before
upgrading. Ordinary HTTP queries, malformed/duplicate values, and production
requests cannot opt into test routing through this query. Existing sockets in
attached application contexts are not retroactively reclassified; those
contexts do not establish a routed browser journey through this mechanism.

## Planned graceful restart qualification

The existing install/heartbeat/clear protocol does not restore a lease after a
target-process restart. Reinstalling with the same identifier is not evidence
that the original file tree survived. A restart journey needs an explicit
shared-owner checkpoint and resume operation; until implemented and qualified,
do not simulate that result with a new root or a copied fixture tree.

The bounded protocol is for a quiescent graceful restart. The run owner stops
test producers and verifies its native jobs have terminated, prepares an
opaque owner-issued ticket and counter checkpoint, uses the supported scenario
lifecycle to stop/start the target, and explicitly resumes only after the old
process exits. The active Test Genie session retains cleanup responsibility.
No cleared or expired lease can be revived.

The ticket binds the exact leased database, owner-created file roots, lease
expiry and target identity. Resume accepts no caller-authoritative root path,
checks the retained tree and bindings, and installs a fresh runtime generation
over the same persisted data. A same-ticket retry for the same already-resumed
live binding returns its prior receipt; a second attachment or post-clear
replay is rejected. Prepare, resume and cleanup are serialized. The checkpoint
has an owner-managed durable locator; the target's new process cannot depend
on the old process's in-memory root path. Resume on the preparing process is
rejected. The harness verifies actual old-process exit through the lifecycle
owner; a changed nonce alone is not process-exit proof.

Ordinary same-ID install retains its existing replacement/generation semantics;
it is never a restart-resume operation. The earlier planned idempotent-install
change was dropped after review of existing pinned-context guarantees. Keep
those guarantees and use the dedicated resume operation. The validation harness
may invoke this shared owner directly under the still-live Test Genie session;
Test Genie itself does not need to restart. Owner clear must succeed before the
harness invokes Test Genie resource cleanup. A failed owner clear retains the
data and retry handle. Test Genie now retains exact cleanup ownership durably, revokes expired
consumer leases, proves native seed exit before clear, and retries failed clear
or backing retirement. This does not make a target restart ticket implicit or
restore target-process counters after an unrelated crash.

Retain pre-restart and post-resume counters separately with their observation
windows; they are not proof of an atomic global drain or crash-time writes.
Read and compare actual revisions, source bytes and outputs after resume.
Submitting a replay remains a routed mutation. This protocol does not provide
automatic crash recovery or alter ordinary primary routing.
