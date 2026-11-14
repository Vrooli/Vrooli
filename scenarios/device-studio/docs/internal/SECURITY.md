# Security — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19. The implementation status changed after this original contract; current bounded geometry execution is described under [Current implemented boundary](#current-implemented-boundary--2026-10-21). The full security requirements and review remain open.

Define the protection boundaries before implementing media import, renderer code execution or paid agents.

## Data Sensitivity

Projects can contain unreleased UI, customer data, source media, filenames and private references. Derived thumbnails, contact sheets, logs and prompts inherit sensitivity. Store originals locally by default. External processing requires per-project consent bound to the actual uploaded bytes and purpose. Public hardware reference availability does not automatically permit AI upload or redistribution.

## Auth And Authorization

Declare personal_local, local_multi_user, remote_vrooli or shared_provider mode explicitly. Personal desktop uses the private OS/app boundary without mandatory website login. Shared/remote use requires verified principals and project/object authorization. App-private supervisor tokens are not human identity. Scoped commerce account linking, entitlement verification and authorization to spend are separate from local access. Validate resource/audience/installation/account/scopes/expiry; never link by email equality or accept a balance from the browser.

Agent Manager owns attenuated identities; actual operation grants use the canonical evaluator. Human initiation/disclosure is not blanket child authority. Keep approvals human-only where their owner requires it. Release review is independent of run completion. Browser account adapters must use the owner-supported code/cookie flow; the researched Web Console fragment callback is a known mismatch and must not be copied.

## Secrets

Use credential authority/native secure storage for BYOK keys and relevant sessions. No tokens in URLs, localStorage, scene scripts, provider prompts, recipes, exports or logs. Workers receive scoped operation references or narrowly authorized capabilities through their owner; do not lend a human refresh token. Sign-out clears local connection material; remote revocation can remain pending and must be reported separately. Cached lease bytes require signature and full binding verification.

## Threat Model

| Entry | Threat | Required boundary |
|---|---|---|
| Model/archive/media import | Traversal, symlink escape, decompression/decoder bombs, malformed files, external URIs | Bounded extraction/decoding, dependency closure and controlled fetch |
| Reference URL/provider output | SSRF, redirects to private services, untrusted instructions | Authorized destinations, response/type/size limits; treat content as data |
| Imported/generated executable content | File/network/process access beyond project | Arbitrary scripts, Blender files with executable content, and model-authored commands are unsupported and are not executed. The supported fixed geometry recipe is compiled by a fixed Go worker under the bounded workspace-sandbox owner. |
| Agent creation | Scope expansion, undisclosed paid calls, media egress | Canonical delegated authority, aggregate budgets, server-side consent enforcement |
| Desktop loopback | Other browser/site or local process invokes app | App-private runtime authentication, origin checks and explicit shared mode |
| Render requests | Unbounded CPU/VRAM/disk/task use | Admission, owned descendants, quotas and cancellation |
| Package/update | Tampered executable or dependency drift | Trusted signed manifest, hashes, atomic staging and target-native evidence |
| Shared release | Wrong revision, incomplete rights, forged approval | Immutable revision review and owner publication gate |

## Security Gaps

The original 2026-10-19 status below is superseded in part by the implementation note. A fixed deterministic geometry compiler now runs through workspace-sandbox with write and network restrictions and resource limits. This is not general arbitrary-code isolation, OS-read confinement, archive sandboxing, or a security certification. Current Blender/native scene execution, paid-provider adapter, and full desktop security review remain out of scope or unqualified. No paid call or credential validation is claimed. Requirements and [PROBLEMS.md](PROBLEMS.md) remain authoritative for open work.

## Current implemented boundary — 2026-10-21

Device Studio supports a finite, versioned geometry recipe compiled by a pinned
Go worker. Inputs are staged in an attempt workspace and passed through the
workspace-sandbox public execution owner with a fixed command, manifest path,
working directory, and environment. Network access and writes outside the
workspace are denied by the selected protected profile. The worker does not
interpret imported or generated scripts, shell commands, URLs, model-authored
paths, Blender projects, or arbitrary archives. Those inputs are unsupported;
rejection before execution is the intended behavior.

The current compiler profile requests a 4 GiB virtual-address-space limit,
45 seconds CPU, 60 seconds wall time, 32 processes/tasks, 64 open files,
`GOMAXPROCS=2`, and `GOMEMLIMIT=256MiB`. `GOMEMLIMIT` is a soft Go garbage
collection target, not a hard memory/RSS cap. The address-space cap is not a
physical-memory limit. The remote sandbox compiler and Device Studio parent
operation are separately owned resource scopes; there is no aggregate RSS
claim. Local inference uses the separate typed Gateway/Ollama route and is not
inside the compiler sandbox. No hosted fallback is admitted by the bounded
local session.

The compiler sandbox enforces write and network boundaries, but this profile
does not establish OS-level read confinement. In particular, do not claim that
only project files are readable or that arbitrary malicious binaries are
safely contained. The supported executable is the fixed compiler, not user
code. General import/archive isolation, full adversarial security review, and
any future executable-content feature require separate owner-backed design and
evidence.

## Cross-References

[data](../concepts/DATA.md), [creation](../domains/creation.md), [monetization](../business/MONETIZATION.md), [acceptance](ACCEPTANCE.md).
