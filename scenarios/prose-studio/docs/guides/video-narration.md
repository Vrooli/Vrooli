# Draft narration from an ordered beat outline

Prose Studio composes one spoken passage per beat, carries prior passages into
the next beat, and assembles them in order. You own the beat plan, the timed
read, the final script, and the film edit. The draft and its beat summaries
are stored in Prose Studio. The bundled profile uses the gateway's local-only
`write.default` route.

## Plan and draft

Choose beats from your own outline. In each summary specify what changes, which
facts or details the beat may use, and anything that must be saved for later.
Allocate words by timing a read at your intended delivery pace and leaving
room for pauses and visual beats. The profile supports up to eight beats and
a combined target of 900 words. Each passage has its own word band of roughly
±20% around its target, with a 512-token output cap. For a four-beat short
aiming for about 175 words and 70–80 seconds, start with roughly 40–50 words
per beat. The local smoke run targeted 42 words per beat and returned 176
words, about 73 seconds at 145 words per minute before pauses. Read the
assembled text aloud against the cut to confirm duration.

Register the scenario's declarations against the updated API:

```bash
prose-studio prose declarations-reindex --request-json '{}' --json
prose-studio prose profile-resolve \
  --request-json '{"key":"prose-studio/video-narration-local"}' --json
```

Save a request like this as `narration-request.json`. The example uses fictional
placeholder beats. Use a fresh document ID for each draft and keep the ID for
readback or recovery.

```json
{
  "document": {
    "id": "narration-draft-01",
    "title": "Working title",
    "profile_key": "prose-studio/video-narration-local",
    "outline": [
      {"intent":"Hook","summary":"Open on the empty hook where a key should be. Establish what the narrator needs to open.","target_words":42},
      {"intent":"Search","summary":"Follow the narrator back through the day's choices. Preserve the key's location for the reveal.","target_words":42},
      {"intent":"Turn","summary":"The narrator notices the letter has moved. Let the change trigger a final search.","target_words":42},
      {"intent":"Payoff","summary":"Reveal the key under the letter and close on the narrator's decision to open the door.","target_words":42}
    ]
  }
}
```

```bash
prose-studio prose document-create --request-json "$(cat narration-request.json)" --json
prose-studio prose document-show --id narration-draft-01 --text-only
```

For API clients, POST the same JSON to
`/vrooli.prose_studio.v1.prose.ProseStudioService/CreateDocument` using Connect
JSON. Resolve the scenario port with `vrooli scenario port prose-studio API_PORT`
and use the normal scenario authentication. Send only the input fields above.
An empty `outline_id` in the response identifies a caller-supplied plan.
Read `document.assembled_text` and check that `document.status` is
`assembled`. Review continuity, facts, tone, and timing in the edit.

The first film-brief draft shifted the point of view from the human
experimenter to the chicken. Naming the human narrator in every beat fixed
the perspective in a second run. The `coherent` verdict reflects configured
repetition and style-drift checks. It does not validate narrator identity or
the order of facts and actions. Specify both in the beats and verify them in
the draft.

## Recover a partial draft

Create, resume, and assemble each claim the document while writing. An
overlapping write for that ID receives `document_busy` (Connect `aborted`,
HTTP 409). Reads and writes for other IDs remain available. If create returns
an error or the client times out, first check the document by its ID.
Once the prior request has ended, resume a stored partial draft. The service
starts from its stored beat plan and skips committed passages. A duplicate
create for a caller-planned ID returns `document_already_exists` after the
active write has finished.

```bash
prose-studio prose document-show --id narration-draft-01 --json
prose-studio prose document-resume --request-json '{"id":"narration-draft-01"}' --json
prose-studio prose document-show --id narration-draft-01 --text-only
```

The corresponding Connect methods are `GetDocument` and `ResumeDocument`,
both with `{"id":"narration-draft-01"}`. If the API rejected the request before
creating the document, correct the request and create it again.

For educational Shorts, use the [media production brief compiler](../../../../tools/media-production/README.md) to carry the learning claim, on-screen evidence, prediction timing, and ending choice into the existing ordered beat contract.
