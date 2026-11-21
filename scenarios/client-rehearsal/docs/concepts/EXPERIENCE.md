# Experience — Client Workflow Rehearsal

Comparable product: an automation testing workbench used by a senior freelance automation engineer. The primary surface is a rehearsal workspace that makes workflow identity, run status, synthetic calls, seeded fault/repair, and review disposition legible without exposing runtime configuration.

On desktop, use the navigated console shell with a compact engagement list and a single main run detail pane. On mobile, show one list or one rehearsal detail at a time, with a clear back action; keep the primary import/rehearse action available and collapse secondary metadata. The detail surface must distinguish imported, running, failed, needs-review, approved, and changes-requested states by text and accessible status semantics, not color alone.

The first-time empty state explains the synthetic-only promise and offers a made-up example workflow. Loading, partial trace, import error, simulator error, and empty history states each name the next useful action. The import form previews secret replacement counts and rejects arbitrary endpoint targets before execution. The run view exposes an ordered trace and fault/repair comparison. The operator must choose an explicit review disposition before a handoff can be marked approved.

Use the generated design tokens and component-library console shell. Keep keyboard access, visible focus, reduced motion, responsive wrapping, stable `data-testid` selectors, and translated strings. This is an operator console, not a live n8n editor or a client portal.
