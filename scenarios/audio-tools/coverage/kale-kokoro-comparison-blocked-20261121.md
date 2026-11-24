# Kale Nano vs Kokoro same-host comparison attempt

- Date: 2026-11-21 (host local time, America/New_York)
- Scope: bounded attempt to compare via the installed Audio Tools product CLI; no provider/default/promotion changes.
- Host route: product API `127.0.0.1:19630` returned HTTP 200 from `/health`.
- Managed lifecycle snapshot before/after attempts: `kale-tts` was already running, healthy, and serving; `kokoro` was stopped. No resource lifecycle action was issued by this comparison.
- Product CLI provider inventory: Kale TTS reported `AVAILABLE`; Kokoro reported `UNAVAILABLE` with `resource is not responding`. The local browser TTS provider was available.
- Product requests attempted:
  - `audio-tools tts synthesize --text 'The quick brown fox jumps over the lazy dog.' --format wav --out /tmp/test.wav --json` returned `Error: synthesize: internal: tts unavailable`.
  - A second product request for voice `kenzie` produced no output file before its CLI process was terminated as hung.
- No audio was produced, so latency, duration/validity, RSS under synthesis, objective speech metrics, and a paired comparison are **not measured**. This artifact is a route failure receipt, not model quality evidence.
- Naturalness is unassessed. No model/provider qualification or default change follows from this attempt.

Reproduction / diagnostics:

```sh
vrooli resource status kale-tts vrooli resource status kokoro
audio-tools provider list --json
audio-tools health show --json
audio-tools tts synthesize --text 'The quick brown fox jumps over the lazy dog.' --format wav --out /tmp/test.wav --json
```

Observed API process: PID 3834899, executable `scenarios/audio-tools/api/audio-tools-api`; the CLI is connected to the existing product instance. Retry the comparison only after the product path can successfully synthesize both explicitly selected local backends through the supported route. Keep Kale experimental and Kokoro as catalog default until joined quality and lifecycle receipts exist.

## Outcome addendum — conditional Kale auto route (2026-11-21)

Implemented a declaration-backed `tts_runtime.conditional_auto` policy. The normal active-provider / eligible commercial default-model gate is unchanged. The policy only admits an explicitly flagged planned provider with a non-default fast/edge role and a commercially permitted candidate model. Kale opts in at priority 10; Kokoro remains the ordinary qualified route at priority 20. The resulting auto chain tries Kale first and traverses to Kokoro only on backend unavailability. Provider/model status, roles, and promotion receipts were not changed.

- Focused tests passed: `GOWORK=off go test ./internal/bootstrap ./internal/providercatalog` and race variants for `./internal/bootstrap ./internal/providercatalog ./internal/catalogconformance` from `scenarios/audio-tools/api`.
- Catalog conformance passed: `make -C scenarios/audio-tools catalog-check`.
- Isolated live product composition passed against the already-running local Kale service (no resource start/stop and no product API restart): `AUDIO_KALE_URL=http://127.0.0.1:18896 GOWORK=off go test ./internal/bootstrap -run '^TestBuildKaleTTSLiveProductRoute$' -count=1 -v` from `scenarios/audio-tools/api`. The test now selects `auto`, so it exercises declaration ordering rather than an explicit backend pin. It returned provider `kale-tts`, model `kale-tts-1-nano`, canonical `voice.feminine.warm` mapped to `kenzie`; first sample was a valid RIFF/WAVE, 97,664 bytes, 2,033 ms audio duration, 638 ms route wall time. Two repeat cache reads hit. Four additional canonical voice mappings produced valid WAVs in 42–62 ms warm route time. Existing resource health was HTTP 200 before the test.
- The original failure above remains an accurate receipt for the pre-change live product process; it was not restarted. This isolated route confirms new-source conditional routing with a real Kale runtime, but it does not prove the already-running product API now uses the new policy, nor does it compare Kokoro audio.
- This is serviceability/route and WAV-integrity evidence only. No human naturalness judgment, held-out quality score, Kokoro paired result, RSS-under-synthesis measure, lifecycle reliability qualification, or model promotion is claimed. Kale remains planned/candidate; Kokoro remains the qualified catalog default.
