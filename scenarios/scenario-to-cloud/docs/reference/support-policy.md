# Cloud deployment support policy

This policy states which target platforms, workload shapes and providers
scenario-to-cloud **certifies**, which it treats as **compatibility-only**, and
which are **unsupported**. A claim is certified only when the named matrix
cells hold current, retrievable evidence. Source existence, a passing package
test or a green aggregate status never upgrades a classification.

Frozen for plan `scenario-to-cloud-professional-vps-delivery-certification`
(phase 1, 2026-09-09). Re-freeze before qualifying a release candidate.

## Target platforms

| OS | Architecture | Classification | Required evidence | Notes |
|---|---|---|---|---|
| Ubuntu 24.04 LTS | linux/amd64 | **Certified (launch baseline)** | QEMU fresh-host lane + authorized real-VPS lane, minimal fixture, all mandatory matrix rows for the lane | Preflight recommends 24.04; native CLI artifact exists for amd64 |
| Ubuntu 24.04 LTS | linux/arm64 | **Certified (launch baseline, evidence pending)** | Same as amd64 on an arm64 lane | Native CLI artifact detection exists; per-resource artifact eligibility is checked per closure, not assumed |
| Ubuntu 22.04 LTS | linux/amd64, arm64 | Compatibility-only | Preflight warning; no certification cells | Older package baselines; not exercised by the certification matrix |
| Ubuntu 20.04 LTS | any | Compatibility-only (deprecated) | Preflight warning | End of standard support; no certification |
| Debian 12 | any | Unsupported | none | Not exercised; preflight reports unsupported distribution |
| Other Linux, macOS, Windows | any | Unsupported as a VPS target | none | macOS/Windows nodes are Bridge targets for other ramps, not cloud targets |

## Workload shapes

| Shape | Classification | Fixture | Notes |
|---|---|---|---|
| Minimal stateless web scenario | Certified | `fixtures/workloads/stateless-web` | No persistent data bindings |
| Headless API scenario | Certified | `fixtures/workloads/headless-api` | No UI surface |
| Durable SQL + file uploads | Certified | `fixtures/workloads/sql-uploads` | Postgres binding + object-store binding; backup/restore required |
| Scenario with a scenario dependency that owns its own resource and credential | Certified | `fixtures/workloads/dependent-chain` | Closure must include transitive requirements with reasons |
| Shared dependency across two deployments | Certified | two-deployment coexistence fixture | Lifecycle demand must preserve the survivor |
| Resource-heavy profile (GPU, large memory) | Compatibility-only | `fixtures/workloads/resource-heavy` | Certified only for the explicit unsupported/insufficient-capacity refusal path |
| Representative hosted commercial backend (landing-page-business-suite) | Certified as an additional consumer | existing deployment declarations | Never the only fixture; billing, desktop and mobile are out of scope |
| Any scenario satisfying the published declaration contract | Supported ("deploy any scenario") | none | Exact unmet requirements are returned for all others |

## Providers and transports

| Concern | Classification | Notes |
|---|---|---|
| Machine reach via vrooli-bridge / nodereach | Implemented path; live qualification pending | Enrollment, grants, typed dispatch, Bridge-owned artifact delivery |
| Direct SSH through the shared bounded reach adapter | Supported with explicit transport selection | Policy-equivalent authorization; never a silent fallback after revocation |
| Cloud-owned raw SSH inventory | Unsupported (retired) | See deletion ledger |
| Cloud provider APIs (DigitalOcean, Hetzner, AWS, ...) | Out of scope | VPS provisioning is the operator's responsibility |
| Kubernetes, multi-region | Out of scope | Rejected alternative until core certification exists |
| DNS providers | Observation only | Cloud verifies DNS; it does not mutate zones outside an authorized delegated zone |
| ACME | Staging for repeated tests; production issuance bounded by external authority | Never disable certificate verification |

## Numeric budgets (minimal seeded fixture)

| Measure | Target |
|---|---|
| Acknowledged-write loss on healthy compatible activation | 0 |
| Planned maintenance unavailability | at most 60 s (stateful workloads declare their own bound) |
| Fresh-host restore time (RTO) | at most 15 min after target available |
| Backup recovery point (RPO) | at most 5 min for the configured protected-write stream |
| Health/alert detection | at most 60 s after threshold met |
| Stale-fence commits | 0 |
| Synthetic-secret leakage occurrences | 0 |
| Soak | at least 24 continuous hours on the release candidate |
| Residual owned resources after cleanup | 0 unexplained |

Machine profile for these budgets: 2 vCPU, 4 GiB RAM, 40 GiB disk, single
public IPv4 (IPv6 optional). Idle management overhead budget on the target:
at most 5% CPU and 300 MiB RSS for control-plane processes; API p95 for
read endpoints at most 500 ms under the fixture load of 20 concurrent readers.
A small sample must be reported as such.

## Supervision and restart

Supervision membership (which processes the runtime supervises) and
auto-restart (whether a supervised long-running process is restarted) are
independent controls. Cloud passes both through the setup/v1 selection and
never derives one from the other.

## Compatibility policy and evidence freshness

A classification above is a claim about a **candidate**, and a candidate is
an exact identity: `release_digest`, `configuration_digest` and
`closure_digest` (`docs/reference/release-identity.md`). Evidence is bound
to that identity by `api/certification` receipts (`candidate{…}`,
`target{os, architecture, machine_id}`, `lane`, `observed_at`). Nothing
below is judged by age alone; it is judged by whether the thing the receipt
describes still exists.

### How evidence ages

| Receipt lane | Bound to | Becomes **stale** when | Becomes **unavailable** when |
|---|---|---|---|
| `package` (in-process, fake target) | the source tree that produced the candidate | the candidate digests change (a new release, configuration or closure) | never; it is re-run by `go test` |
| `qemu` (fresh disposable guest) | candidate + image digest (`certification/lanes/qemu-images.json`) + architecture | the candidate digests change, or the approved image digest changes | the host tools or image are absent (EXT-06); recorded honestly as `verdict: unavailable`, never as passed |
| `real` (authorised disposable VPS) | candidate + target identity (`machine_id`, OS, architecture) + delegated DNS zone | the candidate digests change, or the target is re-provisioned (new `enrollment_generation`) | no authorised target exists (EXT-01) |
| `bas` (browser journeys), `soak`, `review` | candidate + the UI build / the soak window / the reviewer identity | the candidate digests change | the external input is missing (EXT-07..09) |

Rules the readiness computation applies (`api/certification`, tests in
`certification_test.go`):

1. A required (case, lane) cell with no receipt is `missing`; a receipt for
   a different candidate is `stale`; an `unavailable` receipt is counted as
   unavailable, not as passed. Any of the three makes the package
   `not_ready` and the report names the cells.
2. Receipts never expire by wall-clock alone, but every receipt outside the
   `package` lane expires **with the release**: a new candidate has no real,
   QEMU, BAS, soak or review evidence until those lanes run again against
   it. A "small change" is still a new candidate.
3. A live observation (`health/observation`, `edge status`) is evidence of
   the moment it was taken (`freshness: current|stale|unknown`); it never
   certifies a release and a stale observation is never healthy.
4. Evidence is retained for at least the receipt retention budget
   (`certification/budgets.json`: 90 days) and for as long as the release it
   binds is active or retained for rollback, whichever is longer.

### Version support windows

| Component | Window | Notes |
|---|---|---|
| Certified target platform (Ubuntu 24.04 LTS) | while the distribution is in standard support and the platform row above says Certified | a platform leaves the certified row before the distribution's own end of standard support if a required lane cannot run on it any more |
| Compatibility-only platforms (Ubuntu 22.04, 20.04) | best effort; preflight warns; no certification cells | may be moved to Unsupported at any re-freeze |
| Native control-plane binary on the target | the release-bound binary is the only supported one; it is replaced by the next release's binary at activation | mixed versions on one target are never supported (`release.verify` refuses a mismatched CLI) |
| Executable plan schema (`schema_version`) | additive changes keep the version; a semantic change bumps it and a reviewed digest from the old version is `plan_stale` | old digests are never silently re-interpreted |
| Wire contracts (proto `vrooli.scenario_to_cloud.v1.*`, REST envelope, exit codes) | additive within v1; the CLI negotiates the server version and refuses an incompatible one (`docs/reference/identity-and-selectors.md` §"Version negotiation") | |
| Recovery points | readable by any release whose `schema_version` the point's schema strategy declares compatible (`docs/guides/runbooks/restore.md`) | `rollback_incompatible` is the refusal |

### What a re-certification requires

Any of these events starts a new certification of the candidate; none of
them can be absorbed by editing evidence:

- a new `release_digest`, `configuration_digest` or `closure_digest`;
- a change to the certification matrix (`certification/matrix.json`) or to
  a lane manifest;
- a change to the approved QEMU image digests or the certified platform
  rows above;
- a change to the numeric budgets (`certification/budgets.json`), which the
  validator refuses to drift silently;
- a re-freeze of this policy.

Re-certification is: run the deterministic lanes (`go test ./...` in `api/`
and `cli/`, the UI suite) for the candidate; run the QEMU lane journeys and
the authorised real-VPS journeys against the candidate; re-run the BAS
journeys on the candidate's UI build; complete the independent review and
the soak window for that candidate; then compute readiness
(`api/certification` report) and record the verdict in the release notes.
A candidate whose readiness is `not_ready` may be deployed to a
non-production environment; it is not "certified" anywhere in prose until
the report says `ready`.

Standing at the time of writing: the package lane is substantially covered;
every real, QEMU, BAS, review and soak lane is pending the external inputs
listed in the plan's `ledgers/external-inputs.md` (EXT-01..11), and the
current readiness report is `not_ready` (`docs/RELEASE-NOTES.md`).
