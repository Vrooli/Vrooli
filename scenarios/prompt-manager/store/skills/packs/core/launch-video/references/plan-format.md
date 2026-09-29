# Production file contract (version 1)

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
  "version": 1,
  "kind": "launch",
  "product": "scenario-key",
  "profile": "teaser",
  "duration": 24,
  "format": {"width": 1920, "height": 1080, "fps": 30},
  "safety": {"status": "verified", "report": "safety-report.md", "protected_mutations": []},
  "story": {
    "promise": "One supported benefit", "viewer": "One audience",
    "problem": "The starting problem", "transformation": "The observable change",
    "proof": "The decisive recorded moment", "proof_scene": "result",
    "takeaway": "What remains in the viewer's mind"
  },
  "claims": [{
    "id": "c1", "source": "source command/path + revision",
    "read_at": "ISO-8601 timestamp", "approved": "Source meaning or locked text",
    "wording": "Planned paraphrase", "qualifiers": "none",
    "availability": "verified current scope", "exact_copy": false, "verified": true
  }],
  "audio": {"mode": "music", "source_kind": "governed-generation", "direction": "Sound and arc", "selection": "audio-selection.md"},
  "scenes": [{
    "id": "result", "start": 0, "duration": 24, "class": "evidence",
    "purpose": "Prove c1", "adds": "Visible consequence", "claims": ["c1"],
    "focal_point": "Result region", "motion_reason": "Make the result readable",
    "start_state": "Known state", "action": "Recorded user action", "end_state": "Visible result",
    "text": [{"value": "A readable result", "hold": 2}],
    "media": "footage/result.mp4", "source_sha256": "actual SHA-256",
    "capture_instance": "scenario@presentation", "capture_date": "ISO-8601 timestamp",
    "privacy_review": "footage/result-review.md", "product_motion": true,
    "crop": [0, 0, 1920, 1080], "max_display": [1600, 900]
  }],
  "video": "video-v2.mp4", "poster": "poster.jpg",
  "technical_report": "inspection/inspection.json", "review": "quality-review.json"
}
```

The example describes shape, not a recommended 24-second single shot. Scene slots
partition the full duration with no gaps/overlaps; transitions occur inside those
slots. `text.hold` is settled reading time, excluding entrances/exits. `crop` uses
actual encoded source pixels; `max_display` includes the largest zoom, not CSS size
before scaling. Evidence and presentation scenes require moving source media.

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

## Revision 13 preflight

Launch plans require captured media for at least half the duration at the plan
stage; studies require `study_requested: true` and `request_source` to proceed
to polish. A blocked launch request does not imply a study request.

Audio adds `source_kind` (`governed-generation`, `licensed-reuse`, `original-score`).
An original score needs `original_score_requested: true` and `request_source`.
When `variety_requested: true`, `candidate_styles` names distinct actual style
families, not seed IDs. Silence is a timing rough unless explicitly requested.

For operator-authorized staged frontend captures, each scene records
`capture_mode: ui-fixture`, `proof_scope: ui`, visible `disclosure`, `capture_report`,
and `schema_validation` files. Every referenced claim has `scope: ui`. A regular
capture still needs all original provenance/privacy/pixel fields. The final-file
and listening gates apply to both capture modes. These structural checks cannot
prove visual fidelity or schema-report truth; reviewers inspect that evidence.
