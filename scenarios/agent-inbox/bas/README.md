# BAS Automation

Store automation workflows here. Keep it short:

- `cases/<operational-target>/<surface>/` mirrors operational targets (rename folders as needed).
- `flows/` contains multi-surface user journeys.
- `actions/` hosts fixtures referenced via `@fixture/<slug>`.
- `seeds/` includes optional setup scripts when deterministic data is required.

## Directory Structure

```
bas/
├── cases/
│   ├── 01-inbox-list/
│   │   └── ui/
│   │       ├── chat-list-display.json      # [REQ:INBOX-LIST-001]
│   │       └── read-unread-toggle.json     # [REQ:INBOX-LIST-002,INBOX-LIST-003]
│   └── 02-chat-creation/
│       └── ui/
│           └── create-new-chat.json        # [REQ:BUBBLE-001]
├── flows/
│   └── basic-chat-flow.json                # Multi-requirement E2E flow
├── seeds/
├── actions/
└── registry.json
```

Each workflow JSON must include:

```json
{
  "metadata": {
    "description": "What the workflow validates",
    "requirement": "REQ-ID",
    "version": 1
  }
}
```

Reference selectors via `@selector/<key>` from `ui/src/consts/selectors.ts`. After adding or moving a workflow, run from the scenario directory:

```bash
test-genie registry build
```

This regenerates `bas/registry.json`, which is tracked so other agents can see which files exist, which requirements they validate, and what fixtures they depend on.
