# Data — Client Workflow Rehearsal

## Owned records

| Record | Owner | Retention and privacy |
|---|---|---|
| Engagement | `rehearsals` | Local scenario database; synthetic title and metadata only in the first slice. |
| Workflow revision | `rehearsals` | Store sanitized JSON, source format, source hash, and import time. Never persist submitted credential values. |
| Credential binding | `rehearsals` | Store placeholder identity and synthetic stand-in kind; never a real secret or reusable credential. |
| Simulator profile | simulator collaborator | Store a profile identifier and version reference; fixture payloads are made up. |
| Run and trace event | `rehearsals` | Append-only ordered evidence including operation, synthetic request/response summary, status, and timestamps. |
| Fault and repair evidence | `rehearsals` | Keep the injected fault, observed failure, repair action, and post-repair assertion together. |
| Review decision and handoff | `rehearsals` | Durable human disposition and a generated snapshot; never imply approval from execution success. |

SQLite is the initial owner store. Schemas live beside the domain repository and are registered through the API's existing schema registry. Imported workflow JSON is untrusted input: enforce a bounded size, reject invalid structure, remove secret-shaped values before persistence, and never interpret a workflow URL as permission to make a network request.

The first adapter bounds an n8n export to 512 KiB and stores only a structure fingerprint, not the uploaded document. This cap does not satisfy the charter's large-payload workflow requirement. Phase 4 must design streaming or externalized payload evidence with an explicit maximum request size, storage class, retention rule, cleanup behavior, and simulator body-size policy before increasing the cap.
