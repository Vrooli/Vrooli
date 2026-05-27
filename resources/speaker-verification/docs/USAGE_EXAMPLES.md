# Speaker Verification Usage Examples

All examples assume the service is running on the default host port `11452`
(`SPEAKER_VERIFICATION_URL=http://localhost:11452`).

## Check readiness and model info

```bash
curl http://localhost:11452/ready
curl http://localhost:11452/v1/info
```

## Enroll a speaker profile

```bash
curl -X POST "http://localhost:11452/v1/profiles" \
  -F "profile_id=alice" \
  -F "display_name=Alice" \
  -F "notes=primary operator" \
  -F "audio=@alice_enrollment.wav"
```

Leave `profile_id` empty to let the server mint one:

```bash
curl -X POST "http://localhost:11452/v1/profiles" \
  -F "profile_id=" \
  -F "display_name=Guest" \
  -F "notes=" \
  -F "audio=@guest.webm"
```

## List enrolled profiles

```bash
curl http://localhost:11452/v1/profiles | jq .
```

## Verify a clip against a profile

```bash
curl -X POST "http://localhost:11452/v1/verify" \
  -F "profile_id=alice" \
  -F "threshold=0.25" \
  -F "audio=@unknown_clip.wav" | jq .
```

A `matched: true` result means the clip's cosine similarity to Alice's
enrollment embedding met or exceeded the threshold.

## Target-speaker extraction (reserved)

```bash
curl -i -X POST "http://localhost:11452/v1/extract" \
  -F "profile_id=alice" \
  -F "verify=false" \
  -F "audio=@mixture.wav"
# -> HTTP/1.1 501 Not Implemented
# {"error":"target speaker extraction not implemented"}
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
