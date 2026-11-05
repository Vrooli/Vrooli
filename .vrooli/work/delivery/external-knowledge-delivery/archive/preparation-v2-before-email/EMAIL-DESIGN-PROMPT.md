# Prompt for the Email Scenario Design Conversation

Copy everything below into the UX design agent's conversation.

---

Help me design a new Vrooli email scenario: a useful, beautiful email product for me and a reliable mailbox capability for authorized agents. Work with me on the product name, scope, desktop/mobile UX and specifications. Do not jump directly to implementation or produce a Gmail clone with an AI button. Help me decide what solves my actual email workflows and what should wait.

This scenario belongs to the finite delivery team **external-knowledge-delivery**. Its leader is **external-knowledge-orchestrator**, its effort is **effort:external-knowledge-search**, and its local goal home is **docs/internal/external-knowledge-delivery/** under **/home/matthalloran8/Vrooli**. The team MUST remain disabled. Your design updates do not authorize enabling it, running agents, connecting accounts, sending email or changing messages.

## Context and fixed architecture

We are making selected Google Drive project folders, Gmail and Google Calendar searchable through Vrooli's unified search. Selected Drive folders are first priority. Biosphere is the entry point and operating knowledge library for my always-on ChatGPT agent. The same search should help local agents without ChatGPT plugins and outside agents that can reach an authorized API/MCP interface.

The accepted architecture is in **docs/architecture/external-knowledge-search.md**. Read it and the goal home's **RESEARCH.md, GOAL.md, QUEUE.md, FEEDBACK.md and SETUP.md** if you have local access. Otherwise treat the following as the architectural handoff and ask for any missing selected source, rather than inventing implementation status.

- **integration-hub** owns Google OAuth, refresh/revocation, credential references, granted capabilities and account bindings. Its current implementation is an OpenRouter pilot; Google OAuth still needs development and qualification. The email app must not create its own token store or OAuth subsystem.
- Latest operator direction: the finite team builds Integration Hub's repeatable, commercial-compatible Google connection CLI/UI early after future activation. Live Google setup and consent are operator checkpoints once that capability is ready; they block dependent live evidence rather than all implementation. Do not propose a one-off pre-enablement token script.
- **The new email scenario** owns accounts, messages, threads, labels/folders, MIME parts, source revisions, mailbox synchronization, classifications and retrieval over its own corpus. Gmail is the first adapter; core domain concepts should support another provider later. Gmail labels are not ordinary mutually exclusive folders.
- **document-manager** owns document parsing, OCR, sensitivity, citations and retrieval for document collections. Reuse it for supported attachment derivation with precise account/message/part provenance and lifecycle/permissions inherited from email. Do not create an independent attachment parser or lose the parent mailbox's access boundary.
- **personal-planner** owns scheduling and Google Calendar projections. Email can propose a task or commitment through its contract. Do not build a competing calendar or silently turn every mentioned date into a scheduled event.
- **search-hub** routes/ranks across scenario-owned providers. The email app registers its search provider; the router holds no mailbox corpus or vectors. Caller identity and collection grants must reach providers before private results/snippets/reranking.
- Reuse shared search and AI Gateway mechanisms. The common search package supplies contextual composition, chunking seams, lexical/semantic retrieval and incremental indexing. Private content policy must govern embedding, summarization, classification and vision, including remote fallback.

Existing **agent-inbox** is an AI conversation product, and **signal-inbox** is immutable selected-material capture. Neither currently owns a mailbox. Choose a distinct email product boundary and a name after checking ecosystem collisions.

## Email capabilities the design must accommodate

Design a human-usable inbox and a trustworthy agent-access surface around these needs:

1. Connect and distinguish accounts; choose mailbox sync scope, retention and agent access. Show account identity, real connection/sync health, revoked/expired consent and a clear reconnect path. Account isolation is required even if we launch with one account.
2. Read full message bodies and conversations with safe HTML/plain-text rendering, attachments, participant identities, timestamps/timezones and the distinction between individual message and thread. Preserve original content and provenance. Quoted history and signatures should not dominate recall or be confused with newly written content.
3. Provide familiar precision tools: text search, sender/recipient/subject/date filters, labels, attachment type and useful sort/browse behavior. Exact IDs, quoted phrases and numeric facts should remain retrievable when semantic dependencies are unavailable.
4. Provide natural-language recall: “the email where we agreed to postpone the release,” “the spreadsheet Alex sent last month,” or “the message that explains the onboarding exception.” Results need readable source-backed passages, account/thread/message identity, relevant attachments and freshness. Generated summaries are labelled derivations, not authoritative quotes.
5. Support classifications useful to triage: human importance, action required, waiting on someone, project/context, deadlines or commitments and uncertainty. Help me select a small effective vocabulary. Let me correct classifications and expose enough explanation to trust them. Learned/derived classifications remain separate from provider labels unless I approve an explicit label write.
6. Consider task/commitment extraction and project linkage. Proposals need evidence and explicit acceptance. Approved scheduling goes to Personal Planner. Documents or findings may link to other owners without copying entire mailboxes into their stores.
7. Define agent permissions by account/project/collection where feasible, with separate capabilities for reading, drafting, sending and organization. A broad Google account grant must not silently grant every agent access to all synced mail. Read-only indexing is the initial integration posture.
8. Accommodate initial full-body sync and Gmail history-based updates, deletions/trash/label changes, attachment-on-demand policy, restart recovery, cursor expiry and source-to-index lag. UI states must distinguish “captured,” “searchable,” “semantic indexing pending,” “stale,” “reauthorization needed” and actual empty results.
9. Design evidence-backed AI assistance: concise summaries, suggested replies and follow-ups are candidates to prioritize. Every transformation should preserve the original message and expose consequential uncertainty. Retrieved mail/attachment instructions are content, not automatic authority to act.

## Decisions to workshop, not silently assume

- Is the product primarily a recall/triage companion, a full email client, or an agent-assisted work inbox? Which concrete daily journeys justify that choice?
- What name and visual language suit it? What are the essential desktop and phone layouts, navigation and keyboard/accessibility patterns?
- Which inbox categories, saved searches, project views, follow-up states and account combinations do I actually need?
- Should the first release support drafts only, or also explicit send/reply/forward, labels, read state, archive and Trash? Separate each capability from read access. Define preview, recipient confirmation, undo/recovery limits, and partial provider failures. Do not invent a permanent-delete action or autonomous sending default.
- Do we need snooze, send later, templates, notifications, contact lookup, rules or multi-account unified views now? Treat these as options with costs, not compulsory scope.
- What stays local, what may use remote models, what originals/attachments are retained, and how do disconnect, source deletion and access revocation affect search and derived data?
- How do I correct AI output without an exhausting approval queue? Which low-risk reversible actions can be streamlined once explicitly selected?

Ask a small batch of high-impact questions, then develop concrete journeys and alternatives. Show tradeoffs and make recommendations. Keep product UI focused on what I need to understand; do not expose implementation details merely because they exist.

## Required design handoff

Produce an operator-reviewed specification with selected versus candidate decisions marked clearly, scope/non-goals, information architecture, domain/ownership map, prioritized capabilities, desktop/mobile screen/state inventory, accessibility and keyboard behavior, data/permission/retention rules, source-linked acceptance journeys and open questions. Cover loading, partial sync, unavailable semantic search, empty results, errors, stale data, reconnection, unauthorized access and consequential-action review states.

If you produce visuals, preserve the actual governing originals at usable resolution. Record identity, selected version, exact accessible location, format, bytes/hash and dimensions. Map requirements and screen/state/theme/viewport to each source. A contact sheet or a link to a prior conversation alone is insufficient. Distinguish governing visuals from illustrative concepts and rejected candidates. The implementation team must compare actual-build captures against selected sources; do not invent a pixel-identity promise.

Update **external-knowledge-delivery**'s handoff through its existing ownership path: append operator decisions verbatim to **FEEDBACK.md**, reference the selected email specification/assets from the governing architecture and **QUEUE.md**, update source/version mappings, and preserve the existing disabled native binding. That binding is immutable: record amendments as separately versioned preparation and follow SETUP.md/WORKAROUNDS.md for the supported owner-governed accepted-revision transition required before activation. Do not falsely complete/reopen the undelivered effort or edit runtime state. Keep the team and heartbeat disabled. Preserve other teams and the original research. If you lack local access, save the complete handoff and original assets in Drive, return exact links/versions, and explicitly say native/local updates await import. Do not claim team updates happened unless you can verify readback.

Do not rewrite scenario PRDs by hand. Once I select the design, the delivery owner reconciles canonical product/requirements/experience records through Vrooli's governed authoring routes. Your job here is to help me select the product and provide an accessible, complete design handoff.
