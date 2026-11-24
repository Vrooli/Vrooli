---
name: "react-component-library"
description: "Find a reusable React component, discover a page or flow, and build it as a forkable workspace with RCL assets. Use when searching RCL, choosing a curated workspace, importing HTML/CSS into the Design workspace, or checking component reuse and screen quality."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  revision: 1
  modes: ["tools"]
  tags: ["react", "reusable component", "component library", "design workspace", "template", "HTML import", "search"]
  requires:
    scenarios: ["react-component-library"]
    commands: ["react-component-library search query", "react-component-library workspaces list", "react-component-library workspaces fork"]
  origin:
    kind: "authored"
---

## Tools focus: React Component Library

Prioritize an existing RCL asset or curated workspace that matches the requested outcome, and preserve its provenance and current state while making changes.

### Boundary

Use this skill to find, inspect, compose, import, and validate work in React Component Library. It covers the five-destination workbench and the Workspace → Page → Display → State model. It does not publish component releases, edit shared catalog source, or claim that imported HTML behavior is executable. Use the component asset-update workflow for publishing and `ux-capture` for standalone page capture.

### Choose the shortest matching path

| Need | Do this | Stop when |
|---|---|---|
| [S1] Find a reusable component | Run `react-component-library search query "button" --kind asset`. Open the result in Library and inspect its stories, contract, and release status. | The asset's published version and evidence match the intended use. |
| [S1] Find a page or flow | Run `react-component-library search query "landing page" --kind workspace --kind page`. Compare curated, authored, imported, and forked workspaces by their contents and evidence. | A workspace realizes the requested outcome and its current states are suitable. |
| [S1] Browse what exists | Run `react-component-library search query` with no query, then choose a destination or facet. | The result kind and availability status match what you need. |
| [S1] Ask for a shortlist | Run `react-component-library search query "Which workspace best fits this landing page?" --ask`. Check citations against the indexed workspace or asset before using the answer. | Every recommendation points to a result you can open and inspect. |
| [S0] Start from a curated design | Open Design, select the curated workspace, and choose **Fork**. Keep the source workspace read-only. | The editable fork records its curated origin and has a new identity. |
| [S0] Bring in an HTML design | Open Design, open or create an authored workspace, and import the HTML/CSS file into a page. Select a region, inspect its source styling, then replace only when a searchable RCL match preserves the needed layout and behavior. | The screen remains visually acceptable and imported/custom versus RCL-backed pieces are visible. |
| [S0] Move or reorder a layer | Use the layer tree or canvas drop target. Save and reload the workspace before relying on the new structure. | The persisted tree has the intended parent and order. |
| [S0] Check screen health or drift | Open the workspace's Checks and visual comparison views. Treat missing, stale, or unauthenticated capture evidence as unavailable. | The health evidence and baseline comparison refer to the current workspace revision and state. |

### Durable operations

- Use `react-component-library workspaces list` to find saved workspaces.
- Use `react-component-library workspaces get "workspace-id"` to inspect a workspace aggregate.
- Use `react-component-library workspaces fork "workspace-id"` to create an editable copy of a curated workspace.
- Use `react-component-library workspaces validate "workspace-id"` before accepting structural edits.
- Use the UI for HTML import, visual replacement, drag and drop, state capture, and comparison; the CLI does not replace those visual authoring controls.
- Keep curated templates immutable. Promote a new template only through the reviewed workspace seed workflow described in `path:scenarios/react-component-library/docs/internal/WORKBENCH-PROGRAM.md`.
- When search reports semantic service degradation, continue with lexical results and state the degraded status. Do not present Ask as the source of truth.

### Troubleshooting & Edge Cases

| Signal | Response |
|---|---|
| Search index is stale or unavailable | Read the reported status, use only available results, then rerun search after the index owner recovers. |
| A result is a page type or unavailable asset | Treat it as an intent/specification until an implemented workspace or published asset is linked. |
| A curated workspace is read-only | Fork it; do not edit its seed or bypass the write guard. |
| A capture needs authentication | Report capture as unavailable and preserve the current evidence. Do not synthesize a successful receipt. |
| HTML import omits a script-driven interaction | Recreate the interaction as native workspace behavior only when the Workspace contract supports it; otherwise retain it as an explicit parity gap. |
| A candidate component changes the reference or interaction contract | Keep the imported layer custom and record why it has no safe RCL replacement. |

