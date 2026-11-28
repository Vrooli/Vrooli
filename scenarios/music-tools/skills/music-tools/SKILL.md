---
name: "music-tools"
description: "Generate, compose and arrange music: define a composition style, queue a batch of takes from a music model, wait for the jobs, list and reserve the takes you like, and arrange sample-exact audio from retained sources. For choosing a model for a brief, also read the music-generation skill."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["music-tools", "music", "compose", "composition", "generate-music", "song", "track", "audio", "takes", "style", "arrange"]
  requires:
    scenarios: ["music-tools"]
    commands: ["music-tools compose run", "music-tools jobs wait"]
  origin:
    kind: "authored"
---

## Music Tools

| You want | Command |
|---|---|
| See available music models | `music-tools models list` (`models get <id>`) |
| Define or reuse a style | `music-tools styles list`, `styles create`, `styles compile`, `styles import` / `export` |
| Make music | `music-tools compose run` (queues a batch of takes) |
| Follow the job | `music-tools jobs list`, `jobs get`, `jobs wait`, `jobs cancel` |
| Review results | `music-tools takes list`, `takes get` |
| Keep a take | `music-tools takes reserve` (`pool draw` reserves composition takes) |
| Arrange retained sources into one file | `music-tools delivery arrange` |

Run `music-tools <group> <command> --help` for the typed fields. Choosing which
model suits a brief is a separate judgment: read the `music-generation` skill
first, then queue the batch here.
