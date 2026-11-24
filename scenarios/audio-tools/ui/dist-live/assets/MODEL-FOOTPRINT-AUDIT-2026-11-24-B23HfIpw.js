const e=`# Model footprint audit — 2026-11-24

This is a point-in-time runtime attribution, not a qualification receipt. It was
captured on the Linux/amd64 D1 host at 2026-11-24 11:23 UTC. No model was
started, stopped, unloaded, or requested for inference during the audit.

## Current defaults

| Capability | Resolver result | Declared footprint | Qualification boundary |
| --- | --- | --- | --- |
| Batch STT | Whisper Small (\`whisper-small\`) | 488 MB artifact; prior one-host product smoke measured 716 MiB VRAM | Current default. The resolver result does not mean weights are resident now. |
| Streaming STT | Sherpa Zipformer (\`sherpa-streaming\`) | 310 MB artifact | Current streaming choice. The 11-second paced browser cohort still misses the 1,000 ms first-visible p95 target. |
| TTS | Kokoro 82M (\`kokoro-82m-v1.0\`) | 330 MB FP32 artifact | Qualified catalog default and fallback. Kale Nano remains a smaller, conditional candidate without joined quality and lifecycle qualification. |
| Voice cloning | No selectable/default Audio Tools model | — | Candidate declarations do not have the required consent, transient-reference handling, adapter, or held-out quality/resource evidence. |

The Audio Tools API's model resolver selected Whisper Small and Kokoro without
starting either model. Audio Tools' STT default remains compact; available
smaller Zipformer candidates have earlier evidence of worse first-text timing,
so those results do not support a promotion.

## Host snapshot

\`nvidia-smi\` reported a 16,376 MiB RTX 4070 Ti SUPER with 5,375 MiB allocated
and 8% instantaneous utilization. Its compute-process list attributed 608 MiB
and 2,984 MiB to two Ollama \`llama-server\` processes and 1,380 MiB to the
Reranker process. No Audio Tools, Whisper, Sherpa, or Kale process appeared in
the GPU compute list. Those listed allocations total 4,972 MiB; the remaining
403 MiB is not attributed by the compute-process report.

At the same sample, Audio Tools API RSS was 25 MiB, the API-only lab process
was 19 MiB, the Sherpa server was 7 MiB, Kale's Python server was 16 MiB, and
the Whisper resource control process was 11 MiB. These are host process RSS
values, not model-artifact sizes. A five-second \`pidstat\` interval recorded
0.00% CPU for the idle Sherpa server. No Audio Tools inference request was
running during either sample.

## Interpretation

The observed GPU allocation was in shared Ollama and Reranker processes, not
Audio Tools. Audio Tools still has idle CPU-side resource servers, but their
measured memory and Sherpa CPU use were small in this snapshot. Do not unload
the shared Ollama or Reranker processes based on this audit; their owners and
consumers are outside Audio Tools. Preserve the distinction between catalog
selection, process presence, loaded model weights, and active inference when
comparing future snapshots.

## Follow-up observation

At 2026-11-24 12:45:09 UTC, the GPU again reported 5,375/16,376 MiB allocated,
at 1% utilization. The visible compute processes were Ollama runners at 608 and
2,984 MiB plus the shared reranker at 1,380 MiB (4,972 MiB total); 403 MiB was
not attributed by that process list. No Audio Tools, Whisper, Sherpa, or Kale
process appeared in the GPU compute list.

The same host sample showed RSS of about 23.3 MiB for the product API, 20.0 MiB
for the API-only lab process, 7.1 MiB for the attached Sherpa server, 15.5 MiB
for Kale Nano, and 11.0 MiB for the Whisper activity-edge helper. Sherpa's
\`/healthz\` reported streaming model loaded, TTS model not loaded, and speaker
model not loaded. No inference was initiated as part of this observation. This
does not establish peak or concurrent inference memory.

This second sample reinforces that the observed GPU allocation belongs to
shared Ollama and reranker processes, while Audio Tools' current STT resolver
already selects Whisper Small and Sherpa streaming. The product's current
health snapshot separately reports Sherpa STT and Kale TTS available, with
Whisper and Kokoro unavailable. Therefore \`auto\` can use the smaller Kale Nano
route while healthy even though Kokoro remains the static 330-MB FP32 TTS
catalog default. The sample does not measure Kale's loaded-weight memory or
peak use; keep it conditional until quality and lifecycle evidence supports
promotion. Voice cloning still has no selectable default.

## Evidence

- \`vrooli scenario status audio-tools --json\`: product instance healthy,
  serving build \`sha256:4e4e0059945491efb6f996b99828e12cac10c5ae0e24722b6dc9b6c4c4dc79f4\`.
- \`audio-tools model resolve --capability stt --json\`: \`whisper-small\`,
  488 MB, catalog default.
- \`audio-tools model resolve --capability tts --json\`: \`kokoro-82m-v1.0\`,
  330 MB, catalog default.
- \`nvidia-smi\` and process RSS sample at 11:23 UTC; \`pidstat -p 2557133 1 5\`
  recorded zero CPU for the idle Sherpa process.
- Follow-up \`nvidia-smi\`, process RSS, and Sherpa \`/healthz\` sample at 12:45 UTC;
  \`audio-tools health show --json\` at 12:46 UTC reports the effective current
  provider states.
- Current limitations: these are two idle samples on one host. They do not
  measure inference peak memory, cold-start, concurrency, or per-model energy.
`;export{e as default};
