# Handlers

Thin HTTP façade for the Browser Automation Studio API. Handlers validate requests, invoke services, and stream execution updates via websockets.

```mermaid
flowchart LR
    subgraph Entry["Incoming Requests"]
        REST["REST/JSON\n/health, /api/v1/*"]
        WS["Websocket\n/ws?execution_id=<id>"]
    end

    subgraph Handlers["handlers/*"]
        H["handler.go\nwiring + routing"]
        AIH["ai/*\nDOM, screenshots,\nelement analysis"]
        WF["workflows.go\nexecute/list/update"]
        REC["recordings.go\ningest archives"]
    end

    subgraph Services["services/*"]
        WSVC["WorkflowService\n(engine selection,\nexecutor orchestration)"]
        RSVC["RecordingService\narchive normalization"]
        RENDER["ReplayRenderer"]
    end

    subgraph Automation["automation/*"]
        PLAN["executor.PlanCompiler\nengine-aware"]
        EXEC["executor.SimpleExecutor"]
        ENG["engine.Factory\n(DefaultFactory)"]
        RECORDER["recorder.DBRecorder"]
        EVENTS["events.WSHubSink\nper-execution queues"]
    end

    subgraph IO["IO & Infra"]
        DB[(Postgres Repo)]
        MINIO["MinIO"]
        BROW["Browserless or future engine"]
        HUB["websocket Hub"]
    end

    REST --> H --> WSVC
    AIH --> WSVC
    WF --> WSVC
    REC --> RSVC
    WSVC --> PLAN --> EXEC
    EXEC --> ENG --> BROW
    EXEC --> RECORDER --> DB
    EXEC --> EVENTS --> HUB
    RECORDER --> MINIO
    WSVC --> RENDER
```

Notes:
- Handlers are intentionally logic-light; business rules live in `services/`.
- Engine choice honors `ENGINE` / `ENGINE_OVERRIDE` and flows through `automation/engine.DefaultFactory`, keeping runtime engine-agnostic as new engines arrive.
- Websocket clients receive ordered automation envelopes via `WSHubSink`; payloads are contract `EventEnvelope` objects.
