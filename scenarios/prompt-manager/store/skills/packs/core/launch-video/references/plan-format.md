# Production file contract (version 2)

Requires Python 3.11+; media operations also need existing ffmpeg/ffprobe.

Run `python3 <skill-dir>/scripts/production.py validate production.json --stage plan`.
Paths resolve beside `production.json`. Helpers never start services or render a
composition. `plan` checks planning inputs; `polish` additionally probes source
media and requires flagship styleframes/animatic; `deliver` checks actual export,
review identities, revisions, and real product proof. Any failure exits nonzero.

This is a small evidence/timing manifest, not a renderer abstraction. Keep camera
implementation, layers, transitions, and easing in the composition brief/code.
Keep the landing-page source-to-video mapping in `composition/DESIGN.md`, with
rendered references in `visual-reference/`. Record the visual-continuity verdict
there or in the styleframe review. This is a required reviewer checkpoint in
`production.md`; the file helper does not certify visual resemblance.

Required shape:

```json
{
  "version": 2,
  "kind": "launch",
  "product": "scenario-key",
  "profile": "teaser",
  "duration": 24,
  "format": {
    "width": 1920,
    "height": 1080,
    "fps": 30
  },
  "safety": {
    "status": "verified",
    "report": "safety-report.md",
    "protected_mutations": []
  },
  "story": {
    "promise": "One supported benefit",
    "viewer": "One audience",
    "problem": "The starting problem",
    "transformation": "The observable change",
    "proof": "The decisive recorded moment",
    "proof_scene": "result",
    "takeaway": "What remains in the viewer's mind",
    "viewing_context": "Muted product-page player on desktop and phone",
    "category": "Supported product category",
    "setup": "Context needed before the first product interaction",
    "next_action": "Explore the product",
    "footage_balance": "Real interaction proves the promise; conceptual opening supplies context"
  },
  "claims": [
    {
      "id": "c1",
      "source": "source command/path + revision",
      "read_at": "ISO-8601 timestamp",
      "approved": "Source meaning or locked text",
      "wording": "Planned paraphrase",
      "qualifiers": "none",
      "availability": "verified current scope",
      "exact_copy": false,
      "verified": true
    }
  ],
  "audio": {
    "mode": "music",
    "source_kind": "governed-generation",
    "source_id": "Actual selected take ID or retained-source hash",
    "direction": "Sound and arc",
    "selection": "audio-selection.md"
  },
  "scenes": [
    {
      "id": "result",
      "start": 0,
      "duration": 24,
      "class": "evidence",
      "purpose": "Prove c1",
      "adds": "Visible consequence",
      "claims": [
        "c1"
      ],
      "focal_point": "Result region",
      "motion_reason": "Make the result readable",
      "start_state": "Known state",
      "action": "Recorded user action",
      "end_state": "Visible result",
      "text": [
        {
          "value": "A readable result",
          "hold": 2
        }
      ],
      "media": "footage/result.mp4",
      "source_sha256": "actual SHA-256",
      "capture_instance": "scenario@presentation",
      "capture_date": "ISO-8601 timestamp",
      "privacy_review": "footage/result-review.md",
      "product_motion": true,
      "crop": [
        0,
        0,
        1920,
        1080
      ],
      "max_display": [
        1600,
        900
      ],
      "visual_intention": "Readable action and result in a clean browser frame",
      "headline": "Message search",
      "meaning": {
        "feature": "Search messages",
        "sample_input": "example query",
        "observed_result": "Actual matching messages",
        "viewer_inference": "Find prior decisions"
      },
      "proof_kind": "interaction",
      "frame": "Clean thin browser shell; controls unobscured",
      "cursor": "Deliberate pointer preset from observed events",
      "settle_evidence": "Fonts and diagram loaded before take",
      "trim": "Source seconds 1\u201325; waiting compressed only where recorded"
    }
  ],
  "video": "video-v2.mp4",
  "poster": "poster.jpg",
  "technical_report": "inspection/inspection.json",
  "review": "quality-review.json",
  "review_status": "draft",
  "cta": {
    "action": "Explore the product",
    "url": "https://vrooli.com/apps/verified-product",
    "expected_content": "The intended product page",
    "availability": "Verified current availability",
    "destination_report": "destination.json",
    "qr": true,
    "qr_report": "qr-v2/qr.json"
  },
  "render_inputs": [
    "composition/index.html",
    "composition/assets/result.mp4",
    "composition/assets/music.wav"
  ]
}
```

The example describes shape, not a recommended 24-second single shot. Scene slots
partition the full duration with no gaps/overlaps; transitions occur inside those
slots. `text.hold` is settled reading time, excluding entrances/exits. `crop` uses
actual encoded source pixels; `max_display` includes the largest zoom, not CSS size
before scaling. Evidence and presentation scenes require faithful product source media. A state
hold or still is valid; interaction claims need an observed action and result.

Flagship additionally supplies `directions` (at least three objects with `id`,
`structure`, `shot_language`, `audio`, `rationale`), `chosen_direction`, `styleframes`
(objects with `role`, `file`, `review`; roles opening/hero/detail/ending), `animatic`,
`animatic_review`, and `revisions` (objects with `before`, `after`, `notes`, `defect`).
A no-defect first cut requires `no_revision_reason` and `second_review` instead.

Each review JSON uses `sha256`, `reviewer`, `method`, `visual_observations`,
`audio_observations` ("not applicable: still image" is valid), `verdict` (pass/fail),
and `defects` as defined in `audio-review.md`. Human/agent review is evidence, not
an automated saliency or truth score. Review substance remains subject to critique.
The inspector never authors a passing review.

For safety-blocked capture, set `kind: study`, `safety.status: capture-blocked`, and
use only explanatory scenes. `polish` also requires an explicit study request;
a blocked launch is not that request. `deliver` will
fail because a study is not a launch demonstration. Keep this failure in the report;
do not relabel the study or manufacture evidence to turn the gate green.

Inspect a delivered file:

```bash
python3 <skill-dir>/scripts/production.py inspect video-v2.mp4 --out inspection-v2 --at 0,1.2,3.9,4.1,8,12,18,23.9
```

The destination must be new. Reuse inspection output only for the exact same file
hash. Keep intermediate renders, measured diagnostics, and alternatives.

Final reviews declare `audio_status`: `auditioned-pass`, `unreviewed`, `fail`, or
`not-applicable` (silent/still). Music/voice delivery requires `auditioned-pass`.
Measured loudness and technical decode cannot satisfy listening review.

## Capture and audio provenance

Launch plans require decisive genuine product proof and an explicit footage-balance
rationale, without a fixed duration quota. Studies require `study_requested: true` and `request_source` to proceed
to polish. A blocked launch request does not imply a study request.

Audio adds `source_kind` (`governed-generation`, `licensed-reuse`, `original-score`).
An original score needs `original_score_requested: true` and `request_source`.
For version2 with `variety_requested: true`, `audio.variety_rationale` explains
what differs between the actual candidates/films and which observations remain
unreviewed. Differences may be rhythm, bass, texture, arrangement or style within
the requested energy range. `candidate_styles` may retain descriptive annotations;
there is no distinct-family quota. Neither different seed IDs nor different genre
names establish audible variety. The rationale cannot replace actual audition or
make an unheard selection ready for delivery. Silence is a timing rough unless
explicitly requested.

For operator-authorized staged frontend captures, each scene records
`capture_mode: ui-fixture`, `proof_scope: ui`, `disclosure` placement in the report/share context, `capture_report`,
and `schema_validation` files. Every referenced claim has `scope: ui`. A regular
capture still needs all original provenance/privacy/pixel fields. The final-file
and listening gates apply to both capture modes. These structural checks cannot
prove visual fidelity or schema-report truth; reviewers inspect that evidence.

## Version 2 — current production contract

Use `version: 2` for new work. Version 1 remains readable for retained productions;
its old universal footage quota and every-shot motion gate are removed.

Add `review_status: draft|ready`. Story adds `viewing_context`, `category`, `setup`,
`next_action` and `footage_balance` (a rationale, not a quota). Each scene adds
`visual_intention`, `headline` where present, and `meaning` with `feature`,
`sample_input`, `observed_result`, `viewer_inference`. Use `none` explicitly for
an absent component. An exact query/headline collision fails preflight; the
reviewer must still catch less literal misleading wording.

Product sources add `proof_kind: state|interaction`, `frame`, `cursor`,
`settle_evidence` and `trim`. These document actual capture/edit choices.
Interaction proof requires observed `product_motion: true`; a faithful state
hold does not. Other provenance/privacy/pixel fields remain required.

`cta` contains `action`, full public HTTPS `url`, `expected_content`, `availability`
and `destination_report`. A destination report contains that exact `url`,
`checked_at`, `rendered_product_match: true`, a concrete `observation` and file
`evidence` beside the plan. Open the destination in a browser. These declarations
do not automate judgment about product identity or availability.

For QR endings set `cta.qr: true` and `qr_report`. Run:

```bash
python3 <skill-dir>/scripts/verify_qr.py video.mp4 --url https://vrooli.com/apps/verified-slug --at 20.5,22,23.5 --widths 1920,1280,960,640,390 --out qr-v2
```

Use existing approved OpenCV/FFmpeg installations. Missing tools fail explicitly.
The report decodes whole compressed frames without cropping/sharpening and binds
checks to the video hash. Physical scanning and destination uptime remain separate.

Final `review` adds `review_kind: self-review|independent-review|operator-review`,
`opening_comprehension`, `campaign_variety`, and separate `viewer_comprehension`
and `craft` objects. Each has `method`, `verdict` and `scenes`; each scene finding
has `id`, concrete `observation` and `verdict`. Cover every scene. These attestations
remain subject to critique; no validator authors passing taste or meaning findings.

`inspect` accepts `--cuts 4,8,12` to include a frame before, on and after each cut.
It checks encoded audio timing separately from container duration and confirms
that the file hash stays stable during verification. Music/voice readiness requires
full-film audio coverage and actual listening evidence. Share unfinished drafts
with open findings when useful; their readiness check remains unpassed.

## Freeze and join verification

List all composition dependencies as relative file `render_inputs`. Freeze them:

```bash
python3 <skill-dir>/scripts/workflow.py freeze production.json --out render-inputs-v2
```

Render from that snapshot through the managed control-plane host tool. Read
`internal/tools/hyperframes/README.md` and its adjacent versioned request schema.
Create a request declaring every snapshot file (including render-inputs.json),
their SHA-256 values, composition directory, explicit executable paths/hashes,
timeout and memory budget. Invoke `vrooli host hyperframes --request <request>
--out <new-bundle> --json`. When exercising an uninstalled source change, use
the repository's canonical `go run ./cmd/vrooli` entrypoint and record that fact;
do not claim an old installed CLI was updated. Provision only through the SDA
host-tool surface described by that owner. First qualify a tiny native render,
reuse and cancellation, then a representative multi-video trial using the intended
source dimensions. A text-only smoke does not exercise native extraction or its
task budget. Retain owner task observations, including unknown readings. A refused containment or
dependency check is a repair at its owner, not permission to use an old launcher.

The helper copies
only declared regular files (at most 512, 2 GiB), rejects escapes/symlinks, verifies
copies and writes hashes. It starts no services or renders. Treat snapshot inputs
as immutable; each revision uses a new directory. Record renderer version, CPU/
workers, memory/deadline, owned job/PID, cancel path and cleanup evidence in the
production record. Stop dependent work when a producer fails. Do not run an old
composition after a failed build.

The snapshot receipt binds the exact submitted plan hash as well as its declared
files. Predeclare final output/review paths before freezing. Populate those files
after rendering without changing the plan. If the plan changes, freeze a new
snapshot and assess whether the composition must be rendered again; verification
rejects a snapshot belonging to another plan.

After final export and written semantic/craft/listening review, join verification:

```bash
python3 <skill-dir>/scripts/workflow.py verify production.json --snapshot render-inputs-v2 --render-bundle render-bundle-v2 --out workflow-v2
```

For managed renders, pass --render-bundle. This joins the owner's completed
receipt to the exact complete snapshot and actual rendered-video hash. The raw
render and final mastered film may differ; retain the mastering recipe/input
hash and verify the final delivered film separately. Owner receipts bind local
bytes, not browser network isolation: review composition dependencies for
local-only assets and retain that evidence.

It validates pre-polish evidence, checks snapshot hashes, inspects the exact MP4
with cut-boundary samples and validates delivery review/CTA/QR receipts. The
original plan is unchanged; the new receipt retains its resolved delivery view.
No runtime binding currently owns these local-file operations. This helper is not
a program-runtime program, render daemon, durable job queue or publication gate.
Use governed capture, generation and lifecycle owners. Promote repeated remote
joins only after their real bindings exist.

For non-silent version 2 plans, `audio.source_id` identifies the actual selected
take/source. Add operator-specific `excluded_source_ids` when a production rejects
further sources. The checker also reads retained exclusions from `examples.json`.
It rejects those exact source IDs, not a whole genre. Preserve candidate identity,
provenance and the selection reason in `audio-selection.md`. The inspector records
LUFS and true peak; delivery validates `audio.target_lufs` (default -16) ±2 LUFS and
≤-1 dBTP. These signal checks cannot establish listening or taste.

Version 2 flagship directions each retain `rough` (a timed media file) and
`rough_observations` before polish. Compare story, medium, pacing and proof, not
only palette variants. Rejected roughs can remain weak; retain their findings
rather than polishing every alternative. The selected animatic/styleframes still
need their own reviews. Faithful PNG/JPEG/WebP state holds pass without a video
duration; a single still cannot establish an interaction.
