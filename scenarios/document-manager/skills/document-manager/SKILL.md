---
name: "document-manager"
description: "Ingest and search documents: ingest a pdf or other file, parse it into citable passages, organize documents into collections, summarize and classify them, and run a keyword plus semantic query to find or cite a passage."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["document-manager", "documents", "pdf", "ingest", "parse", "cite", "citation", "search", "collections", "summarize", "retrieval"]
  requires:
    scenarios: ["document-manager"]
    commands: ["document-manager intake ingest"]
  origin:
    kind: "authored"
---

## document-manager

Local-first: parsing, anchors and sensitivity-aware routing happen on this machine.

| You want | Command |
|---|---|
| Ingest a file | `document-manager intake ingest`, `document-manager intake watch` |
| Find what is in a document set | `document-manager retrieval query`, `document-manager retrieval similar` |
| Organize | `document-manager corpus create`, `document-manager corpus add-document`, `document-manager corpus list-documents` |
| Summarize or classify | `document-manager enrichment enrich` |
| Export or import a collection | `document-manager corpus export`, `document-manager corpus import` |

Run `document-manager <group> <command> --help` for flags.
