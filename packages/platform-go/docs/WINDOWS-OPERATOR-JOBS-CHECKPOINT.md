# Windows actual operator Jobs checkpoint

Owner: windows_operator_jobs. Isolated source only; no native token, LSA/logon,
device/setup/service/target execution or dependency acquisition.

## Active changed outcome

SCM per-role/virtual-account execution replaced by v2 actual-user operator-job /
operator identity. Explicit loaded HKEY_USERS SID hive persists frozen intent
before dispatch, root before resume and custody ACK before successful Start.
No HKCU/.DEFAULT fallback. Submission mutex is OS-thread-affine. Trusted wrapper
inherits image+Job via explicit HANDLE_LIST and JobList; SDK inherits only its
birth-locked image. EstablishWindowsOwnedServiceCustody runs early, validates
root/image/boot/account/frozen inputs/native Job name+DACL+limits+membership,
clears bootstrap env/inheritance and flushes ACK. CLI closes its temporary Job
lease on every return. Wrapper keeps custody; wrapper exit closes last handle
and kills descendants after external query leases close. Independent inspection
duplicates the exact held wrapper handle transiently and closes it.

Uncertain dispatch never replays or becomes healthy without custody. Suspended
birth before root/resume/ACK can retain an unacknowledged wrapper: canonical next
reconciliation is abort-only exact boot/UUID Job+DACL+limits, never PID dispatch.
Known roots additionally prove exact incarnation and birth image. Stop/release/
absence remain canonical and distinguish empty objects from absent namespaces.
Unknown exit stays unknown. Pure source-used submission gates have crash-stage
fixtures and reject partial/foreign custody.

Parent corrected agent-introduced arbitrary compromised-vendor threat model.
Full restricting-token/capability/privateDesktop implementation was NEVER run on
Windows; archived source/evidence under .vrooli/work/native-portability-history/platform/windows-restricted-authority-candidate-20261111 and removed from active API. PrepareOwnedImageChild now uses
standard CreateProcessW with actual non-elevated operator, exact argv/env/cwd,
existing no-breakaway Job and image-handle-only inheritance. Child actual user +
non-elevation independently observed preclaim, preresume and live/crash verify.
SecurityProof now means only canonical observed principal/non-elevation facts,
not least privilege/sandboxing; no token duplication is requested.

Windows same-operator peer preparation now verifies actualuser/nonElevation and
returns idempotent no-op cleanup; retired virtualaccount ACL-refcount machinery
is archived. Parent resource keeps actual HID/config/pinned image+DLL/finite PawnIO-omission/
listener/generation/once-claim safety. No account pivot or SeIncreaseQuota need.
windows_owned_backend.md documents selected route; no stale cap prerequisites.

## Verified

Current selected-contract receipts after own failure-mode maturation:

- Full platform-go module `go test ./...` PASS4.058s,
  /tmp/windows-operator-jobs-post-review-module.log.
- Focused actual-user/child/custody/payload/crash-stage fixtures with race PASS1.024s,
  named execution /tmp/windows-operator-jobs-post-review-focused.log; includes
  invalid UTF8 refusal and all intended five crash metadata stages.
- Six CGO0 module targets Linux/Darwin/Windows × amd64/arm64 plus Windows
  amd64/arm64 test binary compilation PASS,
  /tmp/windows-operator-jobs-post-review-crossbuilds.json.
- Windows amd64 go vet ./... PASS (tool receipt).

The filenames post-review refer to the owner's own failure-mode source passes,
not fresh independent adversarial approval. All siblings share mutable source;
these observations are not an exact-source isolation claim. Prior failures and
candidate work remain historical evidence. No native Windows execution occurred.

## Integration and remaining

Parent owns common v2 identity + persistent root/custody type/fact registry and
resource SecurityProof conversions. Cmd owner wires early custody before generic
capsule processing. Fresh independent source passes through parent still needed.
Native Windows runtime/token/HID/Qt/boot/logout/reconnect/physical qualification
OPEN, including last-close and each interrupted submit/ACK stage, loaded S4U
hive, console detachment and ancestor scheduler Job/detached executor behavior.
Source fixtures/crossbuilds never qualify hardware/platform behavior. No full
destination closure claim.

Fresh independent review is running under child canonical task
`/root/native_lighting_portability/native_owned_services/windows_operator_jobs/jobs_review`;
report destination docs/WINDOWS-OPERATOR-JOBS-INDEPENDENT-REVIEW.md. A completed
team slot was coordinated with parent before dispatch. Read-only source review,
no native effects; not yet accepted as an independent clean pass.


## Independent review repair — WJ-R1 through WJ-R4

First fresh reviewer found four material gaps (not a clean pass): insufficient
RegFlushKey handle access; visible ACK accepted before flush completion; empty
pre-Payload key stranded lifecycle; root-only UTF8 validation omitted child.
All repaired at the owning boundary. Source-used custody commit now opens
QUERY_VALUE|SET_VALUE, publishes pending ACK, flushes, then publishes its exact
post-flush Ready hash. Readers ignore pending visibility. Pure paused/failing
flush fixtures exercise that actual production ordering. Empty protected key
with no values/subkeys + native Job namespace absence is an explicit unpublished
state; read-only absence and mutex-protected cleanup/reprepare never replay a
published/partial value-bearing intent. Wrapper/child share lossless UTF8 input
validation, with child argv/env keys+values/path/cwd and valid Unicode fixtures.
Focused race PASS1.024s /tmp/windows-operator-jobs-independent-repairs-focused.log.
Full-module/crossbuild/vet reruns pending corresponding independent-repairs logs.
Same independent reviewer is rechecking repairs and possible regressions now.
These deterministic boundary fixtures still do not operate Windows native APIs.

After WJ-R1–R4 repairs: full module PASS4.056s,
/tmp/windows-operator-jobs-independent-repairs-module.log; six CGO0 builds,
Windows both-architecture test compiles and Windows amd64 vet all PASS,
/tmp/windows-operator-jobs-independent-repairs-crossbuilds.json. First reviewer
independently reran exact repair regressions with race PASS1.013s/no skips and
reports all findings addressed/no new material finding on repair recheck. It is
not a second fresh pass. A new fresh source challenger `jobs_final_review` is
running, report destination docs/WINDOWS-OPERATOR-JOBS-FRESH-PASS-B.md. Current
source has no outstanding known material owner finding; native Windows proof
remains OPEN and tests are source-only.

## Source handoff

Fresh pass B found WJ-B1: predispatch foreign observer SID could produce a
canonical-invalid receipt and strand public lifecycle. Repaired at the payload
boundary before effects: observer must be empty or actual account. Regression
freshly re-digests empty/same/foreign SID specs and requires every admitted
payload to create a canonical-valid receipt. Owner race PASS1.013s and independent
reviewer exact regression race PASS1.013s; all three cases executed, no skips.
Pass B report updated with disposition; no additional material finding on recheck.
Both original fresh passes were finding-bearing, so these are not two fresh
clean full-destination acceptance passes. All known findings are repaired.

Final source handoff validation: full module PASS5.433s
/tmp/windows-operator-jobs-handoff-module.log; final six CGO0 target module builds,
Windows amd64/arm64 test compiles and Windows amd64 vet PASS
/tmp/windows-operator-jobs-handoff-crossbuilds.json. Focused broader repair race
PASS1.024s is retained separately; observer repair exact race receipt
/tmp/windows-operator-jobs-pass-b-repair.log. Shared source applicability remains
mutable (no isolated source-version claim).

Source outcome ready for parent integration. Early wrapper custody and resource
child SecurityProof conversions were confirmed present by read-only source
inspection. Parent still owns persistent facts/schema classification, integrated
resource/cmd validation, fresh full-scope acceptance reviews and actual Windows
runtime/hardware qualification. No running test/reviewer resources remain.
