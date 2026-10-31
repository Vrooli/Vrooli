# Local song generation

`songgen` is a managed offline resource for MiniMax Music3 NF4 and HeartMuLa.
The experimental `yue2-q8` profile adds native full-score planning through pinned
audio.cpp 0.9.3. Its Q8 main/F16 VAE weights retain their CC-BY-NC terms; it is
not a new preferred model or a quality acceptance claim. The resource owns its
transaction-local native child, timeout, parent-death binding and model cache.
Acquire only through the declared `yue2-q8` data group after capacity and expanded
storage preview. The native child binds loopback with UI/management disabled,
uses CUDA and four threads, and exits after each operation. There is no CPU or
provider fallback. Generated ABC/semantic witnesses and truncation flags remain
in the resource receipt; a token-ceiling hit is not complete-song acceptance.
The profile is under qualification until governed artifact composition,
installation and actual full-song evidence are recorded.
YuE2 also accepts an optional inline `score` containing ABC with X/K headers,
bounded to16KiB of UTF-8. Paths, URLs and file-loading options are not score
inputs. Native execution must report `abc=provided`; the exact original score
and generated semantic stream remain in the receipt. A supplied score's
untruncated flag describes retained input, not proven acoustic adherence.
The two PyTorch profiles have independent pinned dependency closures. YuE2's
published native server also uses NCCL from the pinned HeartMuLa closure;
this loads a shared library without importing Torch or loading HeartMuLa.
Bounded native launch and runtime log tails survive failed-operation cleanup
through the supervisor's failure record. One isolated
worker owns generation; exit releases its model allocations. Model bundles
remain on disk across requests. Neither profile currently accepts a voice
reference; style and vocalist descriptions are soft conditioning.

For a score-controlled original, use the maintained resource owner:

```sh
resource-songgen content compose --request original-song.json --out new-take \
  --expect-artifact SHA256_FROM_MANIFEST
```

The bounded request contains the existing `profile`, `prompt`, `lyrics`,
`duration`, `seed`, `steps`, `guidance`, `expected_revision` and
`timeout_seconds` fields, plus optional inline `score`. This command currently
qualifies only `yue2-q8`. It checks the selected artifact before generation,
uses canonical caller attribution at the declared activity edge, refuses
redirects, and exports verified `audio.wav`, `receipt.json`, `request.json`,
`score.abc` and `semantic.json` into a new directory without replacement.
SIGINT, SIGTERM and caller timeout cancel the owner request; reconcile owner
state before retrying an uncertain transport result. Normal resource stop
remains the operation for closing the session and its capacity reservation.

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
Compose through Music Tools with an explicit `minimax-music3-nf4`,
`heartmula-3b` or `yue2-q8` style variant. Keep structured BPM/key unset; describe them in
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
