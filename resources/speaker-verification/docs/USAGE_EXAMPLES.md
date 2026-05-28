# Speaker Verification Usage Examples

All examples assume the service is running on the default host port `11452`
(`SPEAKER_VERIFICATION_URL=http://localhost:11452`).

## Check readiness and model info

```bash
curl http://localhost:11452/ready
curl http://localhost:11452/v1/info
```

## Enroll a speaker profile (append a labeled clip)

Each call appends ONE clip to the profile (creating it on the first call) and
recomputes the L2 centroid. Enroll several clips under different conditions to
strengthen the identity. The clip is embedded over its voiced span only; a clip
with too little voiced audio is rejected with HTTP `422`.

```bash
# first clip (normal speech on the laptop mic)
curl -X POST "http://localhost:11452/v1/profiles" \
  -F "profile_id=alice" \
  -F "display_name=Alice" \
  -F "label=laptop-normal" \
  -F "audio=@alice_laptop.wav"

# append a second condition (whisper on the phone) — same profile_id
curl -X POST "http://localhost:11452/v1/profiles" \
  -F "profile_id=alice" \
  -F "label=phone-whisper" \
  -F "audio=@alice_phone.wav"
```

Leave `profile_id` empty to let the server mint one (the response carries the id
to reuse for subsequent clips).

## List profiles / clips

```bash
curl http://localhost:11452/v1/profiles | jq .            # all profiles
curl http://localhost:11452/v1/profiles/alice | jq .       # one profile + clips
curl http://localhost:11452/v1/profiles/alice/clips | jq . # clips only
```

## Verify a clip against a profile

```bash
curl -X POST "http://localhost:11452/v1/verify" \
  -F "profile_id=alice" \
  -F "threshold=0.5" \
  -F "audio=@unknown_clip.wav" | jq .
```

A `matched: true` result means the score (hybrid: best of centroid + per-clip
cosine) met or exceeded the threshold. `sufficient: false` means the clip had
too little voiced audio to judge — `score` is 0 and `matched` is false; record a
longer clip rather than trusting the result.

## Delete a single clip

```bash
# deleting the last clip deletes the (now-empty) profile
curl -X DELETE "http://localhost:11452/v1/profiles/alice/clips/<clip_id>" | jq .
```

## Target-speaker extraction

Isolates the enrolled speaker's voice from a mixture (SepFormer separation +
ECAPA target-selection against the profile centroid), returning cleaned 16 kHz
mono s16le PCM in the body with `X-Speaker-Score` / `X-Speaker-Matched` headers.

```bash
curl -i -X POST "http://localhost:11452/v1/extract" \
  -F "profile_id=alice" \
  -F "verify=true" \
  -F "audio=@mixture.wav" --output cleaned.pcm
```

## Delete a profile

```bash
curl -X DELETE "http://localhost:11452/v1/profiles/alice"
```

## Via the resource CLI

```bash
# Lifecycle (shared control plane)
resource-speaker-verification status
resource-speaker-verification manage start
resource-speaker-verification logs

# Profile operations (lib/api.sh helpers, surfaced through content ops)
resource-speaker-verification content profiles
```

## From another scenario (audio-tools)

audio-tools resolves the service via the exported `SPEAKER_VERIFICATION_URL`
environment variable and calls the endpoints above through its Go HTTP client
(`scenarios/audio-tools/api/internal/stt/pipeline/speaker_client.go`). Never
hard-code the URL; read the exported env var.
