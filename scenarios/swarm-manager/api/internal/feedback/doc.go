// Package feedback captures user-initiated signal against a whole
// initiative. It is the "living initiative" surface: the user submits text
// (and eventually images / attachments), an agent round produces a
// structured Proposal against the initiative's item graph, and the user
// decides which mutations to accept. The apply layer — separated out into
// the proposals package so it's reusable by review and future research
// flows — runs the accepted mutations.
//
// File layout per initiative:
//
//	initiatives/{name}/feedback/
//	  round-001-ui-rewrite/
//	    feedback.json     ← submission + thread + proposals + decision
//	    attachments/...   ← reserved; attachment store lives alongside
//	  round-002-.../
//	    ...
//	initiatives/{name}/.feedback-lock   ← single-agent mutex, see lock.go
//
// Package boundaries:
//
//   - feedback owns round lifecycle, thread state, and folder layout.
//   - feedback never mutates items or initiatives directly. Every mutation
//     to the item graph goes through proposals.Applier, which in turn
//     delegates to backlog / initiatives / execution services.
//   - Agent spawning is an injected interface (AgentSpawner) — the package
//     never imports agent-manager directly, so it stays unit-testable.
//
// This package ships the core: types, disk store, on-disk lock, and the
// service that orchestrates round lifecycle + apply. HTTP handlers,
// multipart attachment upload, and the skill-side prompt wiring live in
// their own files and are added in subsequent workstreams.
package feedback
