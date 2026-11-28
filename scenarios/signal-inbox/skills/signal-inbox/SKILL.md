---
name: "signal-inbox"
description: "Save and find things I come across: capture a bookmark, article, video, link, pasted text or image to read later, categorize and triage them, search everything captured, and see what is still unresolved."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["signal-inbox", "bookmark", "read-later", "save-link", "article", "capture", "inbox", "triage", "categories", "search"]
  requires:
    scenarios: ["signal-inbox"]
    commands: ["signal-inbox signals capture"]
  origin:
    kind: "authored"
---

## signal-inbox

An immutable corpus of captured signals; triage changes state, never deletes.

| You want | Command |
|---|---|
| Save something | `signal-inbox signals capture` |
| Find something I saved | `signal-inbox retrieval search`, `signal-inbox signals list` |
| What is still unresolved | `signal-inbox retrieval ambient` |
| Categorize | `signal-inbox categories list`, `signal-inbox categories confirm`, `signal-inbox categories create` |
| Mark done, annotate | `signal-inbox triage set-disposition`, `signal-inbox triage annotate` |
| Import an archive | `signal-inbox sources import` |

Run `signal-inbox <group> <command> --help` for flags.
