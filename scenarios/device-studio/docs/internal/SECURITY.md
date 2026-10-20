# Security — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

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
| Blender/native scene/generated Python | File/network/process access beyond project | Disable autoexec plus qualified isolation; resource limits alone insufficient |
| Agent creation | Scope expansion, undisclosed paid calls, media egress | Canonical delegated authority, aggregate budgets, server-side consent enforcement |
| Desktop loopback | Other browser/site or local process invokes app | App-private runtime authentication, origin checks and explicit shared mode |
| Render requests | Unbounded CPU/VRAM/disk/task use | Admission, owned descendants, quotas and cancellation |
| Package/update | Tampered executable or dependency drift | Trusted signed manifest, hashes, atomic staging and target-native evidence |
| Shared release | Wrong revision, incomplete rights, forged approval | Immutable revision review and owner publication gate |

## Security Gaps

No product import sandbox, runtime job isolation or funded desktop adapter is implemented yet. Current template middleware is infrastructure only. Current Blender provisioning and desktop local-proof transports need qualification. No live credentials/spend/security certification occurred during the research. Required behavior remains visible in requirements and [PROBLEMS.md](PROBLEMS.md).

## Cross-References

[data](../concepts/DATA.md), [creation](../domains/creation.md), [monetization](../business/MONETIZATION.md), [acceptance](ACCEPTANCE.md).
