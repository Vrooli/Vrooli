# What the CLI returns

Generated, not written: `TestAgentResultsDocumentIsWhatTheCLIPrints` runs the real CLI against an
in-process Hub and fails when this file differs from what it printed. Ids,
times, hashes and paths are shown as placeholders. Refresh it with
`go test ./handlers/conversations -run TestAgentResultsDocument -update` in
`scenarios/notification-hub/api`.

Every `--json` command prints one JSON document on stdout (`ask --and-wait` prints two, the
send's and the wait's), with the Hub's field names in snake_case, on every exit status
(0, 3, 4 and 5). Guidance and the packet check
go to stderr (the writing-guide reminder, which depends on the shell, is left out of
the examples). A field with no value is left out.

## ask --json

`ask_id`, `state`, `outcome` and `warnings` are always present. `outcome` is `created`, `updated`, `replaced`, `reasserted`, `suppressed` or `returned`; `state` is the ask's state after the call. Each warning has `code`, `field`, `message`, `severity` (`advice`: the ask is complete but reads poorly; `dropped`: the Hub left that part out) and `carried_over`.

```text
$ notification-hub conversations ask --subject-key release-check --yes-no --question 'Ship the release today?' --big-picture 'The release is green and the window closes at noon.' --packet work/RELEASE-CHECK --attach plan.md --attach cards.png --attach table.png --form form.yaml --json
# exit status 0
# stdout
{
  "ask_id": "<ask-id>",
  "outcome": "created",
  "warnings": [
    {
      "code": "explainer_missing",
      "field": "attachments",
      "message": "no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls \"$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/\") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section \"Make the page\")",
      "severity": "advice",
      "carried_over": false
    }
  ],
  "state": "pending"
}
# stderr
Packet check: 1 advice. The ask was sent; nothing here blocked it.
  Advice:
    - attachments: no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls "$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section "Make the page") (explainer_missing)
```

`carried_over` is true for a finding the open ask already had, about a part the revision did not touch; the human output folds those into one line (`2 earlier warnings still apply`). Here the first send has no big picture, and the revision adds technical details only:

```text
$ notification-hub conversations ask --subject-key disk-low --yes-no --question 'Free space on the backup drive now?' --json
# exit status 0
# stdout
{
  "ask_id": "<ask-id>",
  "outcome": "created",
  "warnings": [
    {
      "code": "missing_big_picture",
      "field": "big_picture",
      "message": "there is no big picture; add the situation and what each answer causes, so the operator can decide without your context",
      "severity": "advice",
      "carried_over": false
    },
    {
      "code": "explainer_missing",
      "field": "attachments",
      "message": "no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls \"$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/\") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section \"Make the page\")",
      "severity": "advice",
      "carried_over": false
    },
    {
      "code": "packet_missing",
      "field": "packet",
      "message": "no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it",
      "severity": "advice",
      "carried_over": false
    }
  ],
  "state": "pending"
}
# stderr
Packet check: 3 advice. The ask was sent; nothing here blocked it.
  Advice:
    - big_picture: there is no big picture; add the situation and what each answer causes, so the operator can decide without your context (missing_big_picture)
    - attachments: no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls "$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section "Make the page") (explainer_missing)
    - packet: no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it (packet_missing)
```

```text
$ notification-hub conversations ask --subject-key disk-low --technical-details 'Drive at 91 percent.' --json
# exit status 0
# stdout
{
  "ask_id": "<ask-id>",
  "outcome": "updated",
  "warnings": [
    {
      "code": "missing_big_picture",
      "field": "big_picture",
      "message": "there is no big picture; add the situation and what each answer causes, so the operator can decide without your context",
      "severity": "advice",
      "carried_over": true
    },
    {
      "code": "explainer_missing",
      "field": "attachments",
      "message": "no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls \"$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/\") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section \"Make the page\")",
      "severity": "advice",
      "carried_over": true
    },
    {
      "code": "packet_missing",
      "field": "packet",
      "message": "no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it",
      "severity": "advice",
      "carried_over": false
    }
  ],
  "state": "pending"
}
# stderr
Kept as the open ask has them: the question, the options, the recommendation, the big picture, the context link, the urgency, the sender, the attachments, the form.
Packet check: 1 advice. The ask was sent; nothing here blocked it.
  Advice:
    - packet: no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it (packet_missing)
  2 earlier warnings still apply (same as before; --json lists them).
```


## ask (text output)

The last line of stdout is `ask_id=... state=... outcome=...`, with `idempotency_key=...` when the CLI chose the key, and `next: ...` with the command to wait. The packet check and what a revision kept go to stderr.

```text
$ notification-hub conversations ask --subject-key retry-window --yes-no --question 'Retry tonight'\''s backup to the Elements drive now?'
# exit status 0
# stdout
Created ask <ask-id>.
ask_id=<ask-id> state=pending outcome=created idempotency_key=ask-<key> next: notification-hub conversations wait <ask-id>
# stderr
Packet check: 3 advice. The ask was sent; nothing here blocked it.
  Advice:
    - big_picture: there is no big picture; add the situation and what each answer causes, so the operator can decide without your context (missing_big_picture)
    - attachments: no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls "$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section "Make the page") (explainer_missing)
    - packet: no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it (packet_missing)
```

```text
$ notification-hub conversations ask --subject-key retry-window --technical-details 'Run 42; the drive was remounted at 21:10.'
# exit status 0
# stdout
Updated open ask <ask-id> in place; no new notification was sent.
ask_id=<ask-id> state=pending outcome=updated idempotency_key=ask-<key> next: notification-hub conversations wait <ask-id>
# stderr
Kept as the open ask has them: the question, the options, the recommendation, the big picture, the context link, the urgency, the sender, the attachments, the form.
Packet check: 1 advice. The ask was sent; nothing here blocked it.
  Advice:
    - packet: no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it (packet_missing)
  2 earlier warnings still apply (same as before; --json lists them).
```


## ask --dry-run

Exit status 0 when the request is valid. Nothing is stored and no ask is created; stdout is the notification the operator would get, stderr the packet check. A keyed request is checked as the revision merged over the open ask (`Would revise open ask ...`); otherwise it says it would create one. `--json` prints `{revises_ask_id, push_title, push_body, warnings[]}`, a field with no value left out.

```text
$ notification-hub conversations ask --subject-key release-check --technical-details 'Run 43.' --dry-run
# exit status 0
# stdout
Summary:
  Dry run: nothing was stored and no ask was created.
  Would revise open ask <ask-id> in place; no new notification would be sent.
  Sender the operator would see: Agent run (kept from the open ask)
Notification the operator would get:
  Title: Decision required
  Body:
      Ship the release today?
Retrieval Hints:
  To send it, run the same command without --dry-run.
# stderr
Packet check: no new findings. Nothing was sent (dry run); the ask would go out as it is, so fix these first if they matter.
  1 earlier warning still applies (same as before; --json lists them).
```

```text
$ notification-hub conversations ask --subject-key gate-check --yes-no --question 'Open the gate to the staging cluster?' --dry-run --json
# exit status 0
# stdout
{
  "warnings": [
    {
      "code": "missing_big_picture",
      "field": "big_picture",
      "message": "there is no big picture; add the situation and what each answer causes, so the operator can decide without your context",
      "severity": "advice"
    },
    {
      "code": "explainer_missing",
      "field": "attachments",
      "message": "no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls \"$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/\") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section \"Make the page\")",
      "severity": "advice"
    },
    {
      "code": "packet_missing",
      "field": "packet",
      "message": "no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it",
      "severity": "advice"
    }
  ],
  "push_title": "Decision required",
  "push_body": "Open the gate to the staging cluster?",
  "sender": "Agent run",
  "sender_source": "agent_run"
}
```


## wait --json

Exit status 0, 3, 4 and 5 all print the same shape: `ask_id` and `state`, plus what applies. The last line of the human output is `ask_id=... state=...`.


### Exit 3: the operator asked for more context

The ask is still open (`state` is `pending`). Their question is `context_request.question`. Answer it with `ask --subject-key K --context-response '...'`, then wait again.

```text
$ notification-hub conversations wait --subject-key release-check --json
# exit status 3
# stdout
{
  "ask_id": "<ask-id>",
  "state": "pending",
  "reason": "the operator asked for more context; answer it with context_response",
  "context_request": {
    "id": "<ask-id>",
    "question": "Is the storage migration finished?",
    "requested_by": "alice",
    "requested_at": "<time>"
  },
  "packet": "<home>/work/RELEASE-CHECK/BRIEF.md"
}
# stderr
The operator asked for more context before answering ask <ask-id>:
  Is the storage migration finished?
Answer it with --context-response; the operator sees your answer under their question, and the ask is otherwise unchanged:
  notification-hub conversations ask --subject-key release-check --context-response "<your answer to their question>"
  If the answer also changes the ask, send that in the same command (a revised --question, --big-picture, --attach, --remove-attachment, --form or --remove-question); the ask keeps everything else.
Then wait again: notification-hub conversations wait --subject-key release-check
Error: ask <ask-id> is still open: the operator asked for more context (exit status 3); answer it with `ask --subject-key K --context-response '...'`, then wait again
```

The answer is sent with `ask --subject-key K --context-response '...'`; the command's own result is the `ask` document, with `outcome` `updated`:

```text
$ notification-hub conversations ask --subject-key release-check --context-response 'Yes. The storage migration finished at 10:42 and the checksums match.' --json
# exit status 0
# stdout
{
  "ask_id": "<ask-id>",
  "outcome": "updated",
  "warnings": [
    {
      "code": "explainer_missing",
      "field": "attachments",
      "message": "no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls \"$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/\") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section \"Make the page\")",
      "severity": "advice",
      "carried_over": true
    }
  ],
  "state": "pending"
}
# stderr
Kept as the open ask has them: the question, the options, the recommendation, the big picture, the technical details, the context link, the urgency, the sender, the attachments, the form.
Packet check: no new findings. The ask was sent; nothing here blocked it.
  1 earlier warning still applies (same as before; --json lists them).
```


### Exit 4: your wait timed out

`--timeout` ran out. Nothing happened to the ask: it is still open and expires on its own schedule (`--expires`). Wait again, or carry on and read it later with `get`.

```text
$ notification-hub conversations wait --subject-key release-check --timeout 1s --json
# exit status 4
# stdout
{
  "ask_id": "<ask-id>",
  "state": "pending",
  "reason": "the wait timed out; the ask is still open",
  "packet": "<home>/work/RELEASE-CHECK/BRIEF.md"
}
# stderr
Error: your wait timed out (exit status 4): the operator has not answered yet, and ask <ask-id> is still open; the ask has not expired. Wait again: notification-hub conversations wait --subject-key release-check
```


### Exit 0: answered

`answer` is the option key the operator chose (or the default that applied), `note` their free text. `answer_summary` is the whole answer as plain text. `responses[]` holds one entry per form question the operator answered, found by `question_id`: `selected` (a list of choice keys) for `single_choice`, `multi_choice` and `variant`; `text` for `text`; `sections` (a list of `{section_key, verdict, note}`, the verdict `approve`, `change` or `drop`) for `section_review`. `attachment_notes[]` are notes the operator pinned to files. The main answer is `answer`, not a `responses[]` entry.

```text
$ notification-hub conversations wait --subject-key release-check --json
# exit status 0
# stdout
{
  "ask_id": "<ask-id>",
  "state": "answered",
  "answer": "yes",
  "reason": "answered by the operator",
  "note": "Go ahead.",
  "responses": [
    {
      "question_id": "rollout",
      "selected": [
        "staged"
      ]
    },
    {
      "question_id": "layout",
      "selected": [
        "cards"
      ]
    },
    {
      "question_id": "plan",
      "sections": [
        {
          "section_key": "storage",
          "verdict": "approve"
        },
        {
          "section_key": "retention",
          "verdict": "change",
          "note": "Keep 30 days."
        }
      ]
    },
    {
      "question_id": "why",
      "text": "Watch the disk graph for an hour."
    }
  ],
  "answer_summary": "Answer: Yes (yes)\nNote: Go ahead.\nWhich rollout do you prefer: Staged (staged)\nWhich layout should the page use: Card stack (cards)\nReview each section of the plan:\n  - Retention: change — Keep 30 days.\n  - Other sections: approve\nAnything else we should know: Watch the disk graph for an hour.\nNotes on files:\n  - plan.md: The storage section is the one that worries me.",
  "attachment_notes": [
    {
      "attachment": "plan.md",
      "note": "The storage section is the one that worries me."
    }
  ],
  "packet": "<home>/work/RELEASE-CHECK/BRIEF.md"
}
```


## ask --and-wait

The command prints the send's result first and the wait's result after it, then exits with the status `wait` would (0, 3, 4 or 5). In text mode the output has two `ask_id=` lines and the last line is the wait's (`ask_id=... state=... answer=...`); read the id from the first. With `--json` stdout holds two documents, one after the other: the `ask` document, then the `wait` document. A send that fails exits 1 and prints only its error; a wait that fails after a good send exits 1 as well, after the `ask` document. The ask's id is in the first document (`ask_id`); wait again later by `--subject-key`.

```text
$ notification-hub conversations ask --subject-key lunch-order --yes-no --question 'Order lunch for the team today?' --big-picture 'The team is in the office all day and nobody has planned lunch.' --and-wait --timeout 1m
# exit status 0
# stdout
Created ask <ask-id>.
ask_id=<ask-id> state=pending outcome=created idempotency_key=ask-<key>
Summary:
  Ask <ask-id>: answered
Answer:
  Answer: Yes (yes)
  Note: Pizza.
Retrieval Hints:
  Read the whole ask: notification-hub conversations get <ask-id>
ask_id=<ask-id> state=answered answer=yes
# stderr
Packet check: 2 advice. The ask was sent; nothing here blocked it.
  Advice:
    - attachments: no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls "$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section "Make the page") (explainer_missing)
    - packet: no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it (packet_missing)
```

```text
$ notification-hub conversations ask --subject-key dessert-order --yes-no --question 'Order dessert too?' --big-picture 'Lunch is ordered and there is budget left for dessert.' --and-wait --timeout 1m --json
# exit status 0
# stdout
{
  "ask_id": "<ask-id>",
  "outcome": "created",
  "warnings": [
    {
      "code": "explainer_missing",
      "field": "attachments",
      "message": "no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls \"$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/\") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section \"Make the page\")",
      "severity": "advice",
      "carried_over": false
    },
    {
      "code": "packet_missing",
      "field": "packet",
      "message": "no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it",
      "severity": "advice",
      "carried_over": false
    }
  ],
  "state": "pending"
}
{
  "ask_id": "<ask-id>",
  "state": "answered",
  "answer": "no",
  "reason": "answered by the operator",
  "answer_summary": "Answer: No (no)"
}
# stderr
Packet check: 2 advice. The ask was sent; nothing here blocked it.
  Advice:
    - attachments: no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. Make a real one from a starter (ls "$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/") that shows the situation and what happens on approval, and send it with --explainer page.html; it replaces the Hub's page (skill section "Make the page") (explainer_missing)
    - packet: no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it (packet_missing)
```


### Exit 5: closed without an answer

`state` is `withdrawn`, `dismissed` or `expired`; `reason` says why (the asker's reason, the operator's note, or that the ask expired). By ask id a revision with new options also ends here, as `withdrawn` with the reason `replaced`; by `--subject-key` the wait follows the replacement instead.

```text
$ notification-hub conversations wait --subject-key old-volume --json
# exit status 5
# stdout
{
  "ask_id": "<ask-id>",
  "state": "withdrawn",
  "reason": "The release decision made this moot."
}
# stderr
Error: ask <ask-id> was withdrawn without an answer (exit status 5): The release decision made this moot.
```


## withdraw

`--reason` is required. Exit status 0 when the ask was closed; the last line of stdout says `Withdrew ask ...`. A run waiting on the ask ends with status 5 and sees the reason. `--json` prints the ask as a document.

```text
$ notification-hub conversations withdraw --subject-key gate-check-old --reason 'The gate was opened by hand.'
# exit status 0
# stdout
Result:
  Withdrew ask <ask-id>.
What Changed:
  The ask is closed with your reason, and its record is kept. A run waiting on it ends with status 5 and sees the reason.
Next Command:
  notification-hub conversations get <ask-id>
```

```text
$ notification-hub conversations withdraw --subject-key gate-check-json --reason 'The gate was opened by hand.' --json
# exit status 0
# stdout
{
  "ask_id": "<ask-id>",
  "state": "withdrawn"
}
```

Without `--reason` the command fails before it reaches the Hub:

```text
$ notification-hub conversations withdraw --subject-key release-check
# exit status 1
# stderr
Error: missing required flag --reason
Usage: notification-hub conversations withdraw [id] [options]
```


## get --json

`{ask: {...}}`: the whole ask, with `options`, `attachments`, `questions`, `responses`, `context_requests` (oldest first; each has the operator's `question` and, once you answered it, your `response` and `addressed_at`) and the rest. A count with no value is left out, so read a missing `open_context_request_count` as 0.

```text
$ notification-hub conversations get --subject-key old-volume --json
# exit status 0
# stdout
{
  "ask": {
    "id": "<ask-id>",
    "notification_id": "<ask-id>",
    "question": "Roll back to the old volume?",
    "options": [
      {
        "key": "yes",
        "label": "Yes"
      },
      {
        "key": "no",
        "label": "No"
      },
      {
        "key": "other",
        "label": "Other — I have a different answer"
      }
    ],
    "urgency": "normal",
    "deadline": "<time>",
    "state": "withdrawn",
    "reason": "The release decision made this moot.",
    "first_delivered_at": "<time>",
    "source": "alice",
    "requester_ref": "ask-<key>",
    "resolved_at": "<time>",
    "created_at": "<time>",
    "updated_at": "<time>",
    "big_picture": "The new volume is healthy; the old one is still mounted.",
    "title": "Decision required",
    "answered_by_identity_kind": "unknown",
    "subject_key": "old-volume",
    "last_asserted_at": "<time>",
    "attachments": [
      {
        "name": "decision-summary.html",
        "label": "The Hub's page, made from the question and options",
        "kind": "KIND_FILE",
        "role": "ROLE_EXPLAINER",
        "sha256": "<sha256>",
        "size_bytes": "1490",
        "media_type": "text/plain; charset=utf-8",
        "preview_kind": "code",
        "source_status": "unknown"
      }
    ],
    "sender": "Agent run"
  }
}
```

An answered ask with a form, files, an operator's question and the answer to it. `responses[]` has one entry per form question the operator answered (`question_id` and the shape of its type); `attachment_notes[]` are their notes on files; `context_requests[]` has each question with your `response`:

```text
$ notification-hub conversations get --subject-key release-check --json
# exit status 0
# stdout
{
  "ask": {
    "id": "<ask-id>",
    "notification_id": "<ask-id>",
    "question": "Ship the release today?",
    "options": [
      {
        "key": "yes",
        "label": "Yes"
      },
      {
        "key": "no",
        "label": "No"
      },
      {
        "key": "other",
        "label": "Other — I have a different answer"
      }
    ],
    "urgency": "normal",
    "deadline": "<time>",
    "state": "answered",
    "reason": "answered by the operator",
    "answer": "yes",
    "answer_label": "Yes",
    "note": "Go ahead.",
    "answered_by": "alice",
    "answered_at": "<time>",
    "first_delivered_at": "<time>",
    "source": "alice",
    "requester_ref": "ask-<key>",
    "resolved_at": "<time>",
    "created_at": "<time>",
    "updated_at": "<time>",
    "big_picture": "The release is green and the window closes at noon.",
    "title": "Decision required",
    "answered_by_identity_kind": "unknown",
    "subject_key": "release-check",
    "last_asserted_at": "<time>",
    "attachments": [
      {
        "name": "decision-summary.html",
        "label": "The Hub's page, made from the question and options",
        "kind": "KIND_FILE",
        "role": "ROLE_EXPLAINER",
        "sha256": "<sha256>",
        "size_bytes": "1742",
        "media_type": "text/plain; charset=utf-8",
        "preview_kind": "code",
        "source_status": "unknown"
      },
      {
        "name": "BRIEF.md",
        "label": "BRIEF.md",
        "kind": "KIND_FILE",
        "role": "ROLE_EVIDENCE",
        "source_path": "<path>",
        "source_host": "<host>",
        "sha256": "<sha256>",
        "size_bytes": "78",
        "media_type": "text/markdown; charset=utf-8",
        "preview_kind": "markdown",
        "source_status": "unknown"
      },
      {
        "name": "plan.md",
        "label": "plan.md",
        "kind": "KIND_FILE",
        "role": "ROLE_EVIDENCE",
        "source_path": "<path>",
        "source_host": "<host>",
        "sha256": "<sha256>",
        "size_bytes": "79",
        "media_type": "text/markdown; charset=utf-8",
        "preview_kind": "markdown",
        "source_status": "unknown"
      },
      {
        "name": "cards.png",
        "label": "cards.png",
        "kind": "KIND_FILE",
        "role": "ROLE_EVIDENCE",
        "source_path": "<path>",
        "source_host": "<host>",
        "sha256": "<sha256>",
        "size_bytes": "17",
        "media_type": "image/png",
        "preview_kind": "image",
        "source_status": "unknown"
      },
      {
        "name": "table.png",
        "label": "table.png",
        "kind": "KIND_FILE",
        "role": "ROLE_EVIDENCE",
        "source_path": "<path>",
        "source_host": "<host>",
        "sha256": "<sha256>",
        "size_bytes": "12",
        "media_type": "image/png",
        "preview_kind": "image",
        "source_status": "unknown"
      }
    ],
    "questions": [
      {
        "id": "rollout",
        "type": "single_choice",
        "prompt": "Which rollout do you prefer?",
        "choices": [
          {
            "key": "staged",
            "label": "Staged"
          },
          {
            "key": "all-at-once",
            "label": "All at once"
          }
        ]
      },
      {
        "id": "layout",
        "type": "variant",
        "prompt": "Which layout should the page use?",
        "choices": [
          {
            "key": "cards",
            "label": "Card stack",
            "attachment": "cards.png"
          },
          {
            "key": "table",
            "label": "Table",
            "attachment": "table.png"
          }
        ]
      },
      {
        "id": "plan",
        "type": "section_review",
        "prompt": "Review each section of the plan.",
        "sections": [
          {
            "key": "storage",
            "title": "Storage",
            "attachment": "plan.md",
            "anchor": "storage"
          },
          {
            "key": "retention",
            "title": "Retention",
            "attachment": "plan.md",
            "anchor": "retention"
          }
        ],
        "unanswered_verdict": "approve",
        "sections_from": "plan.md"
      },
      {
        "id": "why",
        "type": "text",
        "prompt": "Anything else we should know?"
      }
    ],
    "responses": [
      {
        "question_id": "rollout",
        "selected": [
          "staged"
        ]
      },
      {
        "question_id": "layout",
        "selected": [
          "cards"
        ]
      },
      {
        "question_id": "plan",
        "sections": [
          {
            "section_key": "storage",
            "verdict": "approve"
          },
          {
            "section_key": "retention",
            "verdict": "change",
            "note": "Keep 30 days."
          }
        ]
      },
      {
        "question_id": "why",
        "text": "Watch the disk graph for an hour."
      }
    ],
    "context_requests": [
      {
        "id": "<ask-id>",
        "question": "Is the storage migration finished?",
        "requested_by": "alice",
        "requested_at": "<time>",
        "addressed_at": "<time>",
        "response": "Yes. The storage migration finished at 10:42 and the checksums match."
      }
    ],
    "answer_summary": "Answer: Yes (yes)\nNote: Go ahead.\nWhich rollout do you prefer: Staged (staged)\nWhich layout should the page use: Card stack (cards)\nReview each section of the plan:\n  - Retention: change — Keep 30 days.\n  - Other sections: approve\nAnything else we should know: Watch the disk graph for an hour.\nNotes on files:\n  - plan.md: The storage section is the one that worries me.",
    "attachment_count": 4,
    "question_count": 4,
    "attachment_notes": [
      {
        "attachment": "plan.md",
        "note": "The storage section is the one that worries me."
      }
    ],
    "sender": "Agent run",
    "packet": "<home>/work/RELEASE-CHECK/BRIEF.md"
  }
}
```

