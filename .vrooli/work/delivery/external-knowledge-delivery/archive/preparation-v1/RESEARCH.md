# Research and Source Evidence

Observed 2026-10-14 in the shared checkout. These are dated observations, not delivery certification. The source manifest records exact local paths/hashes; canonical sources remain live and must be re-read at admission. New research appends a dated finding and updates the manifest. Resolved historical material may move to archive; do not erase it.

## Operator context

The operator uses selected Google Drive project folders as the knowledge library and operating procedures for an always-on ChatGPT agent. Biosphere is the entry point. The requested capability serves local agents without ChatGPT plugins as well as authorized outside agents. Drive is first priority, with shared architecture for Calendar, Gmail and later external sources. The operator approved provider-owned projections, an integration-hub Google connection, a document-manager contract extension, and some retained searchable content. The operator requested anticipated-question enrichment, format-aware chunking and a finite delivery team kept disabled while OAuth setup and email UX are prepared.

## Repository observations

| ID | Fact and limit | Evidence |
|---|---|---|
| R01 | Search Hub is a thin federated router: no authoritative content/vectors or provider-specific client code. | scenarios/search-hub/docs/concepts/ARCHITECTURE.md |
| R02 | Its query contract has filters/scoping, but these do not establish caller authorization. Observed fan-out applies descriptor headers and does not propagate caller identity. Recheck current authentication owners before implementation. | packages/proto/schemas/search-hub/v1/routing/routing.proto; scenarios/search-hub/api/handlers/routing/connect_handler.go; scenarios/search-hub/api/internal/routing/router.go; scenarios/search-hub/api/internal/providers/call.go |
| R03 | Document Manager explicitly limits federation because caller identity is absent. Its planned corpus policy is opt-in and confidential/secret units never federate under that contract. | scenarios/document-manager/PRD.md OT-P0-018; docs/internal/DECISIONS.md 2026-08-07 federation row |
| R04 | Meta Optimization Manager consumes coverage/condition/eval evidence. Active/reachable/fresh passing direct and federated evidence are distinct. Architectural Answer coverage is not the same as personal-document search coverage. Some architecture documentation still calls it draft although domain code and later model docs exist. | scenarios/meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md; ARCHITECTURE.md; README.md |
| R05 | Integration Hub implements provider-neutral connection metadata with owner filtering and secure credential references. The current connector is OpenRouter, not Google OAuth. Provision/status/delete credential operations exist; a production Google authorized-read/refresh seam needs development and validation. | scenarios/integration-hub/README.md; api/hub.go; api/credential_cli.go; docs/configuration/integrations/connections.md |
| R06 | Personal Planner has a read-only Google Calendar HTTP adapter with pagination and sync-token reset signalling. Its UI/service sync path uses fixture data. No live OAuth or durable cursor wiring is claimed. The adapter currently extracts a narrow event representation; recurrence/cancellation and full searchable fields need qualification. | scenarios/personal-planner/docs/concepts/INTEGRATIONS.md; api/internal/integrations/google.go, google_test.go, sqlite.go |
| R07 | Shared search has SourceDoc/Chunk/EmbeddingTextComposer seams, PagedSource and ChangeSetSource, incremental/shadow generation mechanics, lexical+semantic fusion and rank evidence. Providers own selection and domain policy. | packages/ai-go/search/README.md; contracts.go; streaming.go |
| R08 | The shared Markdown chunker was extracted from Knowledge Observatory. It is heading-aware, preserves table/fence atomic units and uses approximate 900-token target/135-token overlap defaults. A large atomic block may exceed the nominal hard maximum; embedding adapters need a real bound. | packages/ai-go/search/markdown.go; scenarios/knowledge-observatory/api/internal/aisearch/indexer.go |
| R09 | ContextualComposer prepends title, description and heading path. Architecture Cartographer has its own corpus composer for identity/responsibility/purpose/glossary. These demonstrate index-time composition, not a general generated-question multi-view pipeline. | packages/ai-go/search/markdown.go; scenarios/architecture-cartographer/api/internal/aisearch/domain_index.go |
| R10 | Providers declare question families/golden eval cases, and the Answer space enumerates question demand. The investigation did not establish that all anticipated-question variations are currently embedded as independent views. Treat that as a research/implementation question rather than deny the operator's experience or claim a missing universal feature. | packages/ai-go/search/searchjson.go; docs/reference/search-json.md; scenarios/search-hub/docs/spaces/answer-space.md |
| R11 | Document Manager owns source bytes, derivations, anchors, enrichment and retrieval. Its format registry includes local parsing for common documents/tables, positioned PDF handling, local OCR and optional vision. Registry declarations are not proof every runtime chain or every Google-native export has passed. | scenarios/document-manager/docs/concepts/DOMAINS.md; docs/reference/format-matrix.md; api/internal/derivation/registry.json; api/internal/intake/types.go; api/internal/retrieval/types.go |
| R12 | Document Manager's current retention/source identity does not automatically fit non-archival mutable Drive collections. Existing retrieval code and registration scaffolding require actual end-to-end qualification. Preserve its SQLite store choice; do not force Qdrant because another provider uses it. | scenarios/document-manager/docs/concepts/DATA.md; api/internal/intake/types.go; api/internal/retrieval/registration.go; PRD.md OT-P0-023 |
| R13 | Agent Inbox is AI conversation management; Signal Inbox is immutable selected-material capture. Neither is a mailbox-owning scenario. A new email scenario needs its product contract and ecosystem-fit, not an opportunistic rename of either. | scenarios/agent-inbox/PRD.md; scenarios/signal-inbox/PRD.md |
| R14 | Finite delivery uses Prompt Manager team configuration and Agent Manager orchestration. The native heartbeat-bind-effort path provisions a new leader disabled and separates activation. Existing delivery profiles are registered. | prompt-manager skill read large-effort-orchestration; scenarios/prompt-manager/cli/teams/finite_leader.go; api/internal/teamconfig/finite_leader.go; .vrooli/agent-manager/delivery-*.json |
| R15 | At preparation, no existing team owned external knowledge delivery. personal-planner-delivery was enabled and owned effort:personal-planner-I18. Calendar work must be coordinated, not assigned over its active owner. | Native team list/show and Agent Manager effort list, observed 2026-10-14. |

Relative evidence beneath a scenario refers to that scenario's directory. Source material is linked, not copied into a competing product contract. No user mailbox content or Drive document content was ingested in this investigation.

## Official external sources

| ID | Source URL | Finding |
|---|---|---|
| W01 | https://developers.google.com/identity/protocols/oauth2/web-server | Offline authorization yields refresh capability for absent-user operation. OAuth consent is separate from a ChatGPT connector. |
| W02 | https://developers.google.com/identity/protocols/oauth2/ | External OAuth projects left in Testing usually issue seven-day refresh tokens outside basic identity-only scopes. Revocation/password/account policy can invalidate grants. |
| W03 | https://developers.google.com/workspace/drive/api/guides/api-specific-auth | drive.file is per-file access; drive.readonly and metadata-wide scopes are restricted. Folder selection policy is distinct from Google-granted access. Qualify future descendant discovery rather than assume Picker grants whole trees. |
| W04 | https://developers.google.com/workspace/drive/api/guides/search-files | Parent queries return direct children; recursive traversal needs client-side enumeration. |
| W05 | https://developers.google.com/workspace/drive/api/guides/manage-changes | Drive has starting and continuation change tokens; notifications do not carry full changes. |
| W06 | https://developers.google.com/workspace/drive/api/guides/push | Notification channel lifecycle/expiry must be managed. Push triggers a feed read; it does not replace sync checkpoints. |
| W07 | https://developers.google.com/workspace/gmail/api/guides/sync | Full sync plus history-based partial sync; unavailable history can produce 404 and require a full sync. |
| W08 | https://developers.google.com/workspace/gmail/api/guides/push | Gmail watches require renewal at least every seven days; Google recommends daily renewal. Pub/Sub notifications may be delayed/dropped; fallback history reads are required. |
| W09 | https://developers.google.com/workspace/calendar/api/guides/sync | Initial and incremental per-calendar synchronization; 410 invalid tokens require resync and query parameters must remain compatible. |
| W10 | https://developers.google.com/workspace/calendar/api/guides/push | Calendar notification channels expire and require replacement; notifications contain no changed event body. |
| W11 | https://developers.google.com/identity/protocols/oauth2/production-readiness/restricted-scope-verification | Personal/internal use can qualify for verification exceptions. A broader distributed service has different requirements, including possible security assessment. Check the actual deployment before prescribing verification. |
| W12 | https://developers.google.com/workspace/workspace-api-user-data-developer-policy | Govern retention, disclosure and inference use of Workspace data; personalized retrieval is not permission to train shared models on private content. Recheck policies for a distributed product. |
| W13 | https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt | Supported ChatGPT configurations can use remote MCP custom apps. This does not establish the actual always-on Dots environment's tool access. |

These sources were browsed on 2026-10-14. Revalidate relevant online APIs/scopes/policies at admission and record changed facts. No claim about a newly released ChatGPT feature is inferred from a general custom-app article.

## Decisions and hypotheses

- Accepted: Drive-selected-folder first priority; domain-owned projections; integration-hub connection lifecycle; document-manager remote-backed extension; identity-aware federation; later Calendar/Gmail in the same finite effort; team disabled until operator activation.
- Proposed defaults: about five-minute incremental Drive cadence, normalized text/passage retention without originals, bounded generated question views after baseline A/B, and Gmail as first provider of a provider-neutral email scenario.
- Unselected: exact folders, grants, file-size/language/format policy, vision/multimodal scope, email name/UX/write behavior, OAuth client type/callback, cost limits and numerical search-quality thresholds.
- Open technical finding: the email/Calendar source checkpoints and serving index checkpoints must be decoupled without losing replay safety. Shared ChangeSetSource's promotion checkpoint is an indexing contract, not permission to discard captured provider changes after an embedding outage.

## Preparation limits

No Google Cloud project, OAuth client, credentials, source selection, live sync, delivery run, product implementation or UI selection was created by this research. Native commissioning receipts are in HANDOFF-RECEIPT.md. A disabled prepared team is not an enabled qualified commission.
