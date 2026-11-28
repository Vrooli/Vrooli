const e=`# Sopro voice-cloning implementation contract

Status: design contract only. Sopro remains planned and its model remains a
non-runnable candidate. This document does not authorize downloading weights,
starting a runtime, or exposing a product operation.

## Current evidence

The Audio Tools model seed pins \`samuel-vitorino/sopro-v2-turbo\` at revision
\`c12d0dc7c4a33b3cb6ea3d11ceb70bc60c7c6312\`. That snapshot is recorded as
120M parameters, Apache-2.0, four languages, and approximately 952 MB across
its full repository contents. The pinned model card says current artifacts
require \`sopro>=2.2.0\` (or the browser package \`@soprotts/onnx-web>=0.3.0\`),
and describes CPU inference and 5–20 second reference audio. These upstream
claims do not measure Audio Tools peak memory, latency, or output quality.
The Python package's transitive runtime licenses and full reproducible
environment have not been audited. The separate browser implementation is
not an Audio Tools runtime and is not implicitly approved by this contract.

Upstream references:

- [Pinned Sopro V2 Turbo model card](https://huggingface.co/samuel-vitorino/sopro-v2-turbo/tree/c12d0dc7c4a33b3cb6ea3d11ceb70bc60c7c6312)
- [Sopro runtime source](https://github.com/samuel-vitorino/sopro)

## Required operation boundary

Cloning must be a separately named, explicit operation such as
\`tts/voice-clone\`. Ordinary TTS requests must never infer cloning from a
provider, voice id, model id, or supplied audio field. The route must reject
unknown, planned, or unqualified provider/model pairs and must not appear in
provider actions or model resolution until the owning gates have passed.

The caller must submit an explicit clone intent and a fresh, human-authorized
consent receipt. Consent authority must validate the authenticated owner,
provider and operation purpose, and the exact reference source (a stable
source identifier plus a cryptographic digest of the received audio). The
receipt must be revocable and checked immediately before inference and again
before generated audio is returned. Missing authority, invalid or expired
receipts, revocation, or authority errors fail closed. Audio Tools must not
turn a receipt into a reusable local grant.

## Reference-audio lifecycle and limits

- Accept only bounded decoded audio in the model's declared 5–20 second
  window. Cap encoded bytes, decoded samples, text length, synthesis duration,
  and wall-clock time independently. Reject before model work when any limit
  is exceeded.
- Keep the reference only in request-scoped memory or an isolated private
  temporary location with a strict quota and deadline. If a file is required,
  create it with restrictive permissions and remove it on success, refusal,
  timeout, cancellation, panic, and process shutdown. Cleanup failure must
  fail the operation and raise a non-sensitive operational signal.
- Do not persist reference audio, derived speaker embeddings, consent receipts,
  or generated clone artifacts in application databases, TTS caches, browser
  storage, crash dumps, traces, or logs. Redact request bodies and filenames.
  Do not include audio hashes in ordinary logs; the hash exists only for the
  consent-authority binding and short-lived operation audit record.
- Use an on-demand CPU-capable resource with one inference slot initially,
  explicit RAM/CPU ceilings, bounded startup/shutdown, and no auto-start on
  ordinary Audio Tools traffic. A request must not trigger an unbounded model
  download. Pin every runtime and model artifact by immutable version and
  digest, and publish the complete dependency license inventory before
  packaging.

## Qualification and promotion gates

Before an adapter or resource is wired into production catalogs, produce
reviewable receipts for: source-bound authorization and revocation; refusal
paths; deletion after success and all failure modes; no audio in logs/cache;
concurrency, byte, duration, time, CPU and peak-RAM limits; clean shutdown;
dependency and artifact licensing; and held-out multilingual intelligibility,
speaker similarity, latency and output-safety quality. Test on the target
host, report hardware/software details, and compare resource use with the
existing small ordinary-TTS route. Upstream benchmark claims are not substitutes
for these measurements.

Until those receipts are accepted, keep \`sopro-tts\` planned and its model
\`candidate\`, with no adapter, engine, runnable artifact, selectable route,
default role, or ordinary-TTS fallback. Keep every other voice-cloning
provider under the same boundary. No model/provider promotion is implied by
this contract.
`;export{e as default};
