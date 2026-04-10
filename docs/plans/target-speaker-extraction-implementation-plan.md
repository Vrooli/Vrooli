# Target Speaker Extraction (TSE) Implementation Plan

## 1. Purpose

Add target speaker extraction to the speaker-verification resource and integrate it into the web-console voice pipeline so that only the enrolled user's voice is transcribed by Whisper — even when other voices (TV, other people, background audio) overlap with the user's speech.

Currently, speaker verification gates entire audio segments: if the user's voice is detected anywhere in a segment, the full segment (including all other voices) passes to Whisper. This causes other speakers' words to appear in the transcript. TSE solves this by extracting only the target speaker's waveform before transcription.

---

## 2. Required Reading

Before implementing, the executing agent must run:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement test
```

And read these files to understand current architecture:

```bash
# Speaker verification resource (Python)
cat resources/speaker-verification/service/verification.py
cat resources/speaker-verification/service/embeddings.py
cat resources/speaker-verification/service/audio.py
cat resources/speaker-verification/service/models.py
cat resources/speaker-verification/service/app.py
cat resources/speaker-verification/service/api/verify.py
cat resources/speaker-verification/service/config.py

# Web-console Go backend integration
cat scenarios/web-console/api/speaker_verification_client.go
cat scenarios/web-console/api/speaker_verification_handlers.go
cat scenarios/web-console/api/voice_stream_ws.go

# Existing tests
cat scenarios/web-console/api/voice_stream_ws_test.go
cat scenarios/web-console/api/speaker_verification_config_test.go

# Docker and dependencies
cat resources/speaker-verification/docker/Dockerfile
cat resources/speaker-verification/docker/requirements.txt
```

---

## 3. Problem Statement

**Symptom:** When the user speaks with background audio (TV, other people in the room, music), the transcript includes words from all audio sources, not just the user.

**Root cause:** The current pipeline performs speaker **verification** (binary yes/no: "is this the enrolled user?") on entire audio segments. If the user's voice is anywhere in the segment, the whole segment — including all overlapping audio — is forwarded to Whisper for transcription.

**Why diarization is insufficient:** Speaker diarization identifies *who speaks when* but assumes turn-taking. It cannot separate overlapping voices (e.g., user speaking over TV audio). The user's primary use case is speaking while TV or other ambient audio plays simultaneously.

**Solution:** Target Speaker Extraction (TSE) uses the enrolled speaker's embedding as a reference signal to extract *only that speaker's waveform* from the audio mix. The cleaned waveform is then sent to Whisper, producing a transcript of only the user's words.

---

## 4. Scope

### In scope

- Add a TSE model (SpeechBrain SepFormer or SpEx+) to the speaker-verification Python resource
- New `/v1/extract` API endpoint that accepts audio + profile ID and returns a cleaned WAV of just the target speaker
- Go client method in web-console to call the extraction endpoint
- Integration into the voice stream WebSocket pipeline (segment-final and session-final paths)
- Integration into the HTTP one-shot transcription path
- Configuration: enable/disable TSE, quality vs. speed mode
- Automated tests for the new endpoint and integration
- Performance benchmarking (latency impact measurement)

### Out of scope

- Real-time TSE on partial transcriptions (latency too high; partials remain as-is)
- Multi-speaker transcription (labeling who said what)
- Training custom TSE models — we use pretrained models only
- Changes to the frontend VAD or audio capture pipeline
- Changes to the enrollment flow (existing TitaNet embeddings are reused)

---

## 5. Current Technical Context

### Speaker Verification Resource (`resources/speaker-verification/`)

| Component | File | Role |
|-----------|------|------|
| Embeddings | `service/embeddings.py` | Loads NeMo TitaNet-Large, extracts 192-dim speaker embeddings |
| Verification | `service/verification.py` | Cosine similarity between profile and candidate embeddings |
| Audio processing | `service/audio.py` | Decodes any format via ffmpeg → 16kHz mono float32 numpy array |
| Profile storage | `service/profiles.py` | CRUD for profiles; embeddings stored as `.npy` files |
| API routes | `service/api/verify.py` | `POST /v1/verify` and `POST /v1/embeddings` |
| Config | `service/config.py` | Environment-based settings (device, model, thresholds, paths) |
| Docker | `docker/Dockerfile` | Python 3.11-slim + ffmpeg + torch + NeMo |

**Key detail:** The enrolled profile embedding (192-dim TitaNet vector) is already stored as `embedding.npy` in the profile directory. TSE models that accept a reference embedding can directly consume this.

### Web-Console Voice Pipeline (`scenarios/web-console/api/`)

| Component | File | Role |
|-----------|------|------|
| WS handler | `voice_stream_ws.go` | Real-time streaming; calls verification at segment + final |
| Verification client | `speaker_verification_client.go` | HTTP client wrapping resource API |
| Verification handlers | `speaker_verification_handlers.go` | `evaluateSpeakerVerification()` gate logic |
| Config | `speaker_verification_config.go` | Persisted JSON config (enabled, mode, threshold, etc.) |

**Integration points where TSE must be inserted:**

1. **Segment retranscription** (`voice_stream_ws.go`, segment-boundary handler): After extracting segment audio, before calling Whisper
2. **Final retranscription** (`voice_stream_ws.go`, done handler): After extracting final/tail audio, before calling Whisper
3. **HTTP one-shot** (`speaker_verification_handlers.go` / transcription handler): Before calling Whisper

### Audio Format Flow

```
Browser MediaRecorder (Opus/WebM 48kbps)
  → Go backend receives raw bytes
    → Speaker verification resource: ffmpeg decodes → 16kHz mono float32
    → Whisper: receives WebM or transcoded WAV
```

**Important:** TSE must output 16kHz mono WAV (the format Whisper expects when transcoding is enabled). The resource already normalizes input to 16kHz mono via `audio.py`.

---

## 6. Target End State

### Architecture After Implementation

```
Audio segment arrives (may contain user + TV + other voices)
  │
  ├─ Is TSE enabled AND speaker profile configured?
  │   │
  │   YES → POST /v1/extract (audio + profile_id)
  │   │     ├─ Load enrolled embedding from profile
  │   │     ├─ Run TSE model: isolate target speaker waveform
  │   │     ├─ Run speaker verification on extracted audio (confidence check)
  │   │     └─ Return: extracted WAV + verification score
  │   │
  │   │   If verification score >= threshold:
  │   │     → Send extracted WAV to Whisper → transcript of only user's voice
  │   │   If verification score < threshold (TSE failed to find user):
  │   │     → Reject segment (same as current behavior)
  │   │
  │   NO → Current behavior (verify whole segment, transcribe whole segment)
  │
  └─ Partials: unchanged (no TSE, too latency-sensitive)
```

### User-Visible Behavior Change

- When TSE is enabled and the user speaks with TV in the background, the transcript contains only the user's words
- When TSE is enabled but the user is NOT speaking (only TV audio), the segment is rejected (no transcript)
- Partials still show all audio (acceptable since they're ephemeral and overwritten by segment-finals)
- Slight increase in segment-final latency (~0.5–1.5s additional) — acceptable since segment-finals already take 0.5–2s

---

## 7. Implementation Strategy

### Phase 1: TSE Model Integration in Python Resource

**Goal:** Add the TSE model to the speaker-verification resource with a new `/v1/extract` endpoint.

**Steps:**

1. **Research and select TSE model.** Evaluate:
   - [SpeechBrain SepFormer](https://huggingface.co/speechbrain/sepformer-wsj02mix) — strong separation quality, larger model
   - [SpeechBrain SpEx+](https://huggingface.co/speechbrain/spex-plus) — designed specifically for target speaker extraction using a reference embedding; smaller and faster
   - Compatibility with 192-dim TitaNet embeddings as reference (may need an adapter/projection layer if the TSE model expects a different embedding format)

   **Assumption:** SpEx+ or a similar TSE model can accept a speaker embedding as the conditioning signal. If the model expects a reference audio clip instead of an embedding, we'll need to either (a) store a short reference clip during enrollment, or (b) use an embedding projection layer. **Verify this before coding.**

2. **Add TSE module** — new file `service/extraction.py`:
   - Load TSE model at startup (alongside TitaNet, with lazy loading option)
   - `extract_target_speaker(audio_waveform: np.ndarray, reference_embedding: np.ndarray) -> np.ndarray`
   - Returns cleaned waveform (16kHz mono float32)
   - Handle edge cases: if TSE output is near-silent (RMS < threshold), return None (indicates target speaker not present)

3. **Add API endpoint** — new file `service/api/extract.py`:
   - `POST /v1/extract`
   - Accepts: multipart form with `audio` file + `profile_id` string + optional `verify` bool (default true)
   - Loads profile embedding from store
   - Runs TSE to extract target speaker
   - Optionally runs verification on extracted audio as confidence check
   - Returns: extracted audio as WAV bytes in response body, with headers for score/matched/duration metadata
   - Content-Type: `audio/wav` (binary response, not JSON — the Go backend needs the raw audio to forward to Whisper)
   - Metadata in response headers: `X-Speaker-Score`, `X-Speaker-Matched`, `X-Duration-Ms`

4. **Add Pydantic models** for request validation (in `models.py`)

5. **Update dependencies** — add SpeechBrain to `docker/requirements.txt`:
   ```
   speechbrain>=1.0.0
   ```

6. **Update Dockerfile** if SpeechBrain needs additional system dependencies

7. **Configuration** — add to `config.py`:
   ```python
   TSE_ENABLED: bool = True
   TSE_MODEL: str = "speechbrain/spex-plus"  # or selected model
   TSE_DEVICE: str = "auto"  # share device with TitaNet
   TSE_MIN_OUTPUT_RMS: float = 1e-4  # below this, consider target speaker absent
   ```

**Seams to create:**
- `extraction.py` is a pure module with no HTTP concerns (testable independently)
- TSE model loading is behind a function that can be skipped if `TSE_ENABLED=false` (avoid loading ~500MB model when not needed)
- The `/v1/extract` endpoint calls `extraction.py` functions, keeping route logic thin

**Tests (Python):**
- Unit test `extraction.py` with synthetic audio (mix of two sine waves at different frequencies, extract one)
- Unit test edge case: near-silent TSE output returns None
- Integration test: `/v1/extract` endpoint returns valid WAV audio
- Integration test: `/v1/extract` with `verify=true` returns score headers
- Integration test: `/v1/extract` with unknown profile returns 404

### Phase 2: Go Client Extension

**Goal:** Add `Extract()` method to the speaker verification HTTP client in the web-console.

**Steps:**

1. **Add client method** in `speaker_verification_client.go`:
   ```go
   func (c *SpeakerVerificationResourceClient) Extract(
       ctx context.Context,
       audio []byte,
       profileID string,
       verify bool,
   ) (extractedAudio []byte, score float64, matched bool, err error)
   ```
   - POST multipart form to `/v1/extract`
   - Read response body as raw audio bytes
   - Parse `X-Speaker-Score` and `X-Speaker-Matched` headers
   - Return extracted audio + verification metadata

2. **Add response struct:**
   ```go
   type SpeakerExtractionResult struct {
       Audio    []byte
       Score    float64
       Matched  bool
       Duration float64  // processing time in ms
   }
   ```

**Tests (Go):**
- Unit test with mock HTTP server returning WAV bytes + headers
- Unit test error cases: 404 (profile not found), 400 (audio too short), 503 (model not loaded)

### Phase 3: Voice Pipeline Integration

**Goal:** Insert TSE into the segment-final and session-final transcription paths.

**Steps:**

1. **Update configuration** — add TSE fields to `SpeakerVerificationConfig`:
   ```go
   type SpeakerVerificationConfig struct {
       // ... existing fields ...
       ExtractionEnabled bool `json:"extractionEnabled"` // TSE on/off
   }
   ```

2. **Create extraction gate function** in `speaker_verification_handlers.go`:
   ```go
   func (s *Server) extractTargetSpeaker(ctx context.Context, audio []byte) ([]byte, *speakerVerificationGateDecision, error)
   ```
   - If TSE not enabled or not configured → return original audio unchanged
   - Call `s.speakerClient.Extract(ctx, audio, config.ProfileID, true)`
   - If extraction returns matched=true → return extracted audio + decision
   - If extraction returns matched=false → return nil audio + rejection decision
   - If extraction errors → fall back to original audio with verification-only (current behavior)

3. **Integrate into WebSocket handler** (`voice_stream_ws.go`):

   **Segment path** (in the segment-boundary goroutine):
   ```
   Before (current):
     segmentAudio := extractSegment(buf, start, end)
     decision := evaluateSpeakerVerification(ctx, segmentAudio)
     if decision.Allowed { transcribe(segmentAudio) }

   After:
     segmentAudio := extractSegment(buf, start, end)
     cleanedAudio, decision := extractTargetSpeaker(ctx, segmentAudio)
     if decision.Allowed { transcribe(cleanedAudio) }
   ```

   **Final/tail path** (in the done handler):
   ```
   Before (current):
     tailAudio := extractTail(buf, lastSegmentEnd)
     decision := evaluateSpeakerVerification(ctx, tailAudio)
     if decision.Allowed { transcribe(tailAudio) }

   After:
     tailAudio := extractTail(buf, lastSegmentEnd)
     cleanedAudio, decision := extractTargetSpeaker(ctx, tailAudio)
     if decision.Allowed { transcribe(cleanedAudio) }
   ```

4. **Integrate into HTTP one-shot path** (if one exists in transcription handler):
   - Same pattern: extract before transcribe

5. **Update WebSocket messages:**
   - Add `"extraction"` field to `segment-accepted` / `segment-rejected` messages:
     ```json
     {"type": "segment-accepted", "segmentIndex": 0, "score": 0.85, "threshold": 0.35, "extracted": true}
     ```
   - Add `speaker-status` message field: `"extractionEnabled": true`

6. **Timeout handling:**
   - TSE + verification combined timeout: 15s per segment (TSE ~1–3s, verification ~0.5s, with margin)
   - If TSE times out, fall back to verification-only on original audio

**Seams:**
- `extractTargetSpeaker()` is a single function that encapsulates the entire TSE decision — easy to test and substitute
- Falls back gracefully to current behavior if TSE unavailable
- TSE is independently toggleable from speaker verification

**Tests (Go):**
- Unit test `extractTargetSpeaker()` with mock client returning cleaned audio
- Unit test fallback: mock client errors → returns original audio + verification-only decision
- Unit test disabled: TSE config off → returns original audio unchanged
- Integration test: full WebSocket flow with TSE-enabled mock → segments contain only extracted content
- Integration test: `segment-accepted` message includes `extracted: true`

### Phase 4: Performance Validation and Tuning

**Goal:** Measure latency impact and tune for acceptable UX.

**Steps:**

1. **Benchmark TSE model inference time** on the target hardware:
   - Measure for 2s, 5s, 10s, 30s audio segments
   - Compare CPU vs CUDA performance
   - Document results in `resources/speaker-verification/docs/internal/PROGRESS.md`

2. **Measure end-to-end segment-final latency** with TSE enabled vs disabled:
   - Acceptable target: segment-final within 3s (current ~0.5–2s + TSE overhead)
   - If exceeding 3s on CPU, document that CUDA is required for TSE

3. **Tune TSE output quality:**
   - Test with real-world scenarios: user + TV, user + music, user + another person in conversation
   - Measure Whisper transcription accuracy on extracted audio vs original audio
   - Adjust `TSE_MIN_OUTPUT_RMS` threshold based on false-positive/negative rates

4. **Memory impact:**
   - Measure additional GPU/CPU memory from TSE model (~200–500MB expected)
   - Document in resource README

### Phase 5: Documentation and Cleanup

**Goal:** Update all documentation to reflect the new capability.

**Steps:**

1. **Update resource README** (`resources/speaker-verification/README.md`):
   - Document `/v1/extract` endpoint
   - Document TSE configuration options
   - Add usage examples

2. **Update web-console docs** if they exist (API reference for new config fields)

3. **Update SEAMS.md** (`scenarios/web-console/docs/internal/SEAMS.md`):
   - Document `extractTargetSpeaker()` as a seam
   - Document fallback chain: TSE → verification-only → fallback-allow

4. **Update existing speaker verification implementation plan** (`docs/resources/speaker-verification-implementation-plan.md`):
   - Add Phase 7 or appendix covering TSE

---

## 8. Contract Decisions

### New API Endpoint: `POST /v1/extract`

**Request:** Multipart form-data
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `audio` | file | yes | Audio in any format (WebM, WAV, MP3, etc.) |
| `profile_id` | string | yes | Enrolled speaker profile to extract |
| `verify` | bool | no (default: true) | Run verification on extracted audio |

**Response (success, 200):**
- Body: Raw WAV audio bytes (16kHz mono 16-bit PCM)
- Headers:
  - `Content-Type: audio/wav`
  - `X-Speaker-Score: 0.85` (if verify=true)
  - `X-Speaker-Matched: true` (if verify=true)
  - `X-Duration-Ms: 1234`
  - `X-Audio-Seconds: 3.2` (duration of extracted audio)

**Response (target speaker not detected, 200):**
- Body: Empty or near-silent WAV
- Headers:
  - `X-Speaker-Score: 0.12`
  - `X-Speaker-Matched: false`
  - `X-Duration-Ms: 1100`

**Error responses:**
- `404`: Profile not found
- `400`: Audio too short (< 1s), invalid format, or corrupt
- `503`: TSE model not loaded / not ready

**Rationale for binary response:** The Go backend needs the raw audio to send to Whisper. Returning JSON with base64-encoded audio would add serialization overhead and memory pressure for large segments.

### Configuration Extension

New fields on `SpeakerVerificationConfig`:
```json
{
  "extractionEnabled": true
}
```

`extractionEnabled` is independent of `enabled` (speaker verification). Both must be true for TSE to activate. This allows users to use verification-only without TSE if they prefer.

### Fallback Chain

```
TSE enabled + resource available → extract + verify extracted audio
TSE enabled + resource error    → fall back to verify-only on original audio
TSE disabled                    → verify-only on original audio (current behavior)
Verification disabled           → no gating at all (current behavior)
```

---

## 9. Testing Plan

### Python Resource Tests

| Test | Type | What it validates |
|------|------|-------------------|
| `test_extract_target_speaker_synthetic` | Unit | TSE model extracts dominant signal from synthetic mix |
| `test_extract_near_silent_output` | Unit | Returns None when target speaker not in audio |
| `test_extract_preserves_sample_rate` | Unit | Output is 16kHz mono |
| `test_extract_endpoint_returns_wav` | Integration | `/v1/extract` returns valid WAV with correct headers |
| `test_extract_endpoint_with_verify` | Integration | Headers include score and matched when verify=true |
| `test_extract_unknown_profile_404` | Integration | Returns 404 for nonexistent profile |
| `test_extract_short_audio_400` | Integration | Returns 400 for <1s audio |
| `test_extract_model_not_loaded_503` | Integration | Returns 503 if TSE model failed to load |

### Go Client Tests

| Test | Type | What it validates |
|------|------|-------------------|
| `TestExtract_Success` | Unit (mock HTTP) | Parses WAV body + score headers correctly |
| `TestExtract_ProfileNotFound` | Unit (mock HTTP) | Returns appropriate error on 404 |
| `TestExtract_Timeout` | Unit (mock HTTP) | Context cancellation propagates |

### Go Integration Tests

| Test | Type | What it validates |
|------|------|-------------------|
| `TestExtractTargetSpeaker_Enabled` | Unit | Calls client, returns cleaned audio + decision |
| `TestExtractTargetSpeaker_Disabled` | Unit | Returns original audio unchanged |
| `TestExtractTargetSpeaker_ClientError` | Unit | Falls back to verification-only |
| `TestVoiceStreamWS_SegmentWithTSE` | Integration | Full WS flow: segment → extract → transcribe extracted |
| `TestVoiceStreamWS_TSEFallback` | Integration | TSE error → segment still transcribed via verification-only |

### Manual Validation (Post-Implementation)

- Speak with TV playing in background → transcript should contain only user's words
- Speak alone (no background) → transcript unchanged from current behavior
- Only TV audio, no user speech → segment rejected, no transcript
- Speak over another person talking → only user's words transcribed

---

## 10. Rollout / Validation Checklist

- [ ] TSE model loads successfully on target hardware (check CUDA and CPU)
- [ ] `/v1/extract` returns valid WAV for test audio
- [ ] `/v1/extract` returns correct verification headers
- [ ] Go client `Extract()` parses response correctly
- [ ] `extractTargetSpeaker()` returns cleaned audio when TSE enabled
- [ ] `extractTargetSpeaker()` returns original audio when TSE disabled
- [ ] `extractTargetSpeaker()` falls back on client error
- [ ] WebSocket segment-final uses extracted audio when TSE enabled
- [ ] WebSocket session-final uses extracted audio when TSE enabled
- [ ] `segment-accepted` message includes `extracted` field
- [ ] `speaker-status` message includes `extractionEnabled` field
- [ ] All existing speaker verification tests still pass
- [ ] All existing voice stream tests still pass
- [ ] New Python tests pass
- [ ] New Go tests pass
- [ ] Segment-final latency with TSE is < 3s on target hardware
- [ ] TSE model memory usage documented
- [ ] Speaker verification resource Docker image builds successfully with new dependencies
- [ ] Resource health and readiness endpoints reflect TSE model status

---

## 11. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **TSE model expects reference audio, not embedding** | Medium | High — would require storing enrollment audio clips | Research model input requirements in Phase 1 Step 1 before any coding. If needed, modify enrollment to also store a short reference clip. |
| **TitaNet 192-dim embeddings incompatible with TSE model** | Medium | High — TSE models may use different embedding spaces | May need a learned projection layer, or use the TSE model's own speaker encoder. Evaluate during model selection. |
| **TSE adds too much latency on CPU** | Medium | Medium — segment-finals could exceed 3s | Benchmark early (Phase 4). If CPU is too slow, require CUDA or make TSE CUDA-only with CPU fallback to verification-only. |
| **TSE degrades audio quality, hurting Whisper accuracy** | Low | Medium — extracted audio may have artifacts | Test Whisper accuracy on extracted vs original audio. If degradation is significant, only use TSE when verification score on original audio is low (indicating mixed speakers). |
| **SpeechBrain + NeMo dependency conflicts** | Low | Medium — both use PyTorch but may want different versions | Pin compatible versions. Test in Docker build early. |
| **TSE model size increases Docker image significantly** | Low | Low — expected ~200-500MB additional | Document size impact. Consider lazy model download on first use. |

---

## 12. Non-Goals / Prohibited Patterns

- **Do NOT apply TSE to partial transcriptions.** Partials must remain low-latency (~300ms). TSE on partials would add 1–3s, destroying the real-time feel. Partials are ephemeral and overwritten by segment-finals anyway.
- **Do NOT modify the enrollment flow.** Existing TitaNet embeddings should be reusable. If the TSE model cannot use them directly, add an adapter/projection — do not re-enroll users.
- **Do NOT remove or weaken existing speaker verification.** TSE is additive. The verification gate still applies (on extracted audio). Users who disable TSE get current behavior exactly.
- **Do NOT add speaker diarization.** TSE replaces the need for diarization in this use case. Do not add both.
- **Do NOT create a separate resource for TSE.** It belongs in the speaker-verification resource since it shares the same model infrastructure, profiles, and embeddings.
- **Do NOT add frontend changes** beyond consuming new message fields. No new UI controls needed initially — TSE is controlled by the existing speaker verification config API.

---

## 13. Definition of Done

All of the following must be true:

1. **Functional:** `POST /v1/extract` endpoint exists, accepts audio + profile_id, returns extracted WAV of target speaker only
2. **Integrated:** Web-console segment-final and session-final paths use TSE when enabled, sending only extracted audio to Whisper
3. **Fallback-safe:** TSE errors fall back to verification-only on original audio; no regressions from TSE being unavailable
4. **Independently toggleable:** TSE can be enabled/disabled via `extractionEnabled` config without affecting base speaker verification
5. **Tested:** All tests in Section 9 pass; all pre-existing tests pass
6. **Performant:** Segment-final latency with TSE < 3s on target hardware (documented)
7. **Documented:** Resource README documents `/v1/extract`, config options, and performance characteristics
8. **Buildable:** Docker image builds successfully with SpeechBrain dependency; resource starts and passes health checks
