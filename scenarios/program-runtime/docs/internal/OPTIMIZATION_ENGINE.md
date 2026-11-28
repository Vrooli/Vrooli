# Governed optimization engine

Program Runtime owns the durable run queue and common ordering contract for model-role and desktop-startup comparisons. Domain owners own acquisition, evaluation, rights facts, disk leases, promotion, rollback, and deletion. The engine does not shell out to resource commands or invent replacement policy.

## Invocation

The Connect surface is `OptimizationService.Submit` and `OptimizationService.Wait`; the governed bindings are `program-runtime/optimization/submit` (destructive) and `program-runtime/optimization/wait` (read). A Program Runtime program submits one immutable JSON request and waits once for a stage change or terminal result. A nonterminal wait is a durable job; its worker continues without the caller. Callers do not poll.

The API journal stores the request, current stage, each owner receipt, revision, retry count and retry deadline in Program Runtime's configured SQLite database. Startup recovers nonterminal rows. At most 1,000 runs may be active; repeated submission of the same immutable run is idempotent, while a new run is rejected with resource-exhausted when the bound is full. One worker runs at a time, and `Engine.Advance` also serializes callers by run ID inside the process; this prevents duplicate concurrent owner calls from direct/recovery callers without retaining lock entries for completed runs. Lock waiters honor request cancellation. Submit rejects duplicate JSON object keys at any depth before decoding its typed request. Transient owner failures use 5s/15s/30s backoff and stop after three failures for a stage. Missing adapters and explicitly unconfigured owner operations refuse immediately instead of consuming a retry window. Each owner call receives the stable run ID and stage identity and must replay an existing receipt after an uncertain interruption. The keyed lock is process-local; it is not a multi-instance lease or fencing protocol.

The engine obtains rights and disk preflight receipts before acquisition and evaluation, then rechecks both after evaluation before promotion. This prevents an unattended run from pulling or measuring a candidate before its owner-backed rights and capacity gates pass. Preflight receipts bind the immutable expected candidate identities; post-evaluation receipts bind the acquired content digests. Both proof sets must be current at their respective stages, and the post-evaluation proofs remain current through promotion and cleanup.

## Pinned request and owner receipts

`optimization.Request` pins `run_id`, `target`, `subject`, candidate identities and any expected artifact digests, optional predecessor, suite ID, ordered case IDs and minimum case count. Run, target, suite, candidate, predecessor, artifact-digest and case identifiers must be canonical without surrounding whitespace, preserving exact denominator and owner-operation identities. A one-candidate run must pin a predecessor because the engine requires a cleanup receipt after promotion. Target adapters validate stronger requirements: the LM6 `ollama-role` showcase requires exactly three `chat.default` candidates with distinct expected artifact digests and at least 20 fixed cases; `desktop-startup` requires one new candidate with a digest, a predecessor, and at least 30 paired cells (5 pairs × cold/warm × 3 phases).

`optimization.SubjectDigest(request)` is SHA-256 over deterministic JSON of the target, subject, candidates, expected artifact-digest map, predecessor, suite and ordered cases. `optimization.AcquiredSubjectDigest(request, evidence)` binds the actual digest map from the durable acquisition receipt. Preflight rights/disk proofs use the request digest; post-evaluation proofs use the acquired digest. Every proof carries an allow decision, receipt ID and future expiry; disk proofs must also declare positive required bytes and sufficient available bytes. An allow decision with a zero or missing capacity estimate is invalid.

Owner stage receipts carry stable `receipt_id` and `stage` identities:

- `preflight_rights`, `preflight_disk`: current owner proofs for the immutable candidate expectations, required before acquisition and evaluation.
- `acquire`: `details.candidate_artifact_digests` maps every candidate ID to a nonempty owner-observed artifact digest. Every acquired digest must be distinct, and any supplied expected digest must match exactly.
- `evaluate`: `details` contains the pinned `suite_id`, exact ordered case count/digest, and `candidate_results` for every candidate. Each result contains case count, exact case digest and a finite scalar score.
- `rights`, `disk`: fresh owner proofs bound to `AcquiredSubjectDigest(request, evidence)` and checked again before promotion and cleanup.
- `promote`: disposition `promoted`, pinned `winner_id`, and exact `artifact_digest`.
- `cleanup`: disposition `cleanup_complete`; `details.removal_receipts` maps every losing candidate and any predecessor to a nonempty owner receipt ID.

Any mismatch, missing receipt, stale proof, insufficient headroom, absent adapter or incomplete denominator refuses the run. Promotion cannot precede fresh post-evaluation gates; cleanup cannot precede promotion. The run is complete only after every required cleanup receipt is present.

## Current adapter integration state

The durable engine, Submit/Wait API, and Program Runtime declaration are source-backed. A fixed root Connect boundary and Program Runtime Ollama adapter expose owner-confirmed inventory, policy and rights facts; it refuses when evaluation, rights proof, optimizer disk admission, promotion or cleanup owners are unavailable. The engine preflights rights and disk before acquisition/evaluation, so current role runs refuse before resource inspection. `scenario-to-desktop/api/desktopoptimize` has staged measurement/promotion/cleanup APIs with separate preflight and final proof binding, plus a fixed typed Connect service. Program Runtime registers a desktop-startup adapter that validates the mapped request and owner proofs but refuses before RPC because no genuine desktop artifact acquisition owner exists; the desktop API is configured without live evaluator/artifact lifecycle owners and fails unavailable. These are integration increments, not either target showcase.

The next implementation step is to connect live evaluation, shared rights, capacity admission, promotion, deletion and recovery owners, then register the staged desktop adapter. The fixed Ollama Connect method only routes its one typed owner inspection request through the standard control-plane resource runner; it exposes no general CLI execution. LM8, LM10 and LM11 must call the shared Program Runtime program once their owner adapters are available.
