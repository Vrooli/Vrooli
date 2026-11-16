# Topic templates

Start a topic with:

```sh
python3 -I scripts/new_topic.py --mode <mode> --title "<Title>" \
  --intent-file <words.md> <topic-name>    # every '>' quote, verbatim, with its date and source
```

(`--intent "<one quote>" --source "<chat, date>"` instead takes a single quote.)

It copies `common/` plus the mode's folder into `~/.vrooli/research/topics/<topic-name>/`,
fills the title, date, mode and the operator's words, and runs `mission_check.py`. The result
fails the check on purpose until you replace every `<fill: …>` marker: run the architect pass
and ambition review (`../review-briefs.md`) to write the end state, scorecard and
workstreams from the operator's intent and the code as it is.

What you get, and nothing more:

| File | Why it is there |
|---|---|
| `README.md` | The front page: what this is, where it stands, the next proof |
| `MISSION.md` | The charter, with the mode's approval flow and boundaries filled in |
| `TOOLS.md` | The services research leans on, each with its fallback; add the topic's own |
| `INBOX.md` | Where other levels leave messages for this topic's next agent |
| `work/README.md` | The plan (ready-to-run briefs) and the risk list |
| `knowledge/…` | The page this kind of topic learns into (see each mode) |

| Mode | Knowledge page | Extra |
|---|---|---|
| `build` | `knowledge/MAP.md`: architecture, current vs target | — |
| `propose-then-build` | `knowledge/LANDSCAPE.md`: what exists, what Vrooli lacks, prototypes | — |
| `gated-effect` | `knowledge/RUNBOOK.md`: the effect, its rehearsals and its exact steps | — |
| `artifact` | `knowledge/TASTE.md`: judged versions and what the operator liked | `ARTIFACTS.md` list file |
| `evidence` | `knowledge/CLAIMS.md`: each claim, its sources, its confidence | — |

`skills/`, `programs/` and `feedback/` are created when first needed; the
[topic folder](../../references/topic-folder.md) says how each works.
