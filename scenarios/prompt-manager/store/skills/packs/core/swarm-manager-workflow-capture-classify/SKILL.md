# Capture Classification Workflow

Classify one raw operator capture into zero or more suggested backlog items. Read the capture text. Identify each distinct actionable work intent. Return only the typed result.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| The text contains one or more actionable work intents. | `suggested` — one item per distinct intent. |
| The text is readable but contains no actionable intent: a greeting, a test string, recognizable gibberish, or a bare word with no work implied. | `discarded` |
| You literally cannot decode the text: encoding damage, truncation, or non-language content. Readable text always resolves to `suggested` or `discarded`. | `abstained` |

Edge rules: short but actionable text ("fix login") is `suggested`. Vague but intentful text ("maybe look into caching") is `suggested` as `research` with low confidence. Mixed text: extract the actionable parts and ignore the filler.

## Item field rules

- `kind`: `idea` = new feature or concept to build; `research` = investigation or learning task; `fix` = broken behavior to correct; `execute` = implementation task with known scope; `chore` = maintenance, cleanup, or infrastructure.
- `priority`: 1 is most urgent. Use 3 when the text gives no urgency signal.
- `title`: imperative form ("Fix tunnel reconnect in app-monitor").
- `tags`: scenario names, technologies, and concepts taken from the text.
- `confidence`: your certainty that this item reflects the operator's intent.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.capture}}` | The capture record: id, text, attachments, version. The text is the sole classification source. |

## Boundary

This run is read-only. Do not write files. Do not create backlog items. Swarm applies your classification. `discarded` and `abstained` both leave the capture without suggestions; the distinction exists for the operator's audit trail — keep it honest.

Capture:
{{.capture}}
