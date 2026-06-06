# Browser Extension Readiness

Requirement refs: `BIH-EXT-001`, `BIH-EXT-002`

Future browser-extension validation should confirm that the extension can submit saved-bookmark payloads to `POST /api/v1/bookmarks/process` and receive a processed-count response without bypassing profile isolation.
