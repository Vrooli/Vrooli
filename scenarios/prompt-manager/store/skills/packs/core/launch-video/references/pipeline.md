# Multi-cut pipeline contract

`scripts/pipeline.py` coordinates already-produced, evidence-complete channel
plans through the existing plan validator, sequence packet audit, immutable
input freeze, managed HyperFrames owner, exact MP4 inspection, LV3 gates, and a
retained owner cleanup receipt. It is not a capture runner, tool installer,
creative reviewer, publication path, or landing-page writer.

The production manifest contract is schema 2; schema 1 lacks required plan/cue/sequence hashes and is refused. The pipeline handles all five launch-video products. The manifest lives beside
its source files and pins the source-map/source-packet and operator
offer-sign-off SHA-256 values. For Aquila, Mizar, Rigel and Pan, `source_map`
must be an LV4 `source-bound-production-plan` packet whose `product_id` matches
the manifest, and `source_packet_intake` must pin the accepted multi-product
planning receipt. The coordinator checks the receipt's exact hash, planning-only
status, current coordinator digest, and matching product packet hash before it
checks that route claims are in the packet's source-proof set and each cut's
plan has exactly the ordered claims declared by that packet's channel. Vega
may still use its earlier five-product source-map input, but a supplied LV4
packet requires and follows the same intake receipt and strict checks. The sign-off
JSON must state the matching product ID, `status: approved`,
`operator_approved: true`, and its exact `offer_ref` and `offer_revision`; every
cut plan must match that revision. This preserves human offer approval for
every run. A pending Markdown decision request fails as a non-JSON authorization
artifact. All input, snapshot,
bundle, and evidence paths stay beneath that directory. Each cut names a
version-3 production plan, cue sheet, sequence manifest, snapshot/output paths,
composition directory, and exact SHA-256 identities for the plan, cue sheet,
sequence manifest, Node, browser,
FFmpeg and FFprobe. It may also name the complete LV8 bank/catalog/selection
triple; music-mode plans are rejected by `workflow.py` without this exact-hash
lint. The outer manifest binds a current source map or LV4 packet and operator
offer sign-off. Vega supports its three-cut LV2 slice or all four channels.
Aquila, Mizar, Rigel and Pan require all four unique landing, X, Reddit and
vertical cuts, matching LV4's packet contract.

For a read-only multi-product planning intake, run
`pipeline.py --intake-source-packets <LV4-packet-directory> --out <new-receipt.json>`.
This checks the five-product index, exact packet hashes, route/proof claim joins,
all four per-channel claim orders and script-content hashes, frozen source
files, and cited repository bytes/line ranges. Its receipt means only
`source-packet-set-accepted-for-planning`; `production_status` remains
`not-accepted`. It does not accept a production manifest, operator offer/CTA,
eligible media, snapshots, renders, or staging, and it performs no production
owner action. The production manifest path still requires a real exact-hash
operator sign-off and the full per-cut inputs below.

The deterministic schema-2 boundary mutation matrix is maintained at
`work/LV4/verify_pipeline_boundary_matrix.py`. It uses temporary packet sets,
tool identities and fixture-only approval artifacts to exercise both acceptance
and fail-closed mutations without calling production owners. Its receipt is
fixture/readiness evidence only, never a product film or operator acceptance.

Each fully validated packet cut produces a `<channel>-input-handoff.json`
before its plan validator runs. The handoff joins the exact manifest, source
packet and set-intake receipt, route, channel script hash and ordered claims to
the plan, cues, sequence and tool identities. Its status is only
`inputs-bound-pre-render`; it is not a render or creative acceptance.

Run without `--render` to validate and freeze each fully populated packet. Run
with `--render` only after every upstream source is current and eligible. Set
`VROOLI_BIN` to the exact installed managed CLI. For each sequential render,
`vrooli host hyperframes` verifies its managed installation and obtains its own
fresh RAM/CPU claim plus the 120-GiB-floor disk reservation before writing. A
denial stops the pipeline; there is no direct renderer fallback. Run a fresh
invocation after any upstream change; prior output directories are immutable.
If an owner command or coordinator step raises after a new output directory has
been created, the CLI retains a `pipeline.json` failure envelope with the
blocker and a bounded path/size inventory of partial artifacts. It reports
cleanup as unknown and never retries or deletes owner state. Existing output
directories are never overwritten to record a failure. Small partial artifacts
also carry exact hashes; large ones are listed by size without unbounded hashing.
The envelope binds manifest product, source map and packet-intake receipt
identities; owner-command failures include a classified phase, exact argv, exit
code and stdout/stderr hashes. It does not assert that a new capture, model,
service or render was cleaned up.

The status `machine-gated-unreviewed` means only that exact bytes were joined to
the frozen snapshot, decoded/probed, passed LV3, and carry the managed owner
receipt. It is not creative, audio-listening, comprehension, claim, offer, or
operator acceptance. Final delivery still uses `workflow.py verify` after
review artifacts exist.

Landing Page Business Suite supports native local MP4 embeds through its
`presentation-media` owner. After each cut passes managed rendering, exact-byte
inspection and LV3 gates, the pipeline records a `ready-for-admin-stage`
handoff with the MP4 SHA-256, upload route, size limit, local `media_ref`, and
private-preview route template. It marks the handoff ready only when the exact
export is 1–256 MiB and begins with the ISO BMFF `ftyp` signature required by
LPBS; otherwise it reports `blocked-media-constraint` and the measured size or
signature. An authenticated operator/admin must perform
the step-up upload (with fresh shared disk admission), configure the recorded
`product-demo` playback fields in the local-film-stage admin UI, and save the
private draft. The pipeline does not possess that session or perform the draft
mutation. Its top-level `staging` status is `admin-stage-required` when all
declared verified cuts have handoffs, `pending-render` before a render, or
`blocked` when a cut fails. The draft `media_ref` binds to the exact MP4 bytes;
public activation remains separate and is not performed.
