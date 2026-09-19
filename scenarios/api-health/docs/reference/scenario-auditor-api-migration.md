# Scenario-Auditor API Migration

This ledger accounts for every production rule file under
`scenarios/scenario-auditor/api/rules/api/`. It is migration evidence only:
API Health does not call scenario-auditor at runtime.

## Decision Meanings

| Decision | Meaning |
|---|---|
| kept | API Health keeps the legacy rule intent with materially equivalent ownership and semantics. |
| redesigned | API Health preserves the useful readiness intent but replaces brittle implementation behavior. |
| delegated | Another health provider or generic quality provider owns the concern. |
| deferred | The concern is valid but intentionally not implemented yet. |
| rejected | The legacy concern is not a valid API Health readiness rule. |

## Migration Ledger

| Legacy file | Rule ID | Decision | Owner | API Health mapping | Rationale |
|---|---|---|---|---|---|
| `application_logging.go` | `application_logging` | redesigned | API Health runtime hygiene | `api.unstructured_logging` | Preserves operational logging intent for production API runtime paths without generic format-string policing. |
| `content_type_binary.go` | `content_type_binary` | redesigned | API Health HTTP semantics | `api.content_type_missing` | Folded into explicit writer evidence and route classification instead of independent regex rules. |
| `content_type_csv.go` | `content_type_csv` | redesigned | API Health HTTP semantics | `api.content_type_missing` | Kept only when a production API route clearly writes CSV/text content directly. |
| `content_type_headers.go` | `content_type_headers` | redesigned | API Health HTTP semantics | `api.content_type_missing` | Broad header checks become low-ambiguity content-type evidence for production REST handlers, excluding tests, Connect procedures, and framework-owned responses. |
| `content_type_html.go` | `content_type_html` | redesigned | API Health HTTP semantics | `api.content_type_missing` | Merged into the route-aware analyzer so static/template paths are not treated as feature API defects. |
| `content_type_pdf.go` | `content_type_pdf` | redesigned | API Health HTTP semantics | `api.content_type_missing` | Retained only for obvious direct writes where API Health can cite file and handler evidence. |
| `content_type_streaming.go` | `content_type_streaming` | redesigned | API Health HTTP semantics | `api.content_type_missing` | Represented by route classification and explicit content-type evidence rather than broad source scanning. |
| `content_type_text.go` | `content_type_text` | redesigned | API Health HTTP semantics | `api.content_type_missing` | Merged into the shared content-type analyzer with production API path filtering. |
| `content_type_xml.go` | `content_type_xml` | redesigned | API Health HTTP semantics | `api.content_type_missing` | Retained only where direct writer evidence avoids helper and test false positives. |
| `file_close.go` | `file_close` | delegated | Generic resource hygiene provider | n/a | Disk file-handle hygiene is not API readiness. API Health owns HTTP response body closure because that is API runtime behavior. |
| `goroutine_context.go` | `goroutine_context` | redesigned | API Health runtime hygiene | `api.goroutine_uncancellable` | Kept for API startup and handler paths, using AST evidence for long-lived loops without cancellation. |
| `health_check.go` | `health_check` | redesigned | API Health lifecycle and live health probe | `api.health_metadata_missing`, `api.health_probe_*` | Health readiness now combines service metadata validation, endpoint descriptors, and optional bounded live `/health` probing. |
| `http_client_timeout.go` | `http_client_timeout` | redesigned | API Health runtime hygiene | `api.http_client_unbounded` | Preserved for production API runtime paths and paired with request-context evidence. |
| `http_response_close.go` | `http_response_close` | redesigned | API Health runtime hygiene | `api.response_body_unclosed` | Kept because unclosed response bodies directly affect API runtime resource usage. |
| `http_status_codes.go` | `http_status_codes` | redesigned | API Health HTTP semantics | `api.raw_status_literal` | Raw numeric status checks are retained only for production API files with precise AST location evidence. |
| `preflight.go` | `preflight` | redesigned | API Health lifecycle | `api.preflight_missing`, `api.preflight_late`, `api.preflight_wrong_name` | Reimplemented with api/main.go structure and scenario-name checks instead of line-window heuristics. |
| `security_headers.go` | `security_headers` | delegated | security-health | n/a | CORS, HSTS, clickjacking, and browser security headers are security posture, not API readiness. |
| `server_run.go` | `server_run` | redesigned | API Health lifecycle | `api.direct_listen_and_serve`, `api.server_runner_missing` | Checks api-core server runner adoption and rejects direct ListenAndServe calls in production API entrypoints. |
| `versioned_endpoints.go` | `versioned_endpoints` | redesigned | API Health HTTP semantics | `api.rest_endpoint_unversioned` | Preserved for REST feature endpoints while exempting `/health`, Connect procedures, static assets, and declared `rest_exception` routes. |

## Parity Evidence

The unit test `api/internal/migration/ledger_test.go` walks the legacy
scenario-auditor API rule directory and fails if any production rule file is
missing from this ledger or appears more than once. Existing lifecycle, health
probe, HTTP semantics, runtime hygiene, and fix-registry tests provide the
representative parity fixtures for redesigned rules.

## Residual Cutover Notes

API Health intentionally does not implement security headers or broad disk file
handle checks. Those remain delegated until their owning providers are part of
the Test Genie standards phase. Test Genie cutover must preserve that boundary
instead of treating API Health as a generic scenario-auditor replacement.
