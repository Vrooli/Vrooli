# Classify Capture

## Purpose

Analyze a raw text capture and classify it into one or more suggested backlog items. If the text is unactionable (greetings, test messages, gibberish), produce an empty classification.

## Input Context

You are given raw text that a user typed stream-of-consciousness style. It may contain:
- One clear actionable item ("app-monitor has a tunnel reconnect bug")
- Multiple items in one thought ("fix the tunnel bug AND add backup crons to postgres")
- Vague or partial ideas that still have actionable intent
- Completely unactionable text (greetings, test words, gibberish)

**Capture ID:** {{CAPTURE_ID}}
**Capture text:** {{CAPTURE_TEXT}}

## Classification Rules

### Actionable Text
For each distinct actionable item in the text, determine:
- **kind**: One of `idea`, `research`, `fix`, `execute`, `chore`
  - `idea` — New feature, product concept, or scenario to build
  - `research` — Investigation, exploration, or learning task
  - `fix` — Bug fix, error correction, or broken behavior
  - `execute` — Clear implementation task with known scope
  - `chore` — Maintenance, cleanup, dependency updates, infrastructure
- **title** — Short, descriptive title (imperative form, e.g., "Fix tunnel reconnect in app-monitor")
- **description** — 1-3 sentence expansion of the title with context from the capture
- **priority** — 1-10 (1 = highest). Default to 5 unless urgency is clear from the text
- **tags** — 1-5 relevant tags extracted from context (scenario names, technologies, concepts)
- **confidence** — 0.0-1.0 how confident you are in this classification

### Unactionable Text (No-Op)
If the capture text is unactionable — greetings ("hi", "hello"), test messages ("test", "asdf"), single words with no context, gibberish, or anything that doesn't express intent to do work — produce an **empty items array**.

Examples of unactionable text: "hi", "test", "hello world", "asdf", "123", ".", "ok", "thanks"

### Edge Cases
- Very short but actionable: "fix login" → actionable (kind: fix, infer title)
- Ambiguous: "maybe we should look into caching" → actionable (kind: research, lower confidence)
- Mixed: "hey also can you fix the auth bug" → actionable (ignore the greeting, extract the fix)

## Output

Write a `classification.json` file to the capture folder and update `capture.json` status.

### Step 1: Write classification.json

Write the classification result directly to the capture folder. The agent's scope path is set to the capture directory, so write to `classification.json` in the current working directory:

```bash
cat > classification.json <<'CLASSIFICATION_EOF'
{
  "items": [
    {
      "kind": "<kind>",
      "title": "<title>",
      "description": "<description>",
      "priority": <1-10>,
      "tags": ["<tag1>", "<tag2>"],
      "confidence": <0.0-1.0>
    }
  ],
  "classified_at": "<current RFC3339 timestamp>"
}
CLASSIFICATION_EOF
```

For unactionable text, write an empty items array:
```bash
cat > classification.json <<'CLASSIFICATION_EOF'
{
  "items": [],
  "classified_at": "<current RFC3339 timestamp>"
}
CLASSIFICATION_EOF
```

### Step 2: Update capture status

Update the `status` field in `capture.json` from `"classifying"` to `"classified"`:

```bash
# Read current capture, update status, write back
jq '.status = "classified"' capture.json > capture.tmp.json && mv capture.tmp.json capture.json
```

## Constraints

- Do NOT create backlog items — only write the classification suggestion
- Do NOT modify any files outside the capture folder
- Keep the classification fast — this should complete in under 30 seconds
- Always produce valid JSON
- Always update the capture status to "classified" when done, even for empty classifications
