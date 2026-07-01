// Package capacity is Vrooli's host resource claim/lease arbitration layer.
//
// It introduces a centralized, contention-aware capacity broker so multiple
// GPU/RAM/CPU-intensive scenarios and resources can coexist on one finite host
// without one consumer silently starving the next. The mechanism lives in
// project-level internal/ (reachable by resources, scenarios, and the
// lifecycle — no resource -> scenario dependency); the management/observability
// UX lives in the system-monitor scenario, which reads this ledger (the broker
// never depends on system-monitor). GPU VRAM is the V1 operational target; the
// data model is generic across RAM/CPU for the V2 vision.
//
// This file is the authoritative, frozen contract (plan §8). The schemas,
// verbs, and protocols below are implemented verbatim by the engine, the
// `vrooli capacity` CLI, the lifecycle admission hook, and every adopter. A
// human-facing design note lives beside it in DESIGN.md.
//
// # Patterns, not shared code
//
// The claim/lease engine mirrors the proven patterns of
// internal/scenarioruntime (generation guard / ErrStaleGeneration, heartbeat
// deadline + stale sweep, withTx/withRetryableTx discipline, SchemaVersion +
// PRAGMA user_version, repository-interface-per-concern) but does NOT import or
// refactor it: scenarioruntime's lease engine is welded to its Instance /
// runtime_instances table. We mirror the shape in a new package with its own
// table. Greenfield — no compatibility shims, no dead code.
//
// # 8.1 CapacityClaim (data model, SQLite — table capacity_claims)
//
//	claim_id              TEXT PK            # uuid ("clm-<hex>")
//	owner_kind            TEXT              # "resource" | "scenario" | "op"
//	owner_id              TEXT              # e.g. "whisper", "image-tools", "image-tools:job-<id>"
//	instance_id           TEXT NULL         # links to a scenarioruntime instance when applicable
//	resource_kind         TEXT             # "vram" | "ram" | "cpu" (V1 enforces vram; schema generic)
//	gpu_index             INTEGER NULL      # for vram, which GPU
//	amount_bytes          INTEGER           # current granted amount (the active degradation step)
//	preferred_bytes       INTEGER           # top of the profile
//	floor_bytes           INTEGER           # min-viable
//	priority              INTEGER           # see 8.3 (tier rank: batch=10, service=20, interactive=30)
//	protected             INTEGER           # 1 = never preempt/degrade while active
//	status                TEXT             # "reserved"|"granted"|"degraded"|"released"|"expired"|"preempted"
//	activity_state        TEXT             # "active"|"idle" (work-owner reported; NOT inferred)
//	generation            INTEGER           # optimistic concurrency (mirror ErrStaleGeneration)
//	created_at            TIMESTAMP
//	updated_at            TIMESTAMP
//	last_heartbeat_at     TIMESTAMP
//	heartbeat_deadline_at TIMESTAMP         # liveness; a dead owner's claim is swept to "expired"
//	last_active_at        TIMESTAMP NULL    # last time activity_state transitioned to/refreshed "active"
//	degrade_profile       TEXT(json)        # see 8.2
//
// Liveness is heartbeat_deadline_at: a claim whose deadline has elapsed is
// swept to "expired" by ExpireStaleClaims. Activity is a SEPARATE dimension
// (activity_state) reported by the work-owner — it is never inferred from
// wall-clock age or utilization.
//
// # 8.2 Degradation profile (declared by adopter, data — degrade_profile JSON)
//
//	{ "steps": [ {"label":"large-v3","amount_bytes":...},
//	             {"label":"medium","amount_bytes":...},
//	             {"label":"small","amount_bytes":...} ],
//	  "apply": {"verb":"capacity-degrade","argv":["--to","{label}"]},
//	  "upshift": true }
//
// floor_bytes == the last step's amount_bytes. For image-tools the last "step"
// is "cpu" (off-GPU, amount_bytes 0). "apply" is how the broker asks the
// adopter to step (the adopter implements the resize). "upshift" declares
// whether the broker may move the claim back up when headroom returns.
//
// # 8.3 Priority + protection model (kept minimal)
//
// Tiers, higher wins:
//
//	interactive (30) — user-facing live: whisper transcription, active coding agent
//	service     (20) — resident model servers while idle
//	batch       (10) — image generation, background work
//
// protected=1 is set automatically while activity_state=="active" for
// interactive owners (and cleared when they go idle), so a resource is protected
// only WHILE it is working — dynamic protection, not a static flag. Preemption
// rule: a requester may reclaim from a target ONLY IF the target is "idle"
// beyond idle_grace AND target.protected==0 AND it passes the priority gate.
// Age and utilization NEVER trigger reclaim.
//
// Priority gate (the reclaim-eligibility decision boundary, reclaimEligibleFor):
//
//   - strict default: target.priority < requester.priority. A claim is reclaimable
//     only by strictly-higher-priority work. This is the behavior every claim gets
//     unless it opts in below — byte-identical to the pre-idle-yield engine.
//   - idle-yield opt-in (yield_when_idle=1): when such a claim has dwelt idle
//     beyond its claim-specific idle_grace (or the global policy idle_grace when
//     unset), it yields its capacity to active work at OR ABOVE the
//     idle_yield_floor (a tunable tier lever, default batch). This RELAXES the
//     strict rule to permit equal-priority reclaim — but ONLY for claims that
//     explicitly opted in, and ONLY while idle. Active claims are NEVER demoted.
//     It is how an idle STT resource frees VRAM for an image job, then recovers
//     when it next transcribes. Without the opt-in, an idle interactive claim
//     stays untouchable (strict priority), which is why the activity signal alone
//     is inert — the engine needs this rule too.
//
// Idle-yield is opt-in PER RESOURCE (not a global enforce flip) so the blast
// radius is exactly the resources that declare yield_when_idle. whisper and
// kyutai-stt opt in with longer claim-level idle_grace values so recent speech
// work stays warm while cold-idle STT can yield to image generation.
//
// # The three time axes — "idle" disambiguated (autonomy plan §Phase 0)
//
// The word "idle" is overloaded across the broker; it actually names THREE
// distinct, independent axes, and conflating them is the source of confusion:
//
//   - LIVENESS (heartbeat_deadline_at): is the owner still alive? A claim whose
//     deadline lapses with no observed GPU presence is swept to "expired". This is
//     a binary alive/dead signal, NEVER a workload signal.
//   - ACTIVITY (activity_state ∈ active|idle): is the owner doing work RIGHT NOW?
//     Reported by the work-owner (never inferred). Gates demand-driven reclaim
//     eligibility together with idle_grace (the dwell after a report of "idle").
//     A claim may declare its own idle_grace_seconds; otherwise the global policy
//     idle_grace applies.
//   - IDLE-UNLOAD (idle_unload_ttl): has the owner been idle long enough that the
//     broker should AUTONOMOUSLY free its VRAM, accepting a cold start on next use?
//     This is a NEW axis (autonomy plan). It is distinct from idle_grace:
//     idle_grace bounds *demand-driven* reclaim (someone else wants the space);
//     idle_unload_ttl bounds *proactive* unload (nobody is asking — the broker
//     reclaims because the capacity is sitting idle). Per-resource opt-in
//     (0 = off), with an optional default_idle_unload_ttl lever. The autonomous
//     unload actuates only under enforce=on; advisory logs the would-unload.
//
// # Observed-usage sampling (decaying high-water mark) — telemetry only
//
// Each maintenance sweep samples, per active VRAM claim, the owner's currently
// observed GPU bytes (reusing reconcile's owner attribution) and folds it into a
// DECAYING high-water mark (peak), persisted on the claim as observed_bytes (the
// latest sample), observed_peak_bytes (the decaying peak), observed_at (sample
// time). The peak ages by a half-life lever (observed_peak_halflife) so a stale
// spike does not pin the reservation forever, yet a real working-set peak is not
// erased by a single idle reading:
//
//	peak = max(now_observed, prev_peak * 0.5^(dt/halflife))   dt = now - observed_at
//
// VRAM is non-compressible, so we size to the PEAK, not an average (contract C2).
// Sampling is pure telemetry: it is recorded via RecordObserved, which does NOT
// bump the claim generation (so it never races a concurrent activity report) and
// NEVER feeds the Decide committed math (contract C1 — committed stays = the
// declared reservation; the phantom-free-space problem is fixed honestly by
// right-sizing the declared number and by idle-unload, not by feeding live
// observation into admission).
//
// # Right-sizing recommendation (advisory) — `vrooli capacity recommend`
//
// Once a claim has accumulated enough observed-peak samples, `capacity recommend`
// compares its declared preferred_bytes against observed_peak_bytes and, when the
// peak plus a headroom lever (recommend_headroom) is materially below preferred,
// emits a suggested smaller reservation. It is ADVISORY ONLY — never auto-applied
// (contract C7) — and NEVER recommends below observed_peak + headroom (the
// idle-snapshot trap: a lone idle reading must never shrink a reservation; only
// real-work peak counts). It is the signal/feedback surface a human or operator
// acts on, not an enforcement action.
//
// # Terminal-claim GC (ledger hygiene) — `vrooli capacity gc`
//
// Released/expired/preempted claims are TERMINAL: they hold no capacity and exist
// only as history. Left unpruned the ledger fills with dead rows (the live ledger
// accumulated 71/74 terminal rows incl. test fixtures). GCTerminalClaims deletes
// terminal rows whose updated_at is older than the terminal_retention lever
// (default 24h); active claims are NEVER pruned. The maintenance sweep GCs each
// pass, and `vrooli capacity gc` runs it on demand (reporting count + bytes
// pruned). GC is always safe regardless of enforce mode.
//
// # Reconcile/sweep-driven adoption (claim-on-observe) — declared residents
//
// Sweep only REFRESHES existing claims; a resident that predates the admission
// hook (e.g. kyutai-stt / speaker-verification, up for weeks, never re-admitted)
// shows as UNCLAIMED forever. Claim-on-observe closes that: when the sweep
// observes a GPU consumer that attributes to an owner whose resource.json
// declares a capacity block AND that owner holds no active claim, it CREATES the
// claim on the owner's behalf from the declared spec (contract C6). It is
// idempotent (an owner with an active claim is a no-op), advisory-safe, and only
// fires for owners with a DECLARED block — an undeclared observed consumer still
// only warns (unchanged).
//
// # Activity-source contract (who reports active/idle)
//
// activity_state is the work-owner-reported truth source (never inferred). For a
// given resource there is EXACTLY ONE reporter (single source of truth — two
// reporters would race last-writer-wins). WHERE that reporter sits depends on the
// resource's transport:
//
//   - whisper (request/response HTTP): reported at the RESOURCE EDGE. A host-side
//     reverse proxy in whisper's data path brackets each `POST /asr` (active on
//     request-in, idle after a debounce on response-out) and reports via the
//     `vrooli capacity activity` CLI. The edge is the only place that covers ALL
//     consumers — the host dictation tool and the browser WhisperProvider are
//     clients Vrooli does not own and cannot instrument caller-side. The edge is
//     fail-open: it always forwards the request even if every capacity call fails.
//   - kyutai-stt (websocket streaming): reported CALLER-SIDE (audio-tools brackets
//     the whole transcription session). An edge proxy bracketing individual frames
//     would flap; the caller knows session boundaries the edge cannot see.
//
// So: whisper -> edge; kyutai-stt -> caller-side. This is NOT "the resource always
// owns reporting" — it is transport-driven. audio-tools therefore reports activity
// for kyutai-stt only; it does NOT report whisper (the edge is whisper's sole
// source).
//
// # 8.4 Adopter / CLI contract — `vrooli capacity <verb> [--json]`
//
//	claim --owner-kind --owner-id --resource-kind vram --preferred <bytes> --floor <bytes>
//	      --priority <tier> [--gpu-index N] [--instance-id ID] [--profile <json>] [--ttl <dur>]
//	      [--protected] [--yield-when-idle]
//	    -> {claim_id, granted_amount, step, verdict, warnings}
//	heartbeat --claim-id [--ttl]      # renews liveness; does NOT change activity
//	activity  --claim-id --state active|idle   # work-owner reports activity (the idleness truth source)
//	release   --claim-id
//	degrade   --claim-id --to <label> # broker->adopter callback target (adopter implements the resize)
//	list      [--owner ...] [--json]
//	reconcile [--json]
//	sweep     [--json]               # resident-claim liveness driver (see below)
//	gc        [--json]               # prune terminal claims past terminal_retention
//	recommend [--owner ...] [--json] # advisory right-sizing from observed peaks
//	policy    {get,set} <key> <value>
//
// Resources call these from their lib/docker.sh pre-start/while-running hooks;
// scenarios call from their job runners; the lifecycle calls `claim` at
// admission.
//
// # Resident-claim liveness driver (Sweep)
//
// A claim's liveness is its heartbeat_deadline_at: an owner that stops
// heartbeating is swept to "expired". Op-scoped claims heartbeat themselves over
// their short claim->run->release lifecycle, but third-party resident model
// servers (whisper's onerahmet container, kyutai-stt) hold VRAM continuously and
// have NO shell hook to heartbeat their own claim — so without a driver their
// claim would expire at default_heartbeat_ttl while the container is still
// alive, making the ledger lie (today's adopters paper over this with a
// 6-hour ttl_seconds stopgap).
//
// Sweep(store, snapshot, attr, policy, now) closes that gap by deriving resident
// liveness from the host snapshot: any active non-op claim whose owner is still
// observed holding GPU memory has its heartbeat renewed (refresh happens BEFORE
// the stale sweep, so an observed-alive owner is rescued rather than expired);
// every other active claim past its deadline is then expired as usual. Op-scoped
// and non-vram claims are never presence-refreshed. Sweep mutates the ledger but
// never enforces.
//
// # 8.6 Sweep-driver host + cadence (completion plan §8.2)
//
// The PRIMARY, always-on driver is the platform maintenance pass
// (internal/maintenance.Controller.CleanStaleLocks), which already runs
// ExpireStaleStartingLeases on lifecycle activity; the capacity sweep rides the
// same cadence (no new daemon process). SECONDARY, opportunistic sweeps run on
// every AdmitResource and on `capacity list/reconcile` reads (cheap; cover the
// gaps between maintenance passes). Cadence is the tunable lever
// `sweep_interval` (default DefaultSweepInterval); the opportunistic sweeps mean
// resident liveness no longer depends on a 6h ttl_seconds stopgap. system-monitor
// MAY additionally call `vrooli capacity sweep` while running (belt-and-suspenders
// for its dashboard) but is NEVER required — the broker works with it down.
//
// # 8.7 Resident-claim lifecycle (wired via the lifecycle, not the shell)
//
// Resident model servers (whisper, kyutai-stt, reranker, speaker-verification)
// declare a `capacity` block in resource.json; the broker — not the resource's
// shell — owns their claim lifecycle, because the compose-service driver starts
// them with `docker compose up` directly and never calls lib/docker.sh:
//   - CLAIM on start: the lifecycle admission hook (internal/lifecycle
//     admitResourceCapacity → AdmitResource) records the claim when the resource
//     is started as a dependency. IDEMPOTENT: if an active claim for this
//     (resource owner, resource_kind) already exists it is heartbeat-renewed and
//     reused — a restart/re-admit must not stack or leak a second claim.
//   - LIVENESS while running: the §8.6 sweep presence-refreshes the claim from
//     the GPU snapshot (the container cannot heartbeat itself). The 6h
//     ttl_seconds stopgap is therefore REMOVED — claims fall back to
//     default_heartbeat_ttl, kept alive by the sweep.
//   - RELEASE on stop: when the container disappears its claim is no longer
//     observed on the GPU, so the sweep expires it once the deadline lapses
//     ("killing the container expires it", §10). The explicit
//     `vrooli capacity release --claim-id <id>` verb remains available for
//     op-scoped/manual owners and is a no-op when already released.
//
// # 8.8 Degrade actuator + escalation executor (enforce mode only)
//
// Actuate(ctx, plan EscalationPlan, store, exec ApplyExecutor, policy, now)
// consumes a PlanEscalation result and EXECUTES it (PlanEscalation stays a pure
// planner; Actuate is the orchestration layer it always lacked). For each
// request-degrade action it resolves the target claim's DegradeProfile.Apply
// {Verb,Argv}, substitutes "{label}" with the chosen step, and runs the owner's
// degrade verb through the injectable ApplyExecutor seam (production =
// exec.Command of the owner's CLI; tests = fake — no real exec/docker in unit
// runs). On success it records status=degraded + the new amount_bytes; on
// actuator FAILURE it leaves the claim unchanged and surfaces a warn finding
// (never strand a resource off-GPU). The preempt rung runs only when
// policy.preempt_enabled AND enforce. Debounce (`degrade_debounce`) skips
// re-degrading a target too soon; upshift only when free headroom ≥
// `upshift_headroom` and the target is idle. Every actuation and every skip
// emits an honest log/finding — no silent caps. Actuate runs ONLY in enforce
// mode; advisory logs the plan but does not act.
//
// Upshift (the symmetric counterpart, upshift.go) is IMPLEMENTED: PlanUpshift /
// PlanUpshiftAll / RunUpshift plan and (enforce-only) actuate climbing a
// degraded, idle claim back UP its profile toward preferred when per-GPU free
// headroom clears `upshift_headroom`. It is driven opportunistically from the
// maintenance sweep (idle-time/background — never a synchronous resize on an
// active request), reuses the adopter's resize verb with `--upshift` and the
// `degrade_debounce` window for anti-thrash, and records status=granted at the
// higher amount via UpshiftClaim. Advisory surfaces the would-upshift as a
// recommendation (non-acting); enforce=on actuates. This closes the gap where a
// whisper degraded to `small` under GPU pressure never recovered after the GPU
// freed.
//
// # 8.9 Adopter degrade-verb semantics
//
//   - whisper `capacity-degrade --to <label>` (resource CLI verb): label ∈
//     profile steps; persists the choice as an operator-pin and recreates the
//     container with the smaller ASR_MODEL; idempotent (already-at-target =
//     no-op); REFUSES while activity_state=="active" (the whisper edge marks it —
//     see the activity-source contract) as adopter-side defense in depth; the
//     `--upshift` path recreates larger when idle + headroom returns.
//   - ollama `degrade`: unloads the Nth resident model; idempotent; reported via
//     the planner package.
//   - image-tools SD: existing in-process fp16→CPU resize on a non-grant verdict
//     (verified, not re-built).
//
// # 8.5 Decision contract
//
// Decide(req, snapshot, ledger, policy) returns a Verdict whose Kind is one of:
//
//	Grant        — enough free, or after reclaiming idle lower-priority claims
//	Degrade(step)— grant a lower profile step that fits
//	Queue        — wait, with a position
//	Deny(reason) — cannot satisfy even the floor
//
// Advisory mode (VROOLI_CAPACITY_ENFORCE=advisory, the V1 default — see the
// admission-default note below) records the claim, logs the verdict, and ALWAYS
// lets the caller proceed (the caller chooses its own fallback). Enforced mode
// (`on`) honors the verdict and runs the actuator. Decide is a PURE function: no
// enforcement side effects, no nvidia-smi/docker calls (capacity is read through
// the injected CapacitySource seam).
//
// # Reconciliation contract (plan §7 Phase 2)
//
// Reconcile(snapshot, ledger, policy) -> []Finding. For each observed consumer
// (an attributed GPU process) above policy.TrackingThreshold, classify:
//
//	CLAIMED    — attributed owner holds an active claim covering the usage
//	UNCLAIMED  — no active claim for the attributed owner
//	OVER_CLAIM — observed usage exceeds the owner's granted amount
//
// UNCLAIMED and OVER_CLAIM emit warn-level findings. Auto-stop stays OFF behind
// config + an allowlist; reconciliation only observes and warns in V1.
//
// # Tunable levers (plan §2 control-surface-tunable-levers-design)
//
// Every threshold/policy is a data-declared, tunable lever stored in the
// capacity_policy table and editable via `vrooli capacity policy set`:
// tracking_threshold (bytes), idle_grace (duration), default_heartbeat_ttl,
// reconcile_warn_threshold (bytes over claim), enforce (off|advisory|on),
// preempt_enabled (bool), auto_stop_allowlist (csv), sweep_interval (duration —
// §8.6 cadence), degrade_debounce (duration — §8.8 anti-thrash), upshift_headroom
// (bytes — §8.8 hysteresis), idle_yield_floor (tier — §8.3 the lowest requester
// priority that may reclaim an idle yield-opted claim, default batch),
// terminal_retention (duration — how long a terminal claim survives before GC,
// default 24h), observed_peak_halflife (duration — the decay half-life of the
// observed high-water mark, default 10m), default_idle_unload_ttl (duration — the
// fallback autonomous idle-unload TTL for claims that do not declare their own,
// 0 = off), recommend_headroom (percent — the safety margin added above the
// observed peak when right-sizing, default 20). No silent caps — every threshold
// is logged and tunable.
//
// # Admission default — RATIFIED: advisory (completion plan §8.1)
//
// DefaultPolicy() returns EnforceAdvisory: a started resource's claim is recorded
// and the verdict logged/warned, but the start is NEVER blocked. This is a
// DELIBERATE divergence from the original plan
// (capacity-broker-internal-capacity-arbitration-system-monitor-ux), which
// specified `off` as the default. Advisory was chosen so the ledger is populated
// and observability works out-of-the-box at zero start-path risk.
// VROOLI_CAPACITY_ENFORCE=off remains the byte-identical escape hatch
// (parity-tested in admission_test.go): with it the admission hook is a complete
// no-op and the resource start path is identical to legacy. `on` additionally
// enables actuation (§8.8). The divergence is recorded here and in the original
// plan's note.
package capacity
