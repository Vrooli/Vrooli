# Local song generation

`songgen` is a managed offline resource for MiniMax Music3 NF4 and HeartMuLa.
The two profiles have independent pinned dependency closures. One isolated
worker owns generation; exit releases its model allocations. Model bundles
remain on disk across requests. Neither profile currently accepts a voice
reference; style and vocalist descriptions are soft conditioning.

Install the runtime through `vrooli resource install songgen`. Acquire only the
selected weights through the control-plane owner, then start the resource:

```sh
vrooli resource acquisition data songgen --group minimax-music3-nf4
# Select heartmula-3b instead when that profile is wanted.
vrooli resource start songgen
```

At least one complete group is required; missing groups fail before process
launch with the recovery command. Ordinary install/start skips on-demand groups,
so a profile removed by retention stays absent until it is requested again.
Group acquisition uses the canonical data fence and does not start a model.
Music generators share a lifecycle exclusive
group: finish work and stop the previous member before starting another.
Compose through Music Tools with an explicit `minimax-music3-nf4` or
`heartmula-3b` style variant. Keep structured BPM/key unset; describe them in
the caption. MiniMax uses 1–100 denoising steps and actual CFG guidance
(upstream default 1.7); HeartMuLa uses steps 0 and its actual CFG guidance.
Duration is an upper bound; early model endings are recorded as actual duration.

Acquiring both model groups uses about 51 GB plus the two runtime closures.
MiniMax alone is about 28.5 GB; HeartMuLa is about 22.4 GB.
The default instrumental path remains Stable Audio 3. ACE remains the preferred
lyric model until a later listening decision changes that policy. Human
feedback preferred MiniMax seed 170104 for realism and ACE take 2 for the intended
style; integration does not claim a guaranteed song quality or artist identity.

Model cleanup uses recorded use, not shard modification times. Preview with
`resource-songgen model-cache preview`; save that JSON, then use
`resource-songgen model-cache apply --preview FILE`. The default minimum idle
age is 30 days. Unknown use, pins, changed previews and active generation or
acquisition prevent deletion. Pin with
`resource-songgen model-cache pin --profile minimax-music3-nf4`.
Use `--pinned=false` to unpin. Whole bundles are removed; exported Music takes
and operator evidence are outside this cleanup boundary.

The HTTP surface is loopback-only: `GET /health`, `POST /v1/compose`, completed
operation `/audio` and `/receipt` reads, and `POST /v1/models/retention`.
Requests and receipts are bounded JSON. Cancellation kills and reaps the worker;
Linux parent-death signalling also covers hard supervisor loss.

Run focused checks with `make -C resources/songgen test`. Production qualification
also requires a governed activation and fresh Music pipeline generation for each
profile; source tests do not prove sound quality or current runtime adoption.

Operation scratch lives under the resource's `data/outputs` directory. Its
one-day age limit exceeds the two-hour worker deadline. Byte-pressure deletion
is disabled there, so an active stage cannot be selected just to satisfy a size
ceiling. Music's exported audio and native receipts remain durable.

The supervisor retains a logical resource capacity reservation until normal
resource stop; it does not retain a model allocation. Stop Songgen when its
session is complete before activating another music member or releasing that
reservation for other GPU work. Host capacity is advisory on the current host;
a reservation cannot prevent unrelated services from allocating GPU memory.
Insufficient free VRAM fails the job without publishing a take. Use the
control-plane capacity owner to investigate contention; never silently evict
another active tenant from a song request.
