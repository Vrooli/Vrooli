# AI Helper Endpoints

These handlers provide AI-assisted utilities (DOM tree extraction, element analysis, preview screenshots) that reuse the automation engine so telemetry, retries, and artifact shaping stay consistent with workflow execution.

## Flow
```mermaid
flowchart TD
    subgraph Client["AI helper HTTP endpoints"]
        DOM[DOMHandler\n/dom-tree]
        EL[ElementAnalysisHandler\n/analyze-elements]
        SS[ScreenshotHandler\n/preview-screenshot]
    end
    subgraph Runner["automationRunner (in-process)"]
        PB[Plan & instructions\n(navigate → wait → evaluate → screenshot)]
        EX[SimpleExecutor]
    end
    subgraph Engine["BrowserlessEngine"]
        CDP[CDP session\n(per execution)]
    end
    subgraph Recorder["Recorder / Sinks"]
        MEM[In-memory Recorder\n(tests + helpers)]
        EVT[MemorySink\n(events/heartbeats)]
    end

    DOM --> PB
    EL --> PB
    SS --> PB
    PB --> EX --> Engine
    Engine -->|Run| EX
    EX --> MEM
    EX --> EVT
```

## Notes
- All helpers compile direct `CompiledInstruction` arrays (navigate → optional wait → evaluate/screenshot) and execute via `automationRunner` with the Browserless engine.
- DOM and element analysis now avoid the legacy `resource-browserless` CLI, ensuring heartbeats, retries, and capability checks flow through the new automation stack.
- The runner defaults to the configured engine selection; if unset, it falls back to localhost Browserless for local development.***
