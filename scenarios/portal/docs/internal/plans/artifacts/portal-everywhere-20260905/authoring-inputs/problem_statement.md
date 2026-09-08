The operator began with a concrete friction: an agent takes too long to identify a TV, understand its controls, inspect state, and perform an ordinary action. The desired improvement concerns agent orientation, tool round trips, visual reasoning, and repeated workflow discovery—not only backend execution time.

The deployment-manager skill/program pattern motivated analogous BAS and Device Control improvements. Those earlier changes introduced reusable, versioned workflows, guarded promotion, outcome capture, and measurable learning. This plan extends that pattern to the whole desktop and the ecosystem entry point.

The old vrooli-assistant describes a hotkey overlay for issue capture and agent spawning. It does not constitute a current portable desktop-control product. Initial discussion considered rewriting it as a separate desktop automation engine. Subsequent inspection found existing host-desktop ownership in Device Control. The selected design therefore migrates useful Assistant UX into Portal and extends Device Control instead of duplicating its execution model.

### Current evidence and limits

| Area | Inspected fact | Consequence |
|---|---|---|
| Portal | Chat, agentchat, search, completion, and readiness domains exist. | Extend existing features and retain conversation state. |
| Portal deferred domains | Scenario embeds and voice are deferred in DOMAINS.md. | Deliver them here; do not assume completion. |
| Device Control | host-desktop uses DISPLAY/import/xdotool on Linux and screencapture/osascript on macOS. | Replace weak probes and extend native capabilities. |
| Desktop input | Current host adapter accepts pointer events only. | Keyboard, text, drag, semantic actions, and session targeting require implementation. |
| Windows | Current host-desktop explicitly reports unavailable. | Deliver a real native adapter and live Windows evidence. |
| Readiness | Executable discovery currently contributes to release-grade declarations. | Executable presence cannot prove permissions or successful interaction. |
| Shared targets | api-core/targetmodel already separates transport, trust, readiness, and capabilities. | Extend existing semantics instead of creating a second inventory. |
| Web Console | TargetCatalogService and machine/device projections exist. | Share contracts and proven UI pieces; retain terminal ownership. |
| Bridge | Attached-device records and sequenced interactive frames exist. | Reuse topology and transport; add desktop semantics explicitly. |
| Bridge stream | Open/Resize currently contain terminal-specific fields. | Preserve terminal behavior while adding typed desktop sessions. |
| Desktop ramp | Vanilla Electron includes tray/native plumbing. | Add a governed extension contract instead of copying templates. |
| Live desktop harness | scenario-to-desktop/api/livedesktop provides Linux validation sessions; non-Linux backend is unavailable. | Reuse test facilities without making them the production desktop engine. |
| Skills | Portal has no scenario-owned skills directory in this inspection. | Build its usage and improvement setup with real sensors. |

These are source observations at authoring time. They are not live acceptance results. Earlier conversation reported BAS/Device Control validation; that report is historical context, not a fresh regression oracle. Recheck mutable sources before implementation.

Related Plan Manager records are preserved in related-plans.json. Reuse Portal v0's implemented chat/readiness foundation, existing terminal sizing/session invariants, desktop packaging work, and Bridge enrollment work. Do not infer implementation from draft/active status. Read phase evidence and last activity. This plan extends those capabilities; it does not blindly reexecute their old tasks.
