# Recording Ingestion

Engine-agnostic ingestion path for Chrome extension recordings. Converts archive manifests into `automation/contracts` outcomes and persists via the recorder so replay/export stays consistent with automation runs.

```mermaid
flowchart LR
    subgraph Input["Archive Input"]
        ZIP["recording.zip\nmanifest + frames/*"]
    end

    subgraph Adapter["recording_adapter.go"]
        MAN["Manifest parser\n(normalize frames)"]
        OUT["Frame → contracts.StepOutcome\n(screenshot, DOM, console,\nnetwork, cursor trail)"]
    end

    subgraph Writer["automation/execution-writer.FileWriter"]
        STEPS["execution_steps"]
        ART["execution_artifacts\n(timeline_frame, screenshot,\nconsole, network, dom_snapshot)"]
    end

    subgraph Storage["Storage"]
        MINIO["MinIO (if configured)"]
        LOCAL["Local recordings root\n(fallback/forced via BAS_RECORDING_STORAGE=local)"]
    end

    ZIP --> MAN --> OUT --> Recorder
    OUT --> Storage
    Recorder --> STEPS
    Recorder --> ART
    Storage --> MINIO
    Storage --> LOCAL
```

Key behaviors:
- Storage selection: `BAS_RECORDING_STORAGE=local` forces filesystem storage; otherwise uses the injected storage client (MinIO when available) with filesystem fallback.
- Contract-first: outcomes use `automation/contracts` (no browserless runtime types). Screenshots are persisted through the recorder and referenced from timeline payloads.
- Timeline parity: recorder emits `timeline_frame` artifacts with screenshot IDs, DOM snapshot references, and cursor/overlay metadata so replay/export stays consistent with automation executions.
- Safety limits: frames enforce max asset size (`maxRecordingAssetBytes`) and max archive size (`maxRecordingArchiveBytes`); oversize assets fail fast.
