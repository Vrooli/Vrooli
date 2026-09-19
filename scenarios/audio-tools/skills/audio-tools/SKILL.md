---
name: audio-tools
description: "Use Audio Tools for voice capture, transcription, synthesis, provider selection, and reproducible experiments; distinguish local, BYOK, and owned-service delivery and read evidence without mistaking availability for qualification."
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [tools]
  tags: [audio, voice, dictation, streaming, transcription, corpus, experiments]
  status: active
  revision: 2
  requires:
    scenarios: [audio-tools, program-runtime]
    commands: [audio-tools, program-runtime library run, prompt-manager skill read]
  origin:
    kind: authored
---
## Tools focus: Audio Tools

Select a voice operation with an explicit processing route. Return its observed
delivery and evidence limits, not a claim inferred from provider availability.

In scope: Audio Tools operations and experiment evidence. Product repair belongs
to `prompt-manager skill read audio-tools-improve`. Shared billing, model lifecycle,
and browser capture retain their own owners. Read
`path:scenarios/audio-tools/docs/concepts/INTEGRATIONS.md` before crossing them.

### Choose the operation

Read `audio-tools <group> <command> --help` for arguments to the selected leaf.
The CLI manifest, not this skill, owns option syntax.

| Intent / condition | Leaf and decision |
| --- | --- |
| Which engine can serve this host? | `audio-tools stt engines` [S1]. Availability is serviceability; preserve selection reasons and native-streaming capability. |
| Transcribe a supplied recording | `audio-tools stt transcribe` [S1]. Confirm route and permission to transmit the recording before a remote call. |
| Receive partial text | `audio-tools stt transcribe-stream` [S1]. A final-only batch adapter is not incremental recognition. CLI output does not prove browser rendering latency. |
| Produce speech | `audio-tools tts synthesize` or `synthesize-stream` [S1]. Apply the selected route's data and spend permission. |
| Summarize a transcript | `audio-tools summarize text` [S1]. Treat summarization as a separate operation, not part of STT billing. |
| Convert an audio file | `audio-tools audio transcode` [S1]. Confirm the output destination before overwriting an artifact. |
| Diagnose a failed operation | `audio-tools diagnostics last`, then an authorized `diagnostics run` [S1]. Separate cached evidence from a new probe. |
| Inspect a known experiment | `audio-tools experiment report <id>` [S1]. Keep its lane, recipe, warnings, and provenance; a succeeded job can have failing cells. |
| Read the improvement board | run `audio-tools.setpoint-read` via `program-runtime library run audio-tools.setpoint-read` [S3]. Supply `experiment_id` only for the intended evidence cohort. Branch on the contract envelope; `ok` means collection succeeded, never product acceptance. |
| Create a repeatable evaluation | Read `path:scenarios/audio-tools/docs/internal/TESTING.md` and choose corpus, cells, pacing, and spend before `audio-tools experiment start` [S0 → S1]. Use `experiment wait` once; cancellation of waiting does not cancel the experiment. |
| Embed voice in another app | Read `path:scenarios/audio-tools/docs/concepts/ARCHITECTURE.md#development-target-portable-streaming-voice` [S0]. Reuse the shared capture contract; do not copy session, recovery, or settlement logic. |

The board composes repeated engine, health, and experiment reads. Single audio
operations stay CLI leaves. Promote further composition only after recurring
usage identifies stable inputs and outputs.
The v2 board lists all 15 `portable-voice-v1` targets with unknown outcomes.
It cannot certify the full voice goal without owner-backed receipt joins. Read the
mandate/registry for required coverage; use `audio-tools-improve` for the missing
instrument or product work. Owned hosted delivery remains unqualified even when
an LPBS entitlement fixture succeeds.

### In-use settings

| Setting | Permitted move within the caller's grant | Verification |
| --- | --- | --- |
| Provider preference | Choose local, BYOK, or owned route explicitly; inspect `settings provider` help first | Read active selection and unavailable reason; do not silently enter a billable route |
| BYOK credentials | Use `settings byok-upsert` only for credentials supplied for this purpose | Inspect metadata with `byok-list`; never journal key values |
| Stream policy | Read `stt stream-config` before an authorized `stream-config-set` | Preserve speaker-policy and recovery constraints; retain the before/after configuration with the experiment |
| Engine lifecycle | Use the resource/control-plane owner exposed by provider commands | A restart or model pull needs scope and host impact assessment, not a private repair script |
| Experiment scope | Select named clips and cells; separate deterministic, realtime, and product-path lanes | Record realized recipe and corpus identity; never delete failed cells to improve a score |

### Evidence and learning

Read prior findings in `path:scenarios/audio-tools/docs/internal/PROBLEMS.md` before
non-trivial investigation. Use the existing engagement's evidence log and shared
Memory work-record protocol. This role does not declare a private learning scope
or duplicate Program Runtime's automatic capture. Return route, observed outcome,
artifact/run references, and untested claims. Omit audio, transcripts, keys, and
raw provider errors from general-purpose learning records.

### Troubleshooting & Edge Cases

| Observation | Next action |
| --- | --- |
| Engine list is slow on first access | Preserve cold/warm measurements; investigate discovery/probe timing through `scientific-debugging`, not a larger timeout alone |
| No partials | Inspect native-streaming capability and event cadence; test display commits separately from provider emission |
| Board is `partial` / `unavailable` | Read `errors[0].class` and affected rows. Check lifecycle and binding condition. Keep unaffected readings; do not replace missing readings with zero |
| Board is `failed` / `refused` | For `invalid_input`, correct the selector. For `kernel_runtime` or `binding_error`, inspect the recorded program. For a grant/spend refusal, retain the refusal and request the missing authority; do not escalate silently |
| Experiment is old, failed, canceled, or has no reference words | Keep it diagnostic or unavailable; collect a qualifying cohort before judging targets |
| Subscription fixture appears active but inference fails | Distinguish LPBS entitlement state from provider delivery; use the separate fake boundary described in TESTING.md |
| Local/BYOK fails | Return the configured failure/recovery option; an owned hosted fallback requires explicit policy and implemented delivery |
