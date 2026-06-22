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
// interactive owners. Preemption rule: a requester may reclaim from a target
// ONLY IF the target is "idle" beyond idle_grace AND target.priority <
// requester.priority AND target.protected==0. Age and utilization NEVER trigger
// reclaim.
//
// # 8.4 Adopter / CLI contract — `vrooli capacity <verb> [--json]`
//
//	claim --owner-kind --owner-id --resource-kind vram --preferred <bytes> --floor <bytes>
//	      --priority <tier> [--gpu-index N] [--instance-id ID] [--profile <json>] [--ttl <dur>] [--protected]
//	    -> {claim_id, granted_amount, step, verdict, warnings}
//	heartbeat --claim-id [--ttl]      # renews liveness; does NOT change activity
//	activity  --claim-id --state active|idle   # work-owner reports activity (the idleness truth source)
//	release   --claim-id
//	degrade   --claim-id --to <label> # broker->adopter callback target (adopter implements the resize)
//	list      [--owner ...] [--json]
//	reconcile [--json]
//	sweep     [--json]               # resident-claim liveness driver (see below)
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
// never enforces; it is meant to be driven periodically (system-monitor's
// collector loop, a cleanup pass, or `vrooli capacity sweep`).
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
// Advisory mode (VROOLI_CAPACITY_ENFORCE=off, the V1 default) logs the verdict
// and ALWAYS lets the caller proceed (the caller chooses its own fallback).
// Enforced mode honors the verdict. Decide is a PURE function: no enforcement
// side effects, no nvidia-smi/docker calls (capacity is read through the
// injected CapacitySource seam).
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
// preempt_enabled (bool), auto_stop_allowlist (csv). No silent caps — every
// threshold is logged and tunable.
package capacity
