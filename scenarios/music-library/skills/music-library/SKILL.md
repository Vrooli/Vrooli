---
name: "music-library"
description: "Play and organize my music and learn my taste: save a track with my listening judgment, find tracks by tag, project or accepted use, record where a track was used, and keep notes."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["music-library", "music", "player", "taste", "playlist", "tracks", "listening", "catalog"]
  requires:
    scenarios: ["music-library"]
    commands: ["music-library catalog put"]
  origin:
    kind: "authored"
---

## music-library

A personal catalog of tracks with explicit listener judgments.

| You want | Command |
|---|---|
| Save a track and my judgment | `music-library catalog put` |
| Find tracks | `music-library catalog list` |
| Record a use in a project | `music-library catalog use` |
| Notes | `music-library notes create`, `music-library notes list` |

Run `music-library <group> <command> --help` for flags.

For generating new music, read the `music-tools` skill instead.
