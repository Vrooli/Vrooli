# Device Studio planned verification protocols

**Status: Requirement-level execution remains incomplete. Scoped runtime and manual-review evidence exists for named requirements below; see the [current delivery evidence map](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#p0-requirement-evidence-crosswalk). No whole-release acceptance is recorded here.**

These are reviewable procedures for the unimplemented product contract. Execute a case only when its behavior and supported owner adapters exist. Missing capability is NOT RUN or FAIL, never PASS. Do not invent test commands, provider results, artifact files or a manual-log entry. All fixture dimensions are explicit test inputs or source-backed references; they are not manufactured product facts.

Use isolated project/test storage through the owning Test Genie route before mutating tests. Mock credit/provider boundaries first. Any later real provider proof needs a configured cap, actual authorization and the project media-egress policy; writing this protocol authorizes no spend, account, publication or service change.

## Execution and evidence rules

1. Record requirement/protocol ID, product/source/tool revisions, fixture/source digests, selected platform/profile and project/shot/job IDs.
2. Run the positive flow and the stated negative case through actual supported UI/API/CLI or owner diagnostic interfaces. Retain exact commands or interaction recording, expected versus observed values and typed errors.
3. Retain actual output hashes, media probes/full-decode logs, frame/source timestamp mappings, worker receipts and resource/cost/reconciliation observations when relevant. Unknown cost/telemetry remains unknown.
4. For visual claims inspect final-size stills and required continuous playback; state exactly what was viewed. Keep physical measurements, numeric checks, visual judgment and operator/craft acceptance separate.
5. Record passed, failed, skipped or not_run truthfully. Bind any manual attestation to exact artifacts and scope with the existing owner manual-log contract only after execution.
6. A planned automated validation description is not an implemented test. Replace it with real specific test refs and [REQ:ID] tagging as implementation earns evidence.

Cross-cutting craft, release and platform criteria live in [ACCEPTANCE.md](../docs/internal/ACCEPTANCE.md). This file supplies the exact per-requirement cases.

## Immutable domain and shot contracts

### DS-CONTRACT-IDENTITY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-CONTRACT-IDENTITY
  Given Conflicting manufacturer and artist dimensions for two named variants
  When Create and reopen a definition
  Then Both claims and their provenance remain distinct; unknown facts remain unknown
```

**Counterexample / rejection oracle:** A listing title alone cannot establish a documented product or exact dimensions.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Unit/property tests of identity and measurement normalization.

### DS-CONTRACT-ASSEMBLY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-CONTRACT-ASSEMBLY
  Given Rigid, dual-screen, curved, deforming and optical fixture definitions
  When Compile valid and deliberately broken assemblies
  Then Valid supported assemblies resolve; bad references and known incompatibilities are refused; unknown fit needs review
```

**Counterexample / rejection oracle:** A family tag or wall_mountable flag cannot replace a compatible interface chain.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Contract and property tests for graph references, transforms and capability negotiation.

### DS-CONTRACT-SURFACES

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-CONTRACT-SURFACES
  Given Planar, circular, dual-rigid and continuous-fold display fixtures
  When Bind checkerboard media and compile each supported state
  Then Named surfaces map correctly; hinge gaps and physical occlusions remain excluded
```

**Counterexample / rejection oracle:** A four-corner planar mapping is refused for bending or nonplanar geometry.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Mask/UV tests and rendered diagnostic fixtures.

### DS-CONTRACT-SHOT

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-CONTRACT-SHOT
  Given A saved project using independently versioned libraries
  When Compile, change an upstream draft and reopen the original shot
  Then The original shot resolves identical pinned inputs and exposes its dependency closure
```

**Counterexample / rejection oracle:** Missing or mismatched dependencies produce an actionable refusal, not silently replaced assets.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Repository/serialization and immutable-revision integration tests.

## Qualified representative library

### DS-LIB-PHONE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-LIB-PHONE
  Given The candidate phone, references and final output profiles
  When Review physical diagnostics and continuous rotation
  Then The released recipe records identity, supported views, masks, reference deviations and motion acceptance
```

**Counterexample / rejection oracle:** A front beauty still cannot qualify unseen rear geometry or rotation.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Reference-comparison, mask and temporal acceptance suite.

### DS-LIB-LAPTOP

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-LIB-LAPTOP
  Given Open and closed reference dimensions and the candidate rig
  When Render opening and complete closure on a declared support
  Then Lid and display follow the same pivot; clearances and closed envelope pass the declared criteria
```

**Counterexample / rejection oracle:** A static joined open mesh cannot inherit lid-motion approval.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Rig/clearance property tests plus final-profile continuous-motion review.

### DS-LIB-TABLET

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-LIB-TABLET
  Given A candidate tablet and its supported stand or unsupported-mount case
  When Review portrait, landscape and the declared support poses
  Then The recipe preserves scale, screen orientation, cutouts and exact attachment scope
```

**Counterexample / rejection oracle:** A different generation or unsupported stand cannot pass by filename similarity.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Variant, mapping and attachment fixtures plus craft review.

### DS-LIB-FOLDABLE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-LIB-FOLDABLE
  Given A foldable candidate with separate outer/inner sources and a dual-rigid negative fixture
  When Evaluate each supported state and request an unsupported fold animation
  Then Supported state mappings pass; unsupported deformation is refused and native reflow is not fabricated
```

**Counterexample / rejection oracle:** Spanning content cannot paint a hinge or stretch one recording into a different captured app state.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Topology/state/handoff fixtures and visual state review.

### DS-LIB-WATCH

Status: PASSED for the declared front floating-body v4 protocol. Separate local shipped-starter delivery proof remains pending.

```gherkin
Scenario: DS-LIB-WATCH
  Given A round watch and rectangular diagnostic capture
  When Render the declared views and required text region
  Then Only active circular pixels show content and declared content remains readable
```

**Counterexample / rejection oracle:** Case diameter and square raster cannot be substituted for active circular geometry.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Actual predeclared circular-mask/readability oracle, source geometry/dimension/rights records, exact root review and PRIMARY package replay are retained in [the watch review](../docs/internal/goal/evidence/E5/qualification-review/live-watch-body-v4-lease30-1/README.md). No case-diameter substitution or full wearable claim.

**Planned automation:** Mask/ROI diagnostics and declared-pose review.

### DS-LIB-MONITOR-TV

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-LIB-MONITOR-TV
  Given Reviewed monitor/TV recipes and matching, mismatched and unknown bracket fixtures
  When Compile and render desk and wall placements
  Then Matching documented assemblies pass; mismatch is incompatible and unknown installation evidence requires review
```

**Counterexample / rejection oracle:** A hidden bracket cannot erase required mounting offset or cable clearance.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Attachment/clearance fixtures plus placement review.

### DS-LIB-SMART-DISPLAY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-LIB-SMART-DISPLAY
  Given Reviewed smart-display and digital-frame support variants
  When Place each in a supported scene and request an incompatible orientation or mount
  Then Supported variants preserve physical scale and declared display mapping; invalid placement is refused
```

**Counterexample / rejection oracle:** An ecosystem name such as Google Home cannot identify a screen-bearing hardware model.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Variant/support fixtures and final-look review.

### DS-LIB-VEHICLE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-LIB-VEHICLE
  Given A parked vehicle-host scene, app capture and multiple-panel negative case
  When Render the accepted host view and request a generic ecosystem-wide display size
  Then The recipe records host-specific surfaces, trim/controls occlusion and actual capture scope
```

**Counterexample / rejection oracle:** CarPlay or Android Auto names cannot establish hardware dimensions, cluster access or native availability.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Host/panel/app-window compatibility fixtures plus visual review.

## Photographic device composition

### DS-PHOTO-CALIBRATION

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-PHOTO-CALIBRATION
  Given A permitted fixed-view plate and a capture with different aspect
  When Calibrate, save and replay a bright diagnostic frame
  Then The mapping contains or explicitly crops the source without stretching and records approximation and unknown physical facts
```

**Counterexample / rejection oracle:** Generated hardware appearance cannot be promoted to measured geometry evidence.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Perspective/mask fixtures and manual calibration review.

### DS-PHOTO-ELIGIBILITY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-PHOTO-ELIGIBILITY
  Given A pinned static plate
  When Request screen playback and then phone spin, laptop closure or camera orbit
  Then Eligible playback works and unsupported motion returns a specific reason and qualified alternatives
```

**Counterexample / rejection oracle:** A crop zoom cannot silently stand in for a true physical move.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Strategy eligibility and negative API/UI tests.

### DS-PHOTO-REPLAY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-PHOTO-REPLAY
  Given A saved photographic shot with source start/hold/end intervals
  When Render twice and reopen after restart
  Then Source order, calibrated geometry and settings reproduce within the declared deterministic pipeline
```

**Counterexample / rejection oracle:** A missing plate cannot trigger undisclosed generation or a paid replacement.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Deterministic frame/receipt tests and reopen integration.

### DS-PHOTO-GENERATION

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-PHOTO-GENERATION
  Given An authorized plate-generation request with permitted blank/redacted screens and an unavailable-provider case
  When Generate through owner mocks and later a configured capped provider proof
  Then The actual returned plate and receipt are retained for review; failure leaves imported-plate composition usable
```

**Counterexample / rejection oracle:** A chat-tool image, route-available response or fabricated zero cost cannot certify the unattended provider path.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Owner adapter/receipt/cost-policy tests plus later explicitly configured capped generation proof.

## Controlled physical device motion

### DS-MOTION-PHONE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-MOTION-PHONE
  Given A qualified phone and its accepted rotation recipe
  When Render and continuously review the complete 1080p motion
  Then First, intermediate and last poses agree with tracks and required content remains usable during proof intervals
```

**Counterexample / rejection oracle:** A sequence of unrelated stills or flat-plane spin cannot qualify as physical device rotation.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Trajectory/occlusion checks and continuous playback review.

### DS-MOTION-LID

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-MOTION-LID
  Given A qualified laptop rig on a named desk support
  When Render complete opening and closure at the accepted profile
  Then The display follows the lid and no camera framing hides collisions or unsupported contact
```

**Counterexample / rejection oracle:** A moving screen alone, joined base rotation or unqualified closed thickness fails.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Rig endpoint/sweep tests and continuous playback review.

### DS-MOTION-ENTRY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-MOTION-ENTRY
  Given A qualified phone and laptop surface with known capture frame
  When Render the complete reveal-to-proof transition
  Then The terminal app frame aligns with the genuine capture without landing on bezel, OS padding or cutout
```

**Counterexample / rejection oracle:** A flat photographic crop is not labeled as 3D camera entry and unsupported curved targets are refused.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Projection/clock tests and final-size transition review.

### DS-MOTION-PARITY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-MOTION-PARITY
  Given A shot with easing, holds and joint motion
  When Compare selected sample transforms and scrub/reopen the shot
  Then Pose/time/crop semantics agree; approximate shading is visibly identified
```

**Counterexample / rejection oracle:** An adapter cannot reset the source clock or reinterpret easing without a new compiled revision.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Sampling parity and adapter contract tests.

## Guided and advanced shot editing

### DS-EDIT-GUIDED

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-EDIT-GUIDED
  Given A fresh personal-local workspace with supported local fixtures
  When Complete the primary shot workflow and reopen it
  Then Selections and outputs persist and invalid combinations explain valid alternatives
```

**Counterexample / rejection oracle:** No hidden model/agent launch or account requirement appears in deterministic steps.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Accessible end-to-end happy/empty/error/reopen journeys.

### DS-EDIT-ADVANCED

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-EDIT-ADVANCED
  Given A saved multi-track shot
  When Edit each control, save independent presets and reopen
  Then Changes create explicit draft revisions and evaluated output reflects the selected values
```

**Counterexample / rejection oracle:** Unsupported joint/mount state is refused rather than silently clamped or distorted.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Interaction/model round-trip tests and advanced editing journey.

### DS-EDIT-PREVIEW

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-EDIT-PREVIEW
  Given A worker with measured profile capabilities
  When Change framing and request a final-look checkpoint
  Then The user sees current preview state and which result authorizes expensive final rendering
```

**Counterexample / rejection oracle:** A low-quality proxy or stale preview cannot masquerade as approved final look.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Preview revision/state/latency checks and attended usability review.

### DS-EDIT-ACCESSIBILITY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-EDIT-ACCESSIBILITY
  Given Keyboard-only and narrow-viewport sessions
  When Complete core editing, error recovery, timeline and review tasks
  Then All required actions and state explanations remain reachable with intelligible focus and text
```

**Counterexample / rejection oracle:** A drag-only calibration or color-only error fails acceptance.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Accessibility audits and keyboard/mobile-review journeys.

## Authentic screen presentation

### DS-SCREEN-FIT

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-SCREEN-FIT
  Given Sources with mismatched aspect and a named required text region
  When Switch contain/crop and render landscape and portrait
  Then Only declared crop removes pixels; required content and source provenance remain inspectable
```

**Counterexample / rejection oracle:** Stretching content or silently dropping a required region fails.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Aspect/crop/ROI property and image diagnostics.

### DS-SCREEN-MASKS

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-SCREEN-MASKS
  Given Bright phone, watch, dual-panel and notch diagnostics
  When Render front, angled and intended scene views
  Then No app pixels paint physical occlusions; usable pixels around the phone island remain covered
```

**Counterexample / rejection oracle:** Dark content cannot conceal a missing active-area region or depth fighting.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Mask/UV/depth export diagnostics and visual review.

### DS-SCREEN-CHROME

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-SCREEN-CHROME
  Given Bare app and already-chromed recordings
  When Apply phone and desktop/browser profiles and vary indicators/theme/insets
  Then Only declared simulated layers change and the app viewport remains intact
```

**Counterexample / rejection oracle:** Live content cannot be stretched into status/home padding or misrepresented as real OS execution.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Presentation compositor fixtures and full-size review.

### DS-SCREEN-CLOCKS

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-SCREEN-CLOCKS
  Given VFR and constant-rate sources with known event timestamps
  When Render 24/30fps output with holds, trims and a display handoff
  Then Receipt maps each output sample to the intended source timestamp and does not infer VFR time from frame number alone
```

**Counterexample / rejection oracle:** Reopen/retry cannot restart screen playback or fabricate captured responsive reflow.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Timestamp sampling/property and decode-order tests.

### DS-SCREEN-AUDIO

Status: PASSED for the specified known-audible-marker protocol and first-profile silent-audio contract. Full scenario release acceptance remains separate. See [the P9 evidence review](../docs/internal/goal/evidence/E2V/p8-matrix/audio/acceptance-review.md).

```gherkin
Scenario: DS-SCREEN-AUDIO
  Given A recording with a verified audible marker and inspected audio stream facts
  When Import the recording, render the default shot, and request unsupported audio passthrough
  Then The default output contains no audio stream, original bytes and source audio facts remain intact, and the unsupported policy has a typed actionable repair
```

**Counterexample / rejection oracle:** Source audio cannot leak unexpectedly into a launch-film shot or trigger hidden AI audio.

**Evidence retained:** `p8-matrix/audio/public/phone-landscape30-fixed-audio-marker/run-2/` records the case, source, revisions, jobs and exports. `verified-2/result.json` verifies the 1920×1080, 30 fps, 60-frame output, full decode, output PTS and no-audio profile; the export SHA-256 is `ee93652bc6b4a5a772d0183c9c592d1a6ce52f92d8f8c980e27f2402e8dfc63f`. The original 238,018-byte source is SHA-256 `656e98844e46ff2898b56799cf45ee920bd19205f1989ea1d151622961c42c10` before and after rendering. Inspection records audio stream index 1 as `pcm_s16le`; independent fixture qualification confirms the audible marker. `public-proof/live-run-1/` retains the typed `UNSUPPORTED_AUDIO_POLICY` issue, failed-precondition submit and unchanged render/job/attempt counts. `audio/ui-live/run-2/` contains the desktop repair interaction, successful post-save GetShot, and desktop/mobile read-only captures for the detected-audio and repaired states.

**Implementation checks:** `TestVideoAudioPolicyIsExplicitAndBlocksUnsupportedRender` in `api/internal/shots/service_test.go`; `VideoTimingEditor.test.tsx` covers unreported, zero-stream and detected audio notices, the passthrough-only silent repair, and sample invalidation. Materials reports the focused eight-test VideoTimingEditor suite, UI type-check/string parity and changed-file ESLint passing.

**Validation caveats:** The blocked-repair BAS execution `a31f565a-580d-4722-adc1-931c290f83a7` saved the silent revision, then failed its final DOM assertion because `button:has-text(...)` was passed to native `document.querySelector`. The subsequent GetShot confirms revision 2 is `SILENT`, render-eligible, with no issues; the repaired mobile read-only execution passed. The scoped Test Genie unit run `20261020-133906-431b9dc7` remains FAIL: unit-health reports `COMPANION_REIMPLEMENTED` for the delegating `trackingMediaRPCStub` wrapper and `MISSING_INJECTABLE_SEAM` warnings. Keep the failed BAS and unit receipts; neither is a green end-to-end/unit verdict. This synthetic fixture and output do not establish real-recording or audio-passthrough support.

## Reference-based AI creation and refinement

### DS-CREATION-PIPELINE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-CREATION-PIPELINE
  Given A product brief with available and unavailable free asset leads
  When Run a bounded creation job through success and acquisition-gate cases
  Then Branch decisions, original bytes, revisions and candidate-ready or truthful partial outcomes are retained
```

**Counterexample / rejection oracle:** A vendor thumbnail or blocked download cannot be reported as acquired geometry.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Creation state-machine and source provenance integration tests.

### DS-CREATION-REFERENCE-LOOP

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-CREATION-REFERENCE-LOOP
  Given A deliberately inaccurate candidate and independent reference views
  When Refine proportions then materials and inspect comparison revisions
  Then Changed parameters and before/after mismatches are traceable and unavailable facts remain unknown
```

**Counterexample / rejection oracle:** The job cannot improve a score by changing ground truth, deleting difficult landmarks or substituting generated imagery for geometry.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Reference instrument tests and real held-out-view refinement review.

### DS-CREATION-AUTONOMY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-CREATION-AUTONOMY
  Given An authorized autonomous job with nested work and pinned constraints
  When Exercise normal iterations and a scope expansion request
  Then Approved work proceeds without repeated permission modals; expansion waits and artifacts remain resumable
```

**Counterexample / rejection oracle:** A child cannot reset the parent budget, inherit blanket human authority or outlive cancellation.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Orchestrator/grant/budget fault-injection and activity review.

### DS-CREATION-ISOLATION

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-CREATION-ISOLATION
  Given Malicious archive/script fixtures and a legitimate bounded modeling script
  When Import and execute through the supported contained owner path
  Then Only allowed files/egress/processes are reachable and cancellation removes owned descendants
```

**Counterexample / rejection oracle:** CPU/memory process limits alone cannot qualify filesystem/network isolation.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Sandbox/argv/archive adversarial tests and containment qualification.

### DS-CREATION-REVIEW

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-CREATION-REVIEW
  Given Candidate-ready, budget-exhausted and failed jobs
  When Inspect/select a candidate and attempt shared release
  Then Selection and physical/craft verdicts bind exact revisions; unreviewed candidates remain unpublished
```

**Counterexample / rejection oracle:** An agent cannot self-promote a candidate because a render job succeeded.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Candidate/review/publication boundary integration tests.

## Rights and scoped visual qualification

### DS-REVIEW-RIGHTS

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-REVIEW-RIGHTS
  Given CC0, CC BY, render-only, unknown and editorial-only fixture packages
  When Request commercial media delivery and editable shared bundling
  Then Only rights-compatible operations proceed and generated/approximate assets remain labeled
```

**Counterexample / rejection oracle:** Free access or a model license cannot silently clear thumbnail, brand, third-party image or provider-upload rights.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Rights-policy decision fixtures and package attribution audit.

### DS-REVIEW-PHYSICAL

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-REVIEW-PHYSICAL
  Given A candidate with a good front view and faulty rear/hinge detail
  When Review full promised angle/motion scope
  Then The verdict records deviations, uncertainty and explicit excluded views/states
```

**Counterexample / rejection oracle:** One width anchor, beautiful still or schema fixture cannot certify physical accuracy.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Reference/measurement and motion-sweep checks with attended verdict.

### DS-REVIEW-CRAFT

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-REVIEW-CRAFT
  Given A technically valid but visibly weak scene and an improved candidate
  When Review final-size stills and continuous playback
  Then Craft acceptance is explicit and independent from technical completion and professional claims remain scoped
```

**Counterexample / rejection oracle:** More samples, file admission or an accepted photographic plate cannot certify unrelated 3D quality.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Attended art-direction protocol with accepted/rejected comparison evidence.

### DS-REVIEW-PORTABLE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-REVIEW-PORTABLE
  Given A native scene with procedural shaders and a portable GLB adaptation
  When Export, reimport and inspect intended scene bounds/views
  Then Losses and target-specific adaptations are declared and static GLB is not claimed to contain video playback
```

**Counterexample / rejection oracle:** A clean standalone import cannot qualify depth, texture budget or lighting in every larger scene.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Export/reimport/target-admission tests and visual inspection.

### DS-REVIEW-PUBLICATION

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-REVIEW-PUBLICATION
  Given A reviewed local candidate and an unavailable publication owner
  When Publish through the owner or retain a local accepted revision
  Then Owner references and availability are explicit and local work remains usable
```

**Counterexample / rejection oracle:** Local acceptance cannot silently create an ungoverned shared release.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Partial candidate, source, or scoped visual evidence exists and is mapped at [the remaining-delivery crosswalk](../docs/internal/goal/evidence/E3/production-continuation/final/remaining-delivery-map.md#library-and-review-evidence-crosswalk); this complete protocol, including its counterexample, remains NOT EXECUTED.

**Planned automation:** Owner-adapter and unavailable-state integration tests.

## Durable jobs and reproducible recovery

### DS-JOB-DURABILITY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-JOB-DURABILITY
  Given A multi-stage render/creation job
  When Restart the API during queued and running stages
  Then The job resumes a valid checkpoint or reports a truthful interrupted state with retained evidence
```

**Counterexample / rejection oracle:** An in-memory queue or missing worker receipt cannot become completed output.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Crash/restart persistence and state-machine tests.

### DS-JOB-CANCEL

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-JOB-CANCEL
  Given Concurrent jobs and a nested local/remote child
  When Cancel one parent and inspect all workers/receipts
  Then Only that ownership tree stops and remote consumed-cost uncertainty remains visible
```

**Counterexample / rejection oracle:** Cancellation cannot stop unrelated services/GPU tenants or imply an unsupported remote refund.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Ownership isolation, process cleanup and cancellation fault injection.

### DS-JOB-RECOVERY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-JOB-RECOVERY
  Given Jobs with injected disk exhaustion, corrupt frame and worker exit
  When Fail, free space or repair a dependency, then resume
  Then Completed valid checkpoints survive and retries do not reset source time or overwrite accepted artifacts
```

**Counterexample / rejection oracle:** Lost acknowledgment cannot blindly resubmit a non-idempotent paid operation.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Disk/tool/output failure injection and checkpoint replay tests.

### DS-JOB-CACHE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-JOB-CACHE
  Given A cached fixed shot and one-at-a-time dependency changes
  When Replay unchanged inputs then mutate each relevant dependency
  Then Only valid reusable units hit cache and the receipt names the reused artifact
```

**Counterexample / rejection oracle:** A static plate cannot inherit changing reflection, occlusion, geometry or camera guarantees.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No product-level render replay or stationary replacement comparison has been executed. Focused owner tests are implementation evidence only.

**Automation:** The owner now has opt-in exact-fingerprint reuse and cache invalidation tests. Native stationary replacement comparison remains unexecuted.

### DS-JOB-ACCOUNTING

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-JOB-ACCOUNTING
  Given A funded child whose completion acknowledgment is lost
  When Retry status/settlement and resume the parent
  Then The same execution identity is reused; additional real work is a separately admitted attempt
```

**Counterexample / rejection oracle:** Unknown usage cannot become zero or trigger automatic resubmission.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Accounting replay/reconciliation fault injection against owner mocks and later capped integration.

## Disclosure, privacy and total budgets

### DS-SAFETY-DISCLOSURE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-SAFETY-DISCLOSURE
  Given Local rendering, single AI generation and autonomous agent actions
  When Inspect preflight and run each action
  Then Users can distinguish AI, agent and deterministic work before initiating and activity records every nested action
```

**Counterexample / rejection oracle:** A sparkle icon, hidden setup repair agent or new undisclosed child path is insufficient.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** UI/activity and server dispatch-policy acceptance tests.

### DS-SAFETY-BUDGET

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-SAFETY-BUDGET
  Given Concurrent/nested jobs with known quote, unknown cost and insufficient remaining budget
  When Attempt dispatch and scope expansion
  Then Only safely bounded admitted work runs; estimate, actual and pending cost remain distinct
```

**Counterexample / rejection oracle:** A displayed estimate, per-step reset or reservation above an unenforced provider maximum cannot establish a hard ceiling.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Concurrent budget/property tests and unknown-cost refusal protocol.

### DS-SAFETY-EGRESS

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-SAFETY-EGRESS
  Given A project with real capture media and sharing disabled
  When Request generation/critique through direct and nested routes
  Then Only permitted redacted/reference bytes leave; destination, purpose, consent revision and digests are retained
```

**Counterexample / rejection oracle:** Sign-in, BYOK key or spend consent alone cannot authorize original capture upload.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Egress interception/redaction/consent-revision tests.

### DS-SAFETY-IMPORT

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-SAFETY-IMPORT
  Given Public URL plus redirect/private-address, traversal and oversized archive fixtures
  When Import through public API and UI
  Then Authorized bounded data is preserved with provenance and rejected inputs leave no escaped files or network access
```

**Counterexample / rejection oracle:** A public listing cannot authorize authentication bypass, viewer ripping or arbitrary internal URL fetch.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** SSRF/redirect/archive/path and bounded-import adversarial tests.

### DS-SAFETY-INPUT-BOUNDS

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-SAFETY-INPUT-BOUNDS
  Given Malformed/nonfinite payloads, huge allocations and unauthorized object references
  When Submit through API/CLI and import flows
  Then Invalid work is refused before allocation/dispatch; logs and exported packages exclude credentials
```

**Counterexample / rejection oracle:** A successful JSON parse cannot authorize unbounded memory or unrelated project access.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Payload fuzz/property, authorization and resource-bound tests.

## Local use and normal Vrooli commerce

### DS-COMMERCE-PERSONAL

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-COMMERCE-PERSONAL
  Given A fresh private desktop with account/provider services absent
  When Complete a local saved-shot flow offline
  Then Local work and project data remain usable; remote services are not silently substituted
```

**Counterexample / rejection oracle:** Expired lease, zero credits or sign-out cannot erase projects or block declared local capability.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Offline/signed-out and dependency-unavailable journeys.

### DS-COMMERCE-LINK

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-COMMERCE-LINK
  Given Two commercial accounts and valid, wrong-binding, expired and revoked leases
  When Link, switch, refresh offline and disconnect
  Then Only the selected bound account is shown/usable and local cleanup succeeds independently of remote status
```

**Counterexample / rejection oracle:** A lease snapshot, email match or copied website token cannot authorize funded execution.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Identity-binding/PKCE/replay/revocation and packaged account journeys.

### DS-COMMERCE-FUNDING

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-COMMERCE-FUNDING
  Given Local, BYOK and Vrooli-funded versions of an operation
  When Run owner-mocked reserve/settle, cancellation and insufficient-credit cases
  Then Funding source is explicit and only trusted actual funded work consumes platform credits
```

**Counterexample / rejection oracle:** Text metering or desktop lease issuance cannot alone certify the separate media-funded route.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Funding/adaptor contract tests and later configured capped owner journey.

### DS-COMMERCE-DELIVERY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-COMMERCE-DELIVERY
  Given Entitled/unsigned or unauthorized download requests and an installed offline app
  When Request a precise platform asset then open a local project without sign-in
  Then Server acquisition gate and local runtime behavior are independently correct
```

**Counterexample / rejection oracle:** Listing entitlement cannot become a new local export charge or private licensing system.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Delivery authorization and fresh-install/offline journey tests.

### DS-COMMERCE-PRICING

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-COMMERCE-PRICING
  Given Known and unavailable owner price/usage projections
  When Open account/cost surfaces and reconcile job receipts
  Then Source/freshness/unit are visible and unavailable is not zero; Money Ledger remains financial actuals owner
```

**Counterexample / rejection oracle:** A local hardcoded rank/price cannot become commercial authority.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Account UI projection and stale/unknown owner-response tests.

## Cross-platform desktop architecture

**Approved delivery scope (2026-10-19):** Execute runtime/package cases on the current host only. Retain cross-platform architecture/build-contract checks; Windows/macOS native execution, signing and certification are outside this effort and remain explicitly unverified. Per-platform cases below apply before any future native support claim. See `docs/internal/ACCEPTANCE.md`.


### DS-DESKTOP-ARCHITECTURE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-DESKTOP-ARCHITECTURE
  Given Platform adapter fixtures and paths with spaces/Unicode
  When Build/test adapter interfaces and inspect unavailable capabilities
  Then No Linux shell/path assumption leaks into common contracts and unsupported targets are explicit
```

**Counterexample / rejection oracle:** Cross-compilation alone cannot earn target-native certification.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Cross-platform build/contract tests and native platform matrix review.

### DS-DESKTOP-PACKAGE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-DESKTOP-PACKAGE
  Given A clean user installation without Vrooli checkout or fleet services
  When Install, launch, save, import, locally compose/render/export and uninstall
  Then The advertised local flow succeeds and user data/immutable binaries use proper roots
```

**Counterexample / rejection oracle:** Ambient Python, system fonts, Bash or an undeclared host binary cannot silently satisfy package requirements.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Fresh-user packaged smoke/offline/path and artifact-closure checks.

### DS-DESKTOP-TOOLS

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-DESKTOP-TOOLS
  Given Available CPU/GPU hardware with a missing kernel/backend fixture
  When Probe, request a specific profile and offer eligible fallback
  Then Only qualified profiles execute; CPU fallback is explicit and preserves requested supported semantics
```

**Counterexample / rejection oracle:** Hardware detection cannot imply a usable GPU, and no silent cloud/install fallback occurs.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Tool capability/packaging tests and actual worker diagnostic render.

### DS-DESKTOP-LIFECYCLE

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-DESKTOP-LIFECYCLE
  Given Target-native signed installer and prior-version project fixture
  When Install, disconnect network, restart, update and close with a running worker
  Then Local data survives and owned descendants stop without touching other services
```

**Counterexample / rejection oracle:** An OS badge, build success or mocked updater cannot replace target-native evidence.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Per-platform install/update/offline/security lifecycle protocol.

## Public operations and launch-video integration

### DS-INTEGRATION-OPERATIONS

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-INTEGRATION-OPERATIONS
  Given A saved UI project and a fresh API/CLI caller
  When Create, inspect, reopen, compile, render, cancel and export through public surfaces
  Then Results and constraints match and unavailable operations explain the owner reason
```

**Counterexample / rejection oracle:** The launch skill cannot require editing native files or undocumented database writes for ordinary shots.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** API/CLI contract, auth and parity journeys.

### DS-INTEGRATION-LAUNCH

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-INTEGRATION-LAUNCH
  Given A launch-video task with authentic capture and a saved device shot
  When Assemble a film segment through the supported owner integration
  Then Shot media, timing, funding and provenance receipts are consumed without claiming fixture UI as backend/native proof
```

**Counterexample / rejection oracle:** The skill cannot silently replace app pixels with generated imagery or treat Device Studio as full-film owner.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Invoke `device-studio.launch-video-shot` through Program
Runtime with an existing project, saved shot ID, expected head revision, typed
snapshot and stable render idempotency key. Retain the program execution ID,
exact saved revision and reopen response, render job ID/status and the export
metadata. When the job succeeds, run the returned public
`device-studio renders export` command to a new path and retain its verified
download result plus artifact SHA-256. A queued or running job is observed again
by its exact project/job IDs; cancel only through the owner's expected-version
operation. Compose the exported segment in Launch Video/HyperFrames and review
the actual complete film, which keeps audio and CTA outside Device Studio.
Use the approved photographic phone plate and an authentic app recording for
the source-continuous screen-entry recipe; verify the final decoded frame shows
the active app viewport at the source-clock position declared by the saved
revision. The approved still does not approve future scene craft, and the
synthetic P7 camera calibration output is not photographic craft evidence.

### DS-INTEGRATION-DELIVERY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-INTEGRATION-DELIVERY
  Given A completed shot and portable package
  When Open direct outputs in the intended consumer environment
  Then Artifacts resolve with declared format/fidelity and links do not depend on an internal HTML gallery
```

**Counterexample / rejection oracle:** Local browser HTML success cannot certify a recipient artifact viewer or video-textured static GLB.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Direct artifact/download/reimport checks in intended consumer environment.

## Qualified output and renderer profiles

### DS-OUTPUT-PROFILES

Status: PASSED for the four declared1080p SDR video profiles and supported native2560x1440 still. See [the executed matrix](../docs/internal/goal/evidence/E2V/output-profiles/lease28/README.md) and root manual attestation recorded2026-10-21; full release remains separate.

```gherkin
Scenario: DS-OUTPUT-PROFILES
  Given Source/scene fixtures for all four video profiles and a supported still profile
  When Export, fully decode/probe and inspect test patterns
  Then Raster/SAR/frame timestamps/duration/color match the selected profile and limitations are recorded
```

**Counterexample / rejection oracle:** A larger raster alone cannot certify native 4K detail or HDR support.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. Actual public jobs, exact hashes, all-frame decode/PTS checks, declared color/range and root visual samples are retained in the linked matrix. Native smartv7 job8872318d-5f44-4663-b430-17db2953a619 supplies the supported2560x1440 still. The MP4s omit an explicit SAR field; the recorded square-pixel default interpretation is not an emitted-SAR claim. No 4K detail or HDR support is inferred.

**Planned automation:** Probe/full-decode/color-chart and selected-profile tests.

### DS-OUTPUT-READABILITY

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-OUTPUT-READABILITY
  Given Low-resolution and adequate source variants with a named proof region and readability criteria declared before evaluation
  When Render landscape/portrait and review the proof segment at final size
  Then The inadequate variant cannot receive accepted or final-qualified status
  And An explicit source or composition revision receives acceptance only after its proof segment passes the declared final-size readability criteria
```

**Counterexample / rejection oracle:** A warning, offered alternative, successful encoder or inferred text-size number cannot replace final-size readability review or permit acceptance of inadequate required content. A technically valid preview may remain available while acceptance is blocked.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** ROI/source adequacy fixtures and attended proof-interval review.

**Implementation target (2026-10-21):** Keep the proof declaration in the immutable
shot snapshot and keep each attended verdict in a separate append-only record
bound to shot revision, render job, output digest, and declared region. A successful
render alone never creates acceptance. Existing preview/download behavior remains
available for undeclared or rejected proofs. Final-size evidence must include
inadequate and adequate landscape and portrait variants; source changes create a
new shot revision and cannot reuse an earlier verdict.

### DS-OUTPUT-COLOR

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-OUTPUT-COLOR
  Given Color charts, bright/dark app captures and beauty scene variants
  When Replay native, portable and encoded outputs under their declared profiles
  Then Transforms are explicit and source identity survives the stated mapping/encode limits
```

**Counterexample / rejection oracle:** Unknown color metadata, double gamma or unlit extension conflicts cannot silently pass.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Color/mask/export diagnostics and reference parity review.

### DS-OUTPUT-METRICS

Status: NOT EXECUTED. Committed first-release requirement.

```gherkin
Scenario: DS-OUTPUT-METRICS
  Given A qualified worker and selected 1080p profiles
  When Run bounded preview/final samples and cancel a sample
  Then Receipts distinguish measured values, unavailable telemetry and approximate preview from interactive editing claims
```

**Counterexample / rejection oracle:** Renderer internal allocation counts or prior constrained CPU timings cannot be normalized into product FPS promises.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Owned resource/timing capture and qualification evidence review.

## Variant comparison and bounded batches

### DS-BATCH-COMPARE

Status: NOT EXECUTED. Explicit later roadmap; not first-release completion credit.

```gherkin
Scenario: DS-BATCH-COMPARE
  Given Two independently saved shot variants and one deliberately invalid variant
  When Compare and submit a bounded batch
  Then Each result retains its own revision, review and cost state and one failure does not obscure the others
```

**Counterexample / rejection oracle:** A batch cannot merge budgets or inherit another variant's visual acceptance.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Future batch/variant and receipt-isolation tests.

## Advanced video output profiles

### DS-OUTPUT-ADVANCED

Status: NOT EXECUTED. Explicit later roadmap; not first-release completion credit.

```gherkin
Scenario: DS-OUTPUT-ADVANCED
  Given A future advanced profile and inadequate source negative fixture
  When Request and validate the added profile
  Then The advertised profile has its own actual decode, color and resource acceptance evidence
```

**Counterexample / rejection oracle:** Upscaling or SDR metadata cannot certify native 4K or HDR.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Future advanced-profile media/color acceptance suite.

## Optical and virtual presentation targets

### DS-OPTICAL-TARGETS

Status: NOT EXECUTED. Explicit later roadmap; not first-release completion credit.

```gherkin
Scenario: DS-OPTICAL-TARGETS
  Given A future optical adapter with receiver/viewer calibration and an unsupported target
  When Compile the qualified optical recipe
  Then Optical/viewer geometry, apparent image and limitations remain explicit
```

**Counterexample / rejection oracle:** Projector or headset body dimensions cannot determine an apparent screen.

**Evidence to retain after execution:** The case inputs/revisions, actual actions and observed result versus the Then/counterexample oracle, plus applicable records from the rules above. No evidence has been collected for this product requirement.

**Planned automation:** Future optical mapping and viewer-pose qualification fixtures.
