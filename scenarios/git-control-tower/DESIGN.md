# Git Control Tower workspace design

Git Control Tower is a diff-first repository workspace. Advisory results are
secondary interpretations of one explicitly identified subject; they never
silently select a staged set, checkout a branch, run hooks, or publish to a
host.

## Information hierarchy

1. Subject bar: repository identity, scenario/file scope, source standing, and
   freshness or stale warnings.
2. Changes and diff: the selected evidence remains central on desktop.
3. Inspector: summary, findings, drafts, evidence, and provenance open beside
   the diff on wide screens and as full-width views on mobile.
4. Human handoff: any stage, commit, merge, undo, comment, PR, or release
   action shows exact destination, revision, preconditions, and verified human
   authority before it can be attempted.

## State contract

Every advisory surface renders loading, empty, partial, unavailable, failed,
and stale states. “No findings” means only that examined evidence produced no
finding; it never means omitted or unavailable evidence passed. Host capability
states are shown per capability, not collapsed into one connected/disconnected
badge.

## Responsive behavior

Desktop preserves the file list and diff as the primary workspace with an
optional inspector. Mobile uses the existing tab/panel navigation and full
width detail views. Subject identity and scroll/focus state remain attached to
the selected operation when moving between views.

## Draft safety

Commit, pull-request, and release text is editable and revisioned separately
from repository state. Regeneration displays a comparison and preserves human
edits. Trailer chips distinguish owner-resolved, legacy, ambiguous, and
unresolved references. A draft button cannot stage, commit, checkout, fetch,
push, publish, or grant authority.
