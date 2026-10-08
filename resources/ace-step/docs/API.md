# ACE-Step API

The managed service listens on the upstream resource port and exposes:

- `GET /health` — readiness metadata, selected variant, planner, device, and
  initialization failure details.
- `GET /v1/variants` — installed/default DiT and planner identities.
- `POST /v1/compose` — one generated WAV. The request accepts `caption`,
  `lyrics`, `duration`, `bpm`, `keyscale`, `seed`, optional inference settings,
  and an explicit `rewrite_caption` opt-in.
- `POST /v1/capacity/degrade?to=full|offload-dit` — diagnostic rung operation.

The resource produces one 48 kHz stereo WAV per request. Batch orchestration,
provenance, and pool state are owned by `music-tools`.
