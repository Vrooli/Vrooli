# API and content export

This is a resource-specific HTTP contract, exposed through the existing resource
activity edge. It is not an ACE endpoint and does not populate Music jobs/pools.

`GET /health` returns200 only after exact local closure, CUDA/attention runtime and
model load pass. Loading or initialization failure returns503 with structured
`model_loaded`, error and initialization observations. `active_operation` identifies
generation/draining; it is not a promise of interruptible CUDA kernels.

`POST /v1/compose` accepts one JSON object, at most64KiB, with these fields:

| Field | Contract |
|---|---|
| prompt | Required nonblank text, maximum12,000 characters; actual tokenizer limit is checked before inference and truncation is refused. |
| seed | Required integer0..4294967295; explicit0 is preserved. |
| duration | Finite1..120 seconds; default48. The admitted pilot uses48. |
| steps | Integer1..32; default8. |
| cfg_scale | Finite0..20; default1. |
| timeout_seconds | Integer1..300; default300, enforced before/after operations and at supported sampler callbacks. |
| expected_model_revision | Optional exact `d4d3579395a8ac0df4ab13b5b1d7138ce7290f81`; other revisions are refused. |

Unknown fields, booleans in numeric fields, nonfinite values and invalid inputs
return400 before inference. Overlarge body returns413. Not-ready or busy/draining
returns503; there is no queue. A known generation deadline returns408 after the
worker reaches a cooperative check. Initialization/generation failures stay visible.

Success returns a version1 generation receipt, including opaque `operation_id`,
SHA256 of exact submitted request bytes, normalized effective request, verified
model-file identities, actual runtime/source hashes, requested/effective seed and
sampling settings, loaded sampling-shift class/parameters and actual WAV identity.
Attention bindings are verified; `actual_selected_paths=null` explicitly means
kernel-path use has not been instrumented. GPU allocator allocated/reserved peaks
are separate. Whole-scope and non-Torch peaks are unknown, not zero. Audition is
`unreviewed`.

`GET /v1/outputs/{operation_id}/audio` serves the completed source WAV. Only an
opaque32-character lowercase hex ID is accepted. Output and receipt are committed
together using Linux no-replace atomic publication. Existing output is preserved.

The maintained caller surface is:

```sh
resource-stable-audio-3 content compose --request request.json --out NEW_DIRECTORY
```

`--url` or `STABLE_AUDIO_3_URL` selects an HTTP loopback origin (default8896).
`--timeout` defaults330s and is bounded1s..10m. Parent output directory must exist.
The CLI checks request/model/seed/sampling/runtime receipts, length/hash and actual
stereo44.1kHz PCM24 RIFF frames before atomically publishing `audio.wav` and
`generation-receipt.json` to a new directory. Redirects and preexisting outputs
are refused. It preserves original source audio; Music delivery owns later edits.
It creates its owned export staging directory before the generation POST, so a
missing, non-directory or unwritable parent causes no inference request. The final
name remains protected by no-replace publication if it appears during generation.

On caller SIGINT/SIGTERM, deadline or disconnect, the HTTP request is cancelled.
The server requests cooperative cancellation and retains its generation lock until
the worker actually finishes/drains. In-flight GPU kernels or blocking native calls
are not force-killed. Cancellation before commit suppresses publication; a caller
can lose the response after an already completed commit. Do not infer that a CLI
error proves no completed server output. Inspect health/retained resource state;
normal resource stop remains the process-stop authority.
