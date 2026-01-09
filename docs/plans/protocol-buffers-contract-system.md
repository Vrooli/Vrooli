# Protocol Buffers Contract System Implementation Plan

> **Status:** Draft
> **Created:** 2025-12-05
> **Last Updated:** 2025-12-05
> **Author:** Claude + Human collaboration

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Problem Statement](#problem-statement)
3. [Why Protocol Buffers](#why-protocol-buffers)
4. [Architecture Overview](#architecture-overview)
5. [Implementation Phases](#implementation-phases)
6. [Technical Specifications](#technical-specifications)
7. [Migration Strategy](#migration-strategy)
8. [Governance & Workflow](#governance--workflow)
9. [Risk Assessment](#risk-assessment)
10. [Success Criteria](#success-criteria)
11. [Appendix](#appendix)

---

## Executive Summary

This document outlines the implementation of a **Protocol Buffers-based contract system** for Vrooli's inter-scenario communication. As Vrooli scales to tens of thousands of scenarios written in arbitrary languages (Go, Python, TypeScript, Rust, C, etc.), we need a robust way to:

1. **Define contracts** between scenarios that call each other
2. **Generate type-safe code** for all languages
3. **Detect breaking changes** before they reach production
4. **Document APIs** automatically
5. **Track dependencies** between scenarios

The chosen solution uses **Protocol Buffers** with the **Buf** toolchain, providing:
- Universal language support with consistent code generation
- Built-in backward/forward compatibility
- Automatic breaking change detection
- A mature ecosystem battle-tested at Google scale

**Initial scope:** Implement contracts for the `browser-automation-studio` ↔ `test-genie` integration (the Playbooks phase), then expand to all inter-scenario communication.

---

## Problem Statement

### Current State

Scenarios communicate via HTTP/JSON APIs, but contracts are **implicit**:

```
test-genie                          browser-automation-studio
┌────────────────────┐              ┌────────────────────┐
│                    │   HTTP/JSON  │                    │
│  timeline_parser.go│◄────────────│  executions.go     │
│                    │              │                    │
│  Expects:          │              │  Returns:          │
│  - execution_id    │              │  - execution_id    │
│  - status          │              │  - status          │
│  - frames[]        │              │  - frames[]        │
│  - frames[].error  │              │  - frames[].error? │
│                    │              │                    │
│  WHO KNOWS IF      │              │  WHO KNOWS IF      │
│  THESE MATCH?      │              │  THESE MATCH?      │
└────────────────────┘              └────────────────────┘
```

**Problems discovered during Playbooks phase investigation:**

1. **No shared contract** - `timeline_parser.go` (297 lines) assumes a JSON structure that may not match what BAS actually returns
2. **Silent failures** - If BAS changes its response format, test-genie gets empty/wrong data without clear errors
3. **Inconsistent field usage** - BAS uses `FailureReason`, `Error`, and `Status` inconsistently; test-genie checks them in arbitrary order
4. **No integration tests** - The actual protocol between systems is never verified
5. **Manual parsing** - Every consumer writes their own parsing code, which drifts over time

### Scale Problem

Vrooli aims to have **tens of thousands of scenarios**. Without contracts:

- One breaking change cascades to hundreds of consumers
- No way to know "who uses this API?"
- No way to know "is this change safe?"
- Documentation is always stale
- Every language reimplements parsing differently

### Specific Pain Point: test-genie Playbooks Phase

The Playbooks phase in test-genie has never worked correctly, partly because:

1. test-genie's `timeline_parser.go` expects a specific JSON structure
2. BAS's actual response may differ
3. No validation catches the mismatch
4. Errors are vague: "workflow failed" without context

---

## Why Protocol Buffers

We evaluated multiple approaches for contract management:

| Approach | Language Support | Code Gen | Breaking Detection | Versioning | Verdict |
|----------|-----------------|----------|-------------------|------------|---------|
| **JSON Schema** | Universal | Inconsistent | Build yourself | Build yourself | Good, but requires custom tooling |
| **OpenAPI** | Universal | Good | Limited | Manual | Heavyweight for internal APIs |
| **Shared Go Types** | Go only | N/A | Manual | Manual | Doesn't work for multi-language |
| **Protocol Buffers** | 15+ languages | Excellent | Built-in (buf) | First-class | **Selected** |
| **GraphQL** | Good | Good | Schema-based | Built-in | Better for query APIs, not internal contracts |

### Why Protocol Buffers Won

1. **Battle-tested at scale** - Google runs millions of services with protobuf; 10,000+ scenarios is well within design parameters

2. **Consistent code generation** - Generated code looks the same in every language, reducing cognitive load

3. **Breaking change detection** - `buf breaking` catches incompatible changes automatically in CI

4. **First-class versioning** - Field numbers enable backward/forward compatibility by design

5. **Buf ecosystem** - Modern tooling with excellent DX (linting, formatting, registry)

6. **HTTP/JSON compatible** - `protojson` allows existing REST APIs to use proto types without switching to gRPC

7. **Type safety** - Compile-time errors when contracts are violated, not runtime surprises

### What We're NOT Doing

- **Not switching to gRPC** - We keep HTTP/JSON APIs, just use proto-generated types
- **Not requiring binary format** - JSON serialization via `protojson`
- **Not building a registry service** - Using Buf BSR (hosted) or file-based approach

---

## Architecture Overview

### Package Structure

```
packages/
└── proto/
    ├── Makefile                    # Targets: generate, lint, breaking, check
    ├── README.md                   # Developer documentation
    ├── buf.yaml                    # Buf module configuration
    ├── buf.gen.yaml                # Code generation configuration
    ├── buf.lock                    # Dependency lock file
    │
    ├── schemas/                    # SOURCE OF TRUTH: Human-authored .proto files
    │   │
    │   ├── bas/                    # Browser Automation Studio contracts
    │   │   └── v1/
    │   │       ├── timeline.proto      # GET /executions/{id}/timeline
    │   │       ├── execution.proto     # Execution status, GET /executions/{id}
    │   │       ├── workflow.proto      # Workflow definition schema
    │   │       └── service.proto       # Optional: Full service definition
    │   │
    │   ├── testgenie/              # Test Genie contracts
    │   │   └── v1/
    │   │       ├── phase_result.proto  # Phase execution results
    │   │       ├── playbook.proto      # Playbook registry format
    │   │       └── report.proto        # Test report format
    │   │
    │   ├── common/                 # Shared types across scenarios
    │   │   └── v1/
    │   │       ├── pagination.proto    # Pagination request/response
    │   │       ├── error.proto         # Standard error response
    │   │       ├── health.proto        # Health check response
    │   │       └── timestamp.proto     # Re-export google.protobuf.Timestamp
    │   │
    │   └── [future scenarios]/     # Added as scenarios adopt contracts
    │
    └── gen/                        # GENERATED: Auto-generated code (committed)
        ├── go/
        │   ├── bas/v1/
        │   │   ├── timeline.pb.go
        │   │   ├── execution.pb.go
        │   │   └── workflow.pb.go
        │   ├── testgenie/v1/
        │   └── common/v1/
        │
        ├── typescript/
        │   ├── bas/v1/
        │   ├── testgenie/v1/
        │   └── common/v1/
        │
        └── python/
            ├── bas/v1/
            ├── testgenie/v1/
            └── common/v1/
```

### Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Developer Workflow                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. Edit .proto file (source of truth)                                     │
│     └── packages/proto/schemas/bas/v1/timeline.proto                       │
│                                                                             │
│  2. Generate code                                                           │
│     └── cd packages/proto && make generate                                  │
│         └── buf generate                                                    │
│             ├── gen/go/bas/v1/timeline.pb.go                               │
│             ├── gen/typescript/bas/v1/timeline.ts                          │
│             └── gen/python/bas/v1/timeline_pb2.py                          │
│                                                                             │
│  3. Use in scenarios                                                        │
│     ├── BAS: import basv1 "packages/proto/gen/go/bas/v1"                   │
│     └── test-genie: import basv1 "packages/proto/gen/go/bas/v1"            │
│                                                                             │
│  4. Commit all changes                                                      │
│     └── git add packages/proto/ scenarios/                                  │
│                                                                             │
│  5. CI validates                                                            │
│     ├── buf lint (style/quality)                                           │
│     ├── buf breaking (no breaking changes)                                 │
│     └── generated code matches protos                                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### How Scenarios Use Contracts

**Producer (BAS) - returns proto-typed responses:**

```go
package handlers

import (
    "google.golang.org/protobuf/encoding/protojson"
    basv1 "packages/proto/gen/go/bas/v1"
)

func (h *Handler) GetExecutionTimeline(w http.ResponseWriter, r *http.Request) {
    exec := h.loadExecution(r.Context(), executionID)

    // Build response using generated type
    // Compiler enforces all fields are correct types
    timeline := &basv1.ExecutionTimeline{
        ExecutionId: exec.ID.String(),
        Status:      toProtoStatus(exec.Status),
        Progress:    int32(exec.Progress),
        StartedAt:   timestamppb.New(exec.StartedAt),
        Frames:      toProtoFrames(exec.Steps),
    }

    // Serialize to JSON using proto-aware marshaler
    data, err := protojson.MarshalOptions{
        UseProtoNames:   true,  // snake_case (execution_id not executionId)
        EmitUnpopulated: false, // omit empty/zero fields
    }.Marshal(timeline)

    w.Header().Set("Content-Type", "application/json")
    w.Write(data)
}
```

**Consumer (test-genie) - expects proto-typed responses:**

```go
package execution

import (
    "google.golang.org/protobuf/encoding/protojson"
    basv1 "packages/proto/gen/go/bas/v1"
)

func (c *Client) GetTimeline(ctx context.Context, execID string) (*basv1.ExecutionTimeline, error) {
    data, err := c.fetchRaw(ctx, "/executions/%s/timeline", execID)
    if err != nil {
        return nil, err
    }

    // Unmarshal using proto-aware unmarshaler
    var timeline basv1.ExecutionTimeline
    if err := protojson.Unmarshal(data, &timeline); err != nil {
        return nil, fmt.Errorf("BAS returned invalid timeline: %w", err)
    }

    // Type-safe access - compiler knows all fields
    for _, frame := range timeline.Frames {
        if frame.Status == basv1.FrameStatus_FRAME_STATUS_FAILED {
            // frame.Error is *string (optional), properly typed
        }
    }

    return &timeline, nil
}
```

**Both sides use the same generated type** - drift is impossible.

---

## Implementation Phases

### Phase 0: Infrastructure Setup (1-2 days)

**Goal:** Set up the proto package structure and tooling.

#### Tasks

- [ ] **0.1** Install Buf CLI and add to project tooling
  ```bash
  # Add to scripts/setup or similar
  curl -sSL https://github.com/bufbuild/buf/releases/download/v1.28.1/buf-Linux-x86_64 -o /usr/local/bin/buf
  chmod +x /usr/local/bin/buf
  ```

- [ ] **0.2** Create package directory structure
  ```bash
  mkdir -p packages/proto/{schemas,gen}/{bas,testgenie,common}/v1
  ```

- [ ] **0.3** Create `buf.yaml` (module configuration)
  ```yaml
  # packages/proto/buf.yaml
  version: v2
  modules:
    - path: schemas
      name: buf.build/vrooli/schemas
  lint:
    use:
      - DEFAULT
      - COMMENTS
    except:
      - PACKAGE_VERSION_SUFFIX
  breaking:
    use:
      - FILE
  ```

- [ ] **0.4** Create `buf.gen.yaml` (code generation configuration)
  ```yaml
  # packages/proto/buf.gen.yaml
  version: v2
  managed:
    enabled: true
    override:
      - file_option: go_package_prefix
        value: github.com/vrooli/vrooli/packages/proto/gen/go
  plugins:
    # Go
    - remote: buf.build/protocolbuffers/go
      out: gen/go
      opt: paths=source_relative

    # TypeScript (using connect-es for modern TS)
    - remote: buf.build/bufbuild/es
      out: gen/typescript
      opt: target=ts

    # Python
    - remote: buf.build/protocolbuffers/python
      out: gen/python

    # Python type stubs
    - remote: buf.build/protocolbuffers/pyi
      out: gen/python
  ```

- [ ] **0.5** Create `Makefile`
  ```makefile
  # packages/proto/Makefile
  .PHONY: generate lint breaking check clean

  generate:
  	buf generate

  lint:
  	buf lint

  format:
  	buf format -w

  breaking:
  	buf breaking --against '.git#branch=master,subdir=packages/proto'

  check: lint
  	buf generate
  	@if ! git diff --quiet gen/; then \
  		echo "ERROR: Generated code is out of date. Run 'make generate' and commit."; \
  		git diff --stat gen/; \
  		exit 1; \
  	fi

  clean:
  	rm -rf gen/
  ```

- [ ] **0.6** Create `README.md` with developer documentation

- [ ] **0.7** Add CI workflow for proto validation
  ```yaml
  # .github/workflows/proto.yml
  name: Proto Validation

  on:
    push:
      paths: ['packages/proto/**']
    pull_request:
      paths: ['packages/proto/**']

  jobs:
    validate:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
          with:
            fetch-depth: 0  # Needed for breaking change detection

        - uses: bufbuild/buf-setup-action@v1
          with:
            version: '1.28.1'

        - name: Lint
          run: buf lint
          working-directory: packages/proto

        - name: Check breaking changes
          run: buf breaking --against '.git#branch=master,subdir=packages/proto'
          working-directory: packages/proto

        - name: Verify generated code
          run: |
            buf generate
            git diff --exit-code gen/ || {
              echo "Generated code out of date. Run 'cd packages/proto && make generate'"
              exit 1
            }
          working-directory: packages/proto
  ```

#### Deliverables
- `packages/proto/` directory with all configuration
- `buf` CLI available in development environment
- CI workflow validating protos on every push

---

### Phase 1: First Contract - BAS Timeline (2-3 days)

**Goal:** Define the BAS ExecutionTimeline contract and generate code.

#### Tasks

- [ ] **1.1** Create common types proto
  ```protobuf
  // packages/proto/schemas/common/v1/types.proto
  syntax = "proto3";
  package common.v1;

  option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1;commonv1";

  // Pagination for list endpoints
  message PaginationRequest {
    int32 limit = 1;
    int32 offset = 2;
  }

  message PaginationResponse {
    int32 total = 1;
    int32 limit = 2;
    int32 offset = 3;
    bool has_more = 4;
  }

  // Standard error response
  message ErrorResponse {
    string code = 1;
    string message = 2;
    map<string, string> details = 3;
  }

  // Health check response
  message HealthResponse {
    string status = 1;  // "healthy", "degraded", "unhealthy"
    string version = 2;
    map<string, string> checks = 3;
  }
  ```

- [ ] **1.2** Create BAS timeline proto (the critical contract)
  ```protobuf
  // packages/proto/schemas/bas/v1/timeline.proto
  syntax = "proto3";
  package bas.v1;

  import "google/protobuf/timestamp.proto";

  option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/bas/v1;basv1";

  // ExecutionTimeline is returned by GET /api/v1/executions/{id}/timeline
  // This is the primary contract between BAS and test-genie's playbooks phase.
  message ExecutionTimeline {
    // Unique identifier for this execution (UUID format)
    string execution_id = 1;

    // Workflow ID if this was a saved workflow (not adhoc)
    optional string workflow_id = 2;

    // Current execution state
    ExecutionStatus status = 3;

    // Progress percentage (0-100)
    int32 progress = 4;

    // When execution started
    google.protobuf.Timestamp started_at = 5;

    // When execution completed (set when status is COMPLETED or FAILED)
    optional google.protobuf.Timestamp completed_at = 6;

    // Ordered list of execution frames (one per workflow step)
    repeated TimelineFrame frames = 7;

    // Console/execution logs
    repeated TimelineLog logs = 8;
  }

  // ExecutionStatus represents the state of a workflow execution.
  enum ExecutionStatus {
    EXECUTION_STATUS_UNSPECIFIED = 0;
    EXECUTION_STATUS_PENDING = 1;
    EXECUTION_STATUS_RUNNING = 2;
    EXECUTION_STATUS_COMPLETED = 3;
    EXECUTION_STATUS_FAILED = 4;
    EXECUTION_STATUS_CANCELLED = 5;
  }

  // TimelineFrame contains data for a single execution step.
  message TimelineFrame {
    // Zero-based index of this step in the execution
    int32 step_index = 1;

    // Node ID from the workflow definition
    string node_id = 2;

    // Node type (navigate, click, assert, etc.)
    string step_type = 3;

    // Step execution status
    FrameStatus status = 4;

    // Whether this step succeeded
    bool success = 5;

    // Execution duration in milliseconds
    int32 duration_ms = 6;

    // Error message if step failed
    optional string error = 7;

    // Final URL after step execution (for navigate steps)
    optional string final_url = 8;

    // Progress percentage at this step (0-100)
    int32 progress = 9;

    // Screenshot captured during/after this step
    optional ScreenshotRef screenshot = 10;

    // DOM snapshot captured during/after this step
    optional DOMSnapshot dom_snapshot = 11;

    // Assertion result (for assert steps)
    optional AssertionResult assertion = 12;
  }

  // FrameStatus represents the state of a single step.
  enum FrameStatus {
    FRAME_STATUS_UNSPECIFIED = 0;
    FRAME_STATUS_PENDING = 1;
    FRAME_STATUS_RUNNING = 2;
    FRAME_STATUS_COMPLETED = 3;
    FRAME_STATUS_FAILED = 4;
    FRAME_STATUS_SKIPPED = 5;
  }

  // ScreenshotRef contains metadata about a captured screenshot.
  message ScreenshotRef {
    // Unique artifact ID
    string artifact_id = 1;

    // URL to download the full screenshot
    string url = 2;

    // URL to download a thumbnail (optional)
    optional string thumbnail_url = 3;

    // Image dimensions
    int32 width = 4;
    int32 height = 5;
  }

  // DOMSnapshot contains metadata about a captured DOM state.
  message DOMSnapshot {
    // Unique artifact ID
    string artifact_id = 1;

    // Truncated HTML preview (first ~500 chars)
    optional string preview = 2;

    // URL to download full DOM HTML
    string storage_url = 3;
  }

  // AssertionResult contains the outcome of an assert step.
  message AssertionResult {
    // Whether the assertion passed
    bool passed = 1;

    // Type of assertion (exists, visible, text_equals, etc.)
    string assertion_type = 2;

    // CSS selector being asserted on
    optional string selector = 3;

    // Expected value (for comparison assertions)
    optional string expected = 4;

    // Actual value found
    optional string actual = 5;

    // Custom failure message from workflow
    optional string message = 6;
  }

  // TimelineLog represents a single log entry during execution.
  message TimelineLog {
    // Unique log entry ID
    string id = 1;

    // Log level
    LogLevel level = 2;

    // Log message content
    string message = 3;

    // Associated step name (if applicable)
    optional string step_name = 4;

    // When the log was recorded
    google.protobuf.Timestamp timestamp = 5;
  }

  // LogLevel for timeline logs.
  enum LogLevel {
    LOG_LEVEL_UNSPECIFIED = 0;
    LOG_LEVEL_DEBUG = 1;
    LOG_LEVEL_INFO = 2;
    LOG_LEVEL_WARN = 3;
    LOG_LEVEL_ERROR = 4;
  }
  ```

- [ ] **1.3** Create BAS execution status proto
  ```protobuf
  // packages/proto/schemas/bas/v1/execution.proto
  syntax = "proto3";
  package bas.v1;

  import "google/protobuf/timestamp.proto";
  import "bas/v1/timeline.proto";

  option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/bas/v1;basv1";

  // Execution is returned by GET /api/v1/executions/{id}
  // Used by test-genie to poll execution status.
  message Execution {
    // Unique execution ID
    string id = 1;

    // Associated workflow ID (if not adhoc)
    optional string workflow_id = 2;

    // Workflow version at time of execution
    int32 workflow_version = 3;

    // Current status
    ExecutionStatus status = 4;

    // How the execution was triggered
    TriggerType trigger_type = 5;

    // Parameters passed to execution
    map<string, string> parameters = 6;

    // Progress percentage (0-100)
    int32 progress = 7;

    // Human-readable description of current step
    string current_step = 8;

    // Timestamps
    google.protobuf.Timestamp started_at = 9;
    optional google.protobuf.Timestamp completed_at = 10;

    // Error message if failed
    optional string error = 11;

    // Detailed failure reason (more context than error)
    optional string failure_reason = 12;
  }

  // TriggerType indicates how an execution was started.
  enum TriggerType {
    TRIGGER_TYPE_UNSPECIFIED = 0;
    TRIGGER_TYPE_MANUAL = 1;      // User clicked "Run"
    TRIGGER_TYPE_SCHEDULED = 2;    // Cron/scheduled execution
    TRIGGER_TYPE_API = 3;          // API call (including test-genie)
    TRIGGER_TYPE_WEBHOOK = 4;      // External webhook trigger
  }

  // ExecuteAdhocRequest is the request body for POST /api/v1/workflows/execute-adhoc
  message ExecuteAdhocRequest {
    // The workflow definition to execute
    WorkflowDefinition flow_definition = 1;

    // Runtime parameters
    map<string, string> parameters = 2;

    // Whether to wait for completion before responding
    bool wait_for_completion = 3;

    // Optional metadata for logging/debugging
    optional ExecutionMetadata metadata = 4;
  }

  message ExecutionMetadata {
    string name = 1;
    string description = 2;
  }

  // ExecuteAdhocResponse is returned by POST /api/v1/workflows/execute-adhoc
  message ExecuteAdhocResponse {
    // The created execution ID
    string execution_id = 1;

    // Initial status (usually PENDING or RUNNING)
    ExecutionStatus status = 2;

    // If wait_for_completion was true, final status info
    optional google.protobuf.Timestamp completed_at = 3;
    optional string error = 4;
  }
  ```

- [ ] **1.4** Create BAS workflow definition proto (for ExecuteAdhocRequest)
  ```protobuf
  // packages/proto/schemas/bas/v1/workflow.proto
  syntax = "proto3";
  package bas.v1;

  option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/bas/v1;basv1";

  // WorkflowDefinition is the schema for workflow JSON files.
  // Used in execute-adhoc requests and stored workflows.
  message WorkflowDefinition {
    // Workflow metadata
    optional WorkflowMetadata metadata = 1;

    // Execution settings
    optional WorkflowSettings settings = 2;

    // Workflow nodes (steps)
    repeated WorkflowNode nodes = 3;

    // Connections between nodes
    repeated WorkflowEdge edges = 4;
  }

  message WorkflowMetadata {
    string description = 1;
    int32 version = 2;
    optional string reset = 3;  // "none" or "full"
  }

  message WorkflowSettings {
    optional ViewportSettings execution_viewport = 1;
  }

  message ViewportSettings {
    int32 width = 1;
    int32 height = 2;
    string preset = 3;  // "desktop", "mobile", "custom"
  }

  message WorkflowNode {
    string id = 1;
    string type = 2;  // "navigate", "click", "type", "assert", "wait", etc.
    map<string, google.protobuf.Value> data = 3;  // Node-specific configuration
  }

  message WorkflowEdge {
    string id = 1;
    string source = 2;
    string target = 3;
    optional string type = 4;  // "smoothstep", etc.
  }
  ```

- [ ] **1.5** Generate code and verify
  ```bash
  cd packages/proto
  make generate
  make lint
  ```

- [ ] **1.6** Write example/test code demonstrating usage
  ```go
  // packages/proto/examples/bas_timeline_test.go
  package examples_test

  import (
      "testing"
      "google.golang.org/protobuf/encoding/protojson"
      basv1 "packages/proto/gen/go/bas/v1"
  )

  func TestTimelineSerialization(t *testing.T) {
      timeline := &basv1.ExecutionTimeline{
          ExecutionId: "abc-123",
          Status:      basv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
          Progress:    100,
          Frames: []*basv1.TimelineFrame{
              {
                  StepIndex: 0,
                  NodeId:    "navigate-1",
                  StepType:  "navigate",
                  Status:    basv1.FrameStatus_FRAME_STATUS_COMPLETED,
                  Success:   true,
              },
          },
      }

      data, err := protojson.Marshal(timeline)
      if err != nil {
          t.Fatalf("marshal failed: %v", err)
      }

      var parsed basv1.ExecutionTimeline
      if err := protojson.Unmarshal(data, &parsed); err != nil {
          t.Fatalf("unmarshal failed: %v", err)
      }

      if parsed.ExecutionId != timeline.ExecutionId {
          t.Errorf("execution_id mismatch")
      }
  }
  ```

#### Deliverables
- `schemas/common/v1/types.proto` - Shared types
- `schemas/bas/v1/timeline.proto` - Timeline contract
- `schemas/bas/v1/execution.proto` - Execution status contract
- `schemas/bas/v1/workflow.proto` - Workflow definition schema
- Generated code in `gen/go/`, `gen/typescript/`, `gen/python/`
- Example test demonstrating usage

---

### Phase 2: Migrate BAS (2-3 days)

**Goal:** Update BAS to use generated types for timeline and execution responses.

#### Tasks

- [ ] **2.1** Add proto dependency to BAS
  ```go
  // In scenarios/browser-automation-studio/api/go.mod
  require (
      github.com/vrooli/vrooli/packages/proto v0.0.0
      google.golang.org/protobuf v1.31.0
  )

  replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
  ```

- [ ] **2.2** Create conversion functions (internal models → proto types)
  ```go
  // scenarios/browser-automation-studio/api/proto/convert.go
  package proto

  import (
      "github.com/vrooli/browser-automation-studio/database"
      basv1 "packages/proto/gen/go/bas/v1"
      "google.golang.org/protobuf/types/known/timestamppb"
  )

  func ExecutionToProto(exec *database.Execution) *basv1.Execution {
      pb := &basv1.Execution{
          Id:              exec.ID.String(),
          WorkflowVersion: int32(exec.WorkflowVersion),
          Status:          statusToProto(exec.Status),
          Progress:        int32(exec.Progress),
          CurrentStep:     exec.CurrentStep,
          StartedAt:       timestamppb.New(exec.StartedAt),
      }

      if exec.WorkflowID != nil {
          s := exec.WorkflowID.String()
          pb.WorkflowId = &s
      }

      if exec.CompletedAt != nil {
          pb.CompletedAt = timestamppb.New(*exec.CompletedAt)
      }

      if exec.Error.Valid {
          pb.Error = &exec.Error.String
      }

      return pb
  }

  func statusToProto(s string) basv1.ExecutionStatus {
      switch s {
      case "pending":
          return basv1.ExecutionStatus_EXECUTION_STATUS_PENDING
      case "running":
          return basv1.ExecutionStatus_EXECUTION_STATUS_RUNNING
      case "completed":
          return basv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED
      case "failed":
          return basv1.ExecutionStatus_EXECUTION_STATUS_FAILED
      case "cancelled":
          return basv1.ExecutionStatus_EXECUTION_STATUS_CANCELLED
      default:
          return basv1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
      }
  }

  // TimelineToProto converts internal timeline to proto format
  func TimelineToProto(timeline *service.ExecutionTimeline) *basv1.ExecutionTimeline {
      // ... conversion logic
  }
  ```

- [ ] **2.3** Update BAS handlers to use proto types
  ```go
  // scenarios/browser-automation-studio/api/handlers/executions.go
  import (
      "google.golang.org/protobuf/encoding/protojson"
      basv1 "packages/proto/gen/go/bas/v1"
      protoconv "github.com/vrooli/browser-automation-studio/proto"
  )

  var jsonMarshaler = protojson.MarshalOptions{
      UseProtoNames:   true,  // snake_case field names
      EmitUnpopulated: false, // omit empty fields
  }

  func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
      // ... existing logic to get execution ...

      // Convert to proto type
      pb := protoconv.ExecutionToProto(execution)

      // Serialize with protojson
      data, err := jsonMarshaler.Marshal(pb)
      if err != nil {
          h.respondError(w, ErrInternalServer)
          return
      }

      w.Header().Set("Content-Type", "application/json")
      w.Write(data)
  }

  func (h *Handler) GetExecutionTimeline(w http.ResponseWriter, r *http.Request) {
      // ... existing logic ...

      pb := protoconv.TimelineToProto(timeline)
      data, err := jsonMarshaler.Marshal(pb)
      // ...
  }
  ```

- [ ] **2.4** Update BAS execute-adhoc to accept proto-compatible requests
  - The existing JSON structure should be compatible
  - Add validation using proto schema if desired

- [ ] **2.5** Test BAS endpoints return valid proto-JSON
  ```go
  func TestTimelineMatchesProto(t *testing.T) {
      // Call BAS endpoint
      resp, _ := http.Get(basURL + "/api/v1/executions/" + execID + "/timeline")
      data, _ := io.ReadAll(resp.Body)

      // Verify it unmarshals to proto type
      var timeline basv1.ExecutionTimeline
      err := protojson.Unmarshal(data, &timeline)
      if err != nil {
          t.Fatalf("BAS response doesn't match proto schema: %v", err)
      }
  }
  ```

#### Deliverables
- BAS handlers updated to use proto types
- Conversion functions from internal models to proto
- Integration tests verifying proto compliance
- No breaking changes to API (same JSON structure)

---

### Phase 3: Migrate test-genie (2-3 days)

**Goal:** Update test-genie to use generated types instead of `timeline_parser.go`.

#### Tasks

- [ ] **3.1** Add proto dependency to test-genie
  ```go
  // In scenarios/test-genie/api/go.mod
  require (
      github.com/vrooli/vrooli/packages/proto v0.0.0
  )

  replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
  ```

- [ ] **3.2** Update BAS client to return proto types
  ```go
  // scenarios/test-genie/api/internal/playbooks/execution/client.go
  import (
      "google.golang.org/protobuf/encoding/protojson"
      basv1 "packages/proto/gen/go/bas/v1"
  )

  // GetTimeline retrieves the timeline using the proto contract.
  func (c *HTTPClient) GetTimeline(ctx context.Context, executionID string) (*basv1.ExecutionTimeline, error) {
      data, err := c.fetchRaw(ctx, "/executions/%s/timeline", executionID)
      if err != nil {
          return nil, err
      }

      var timeline basv1.ExecutionTimeline
      if err := protojson.Unmarshal(data, &timeline); err != nil {
          return nil, fmt.Errorf("BAS returned invalid timeline (proto violation): %w", err)
      }

      return &timeline, nil
  }

  // GetStatus retrieves execution status using the proto contract.
  func (c *HTTPClient) GetStatus(ctx context.Context, executionID string) (*basv1.Execution, error) {
      data, err := c.fetchRaw(ctx, "/executions/%s", executionID)
      if err != nil {
          return nil, err
      }

      var exec basv1.Execution
      if err := protojson.Unmarshal(data, &exec); err != nil {
          return nil, fmt.Errorf("BAS returned invalid execution (proto violation): %w", err)
      }

      return &exec, nil
  }
  ```

- [ ] **3.3** Remove or deprecate `timeline_parser.go`
  - The 297-line manual parser is replaced by `protojson.Unmarshal`
  - Keep temporarily for comparison testing, then delete

- [ ] **3.4** Update runner.go to use proto types
  ```go
  // Direct field access with proper types
  for _, frame := range timeline.Frames {
      if frame.Status == basv1.FrameStatus_FRAME_STATUS_FAILED {
          // frame.Error is *string, properly typed
          if frame.Error != nil {
              log.Printf("Step %d failed: %s", frame.StepIndex, *frame.Error)
          }
      }

      if frame.Screenshot != nil {
          // Download screenshot
          data, _ := c.DownloadAsset(ctx, frame.Screenshot.Url)
      }
  }
  ```

- [ ] **3.5** Update error handling with better context
  ```go
  // Proto errors are clear about what field failed
  if err := protojson.Unmarshal(data, &timeline); err != nil {
      return nil, &ContractViolationError{
          Endpoint: "/executions/{id}/timeline",
          Schema:   "bas.v1.ExecutionTimeline",
          RawData:  data,
          Cause:    err,
      }
  }
  ```

- [ ] **3.6** Integration test: Playbooks phase with real BAS
  ```go
  func TestPlaybooksPhaseWithBAS(t *testing.T) {
      if testing.Short() {
          t.Skip("integration test")
      }

      // Start BAS
      // Run a simple playbook
      // Verify timeline is parsed correctly
      // Verify artifacts are collected
  }
  ```

#### Deliverables
- test-genie using proto types for BAS communication
- Removed/deprecated manual timeline parser
- Integration test proving the contract works
- Clear error messages on contract violations

---

### Phase 4: CI/CD & Governance (1-2 days)

**Goal:** Ensure contracts are validated on every change.

#### Tasks

- [ ] **4.1** Add proto validation to CI (already done in Phase 0)

- [ ] **4.2** Add contract test job that runs BAS + test-genie together
  ```yaml
  contract-test:
    runs-on: ubuntu-latest
    needs: [build-bas, build-test-genie]
    steps:
      - name: Start BAS
        run: make -C scenarios/browser-automation-studio start

      - name: Run playbooks phase
        run: |
          cd scenarios/test-genie
          ./api/test-genie execute browser-automation-studio --phase playbooks

      - name: Verify no contract errors
        run: |
          # Check logs for "proto violation" or "contract violation"
          ! grep -i "violation" test-output.log
  ```

- [ ] **4.3** Document contract change process
  ```markdown
  ## How to Change a Contract

  1. Edit the `.proto` file in `packages/proto/schemas/`
  2. Run `cd packages/proto && make generate`
  3. Run `make breaking` to check for breaking changes
  4. If breaking:
     - Bump the version (v1 → v2)
     - Or add field as optional (non-breaking)
  5. Update producer (BAS) to populate new fields
  6. Update consumer (test-genie) to use new fields
  7. Commit all changes together
  8. CI will validate everything
  ```

- [ ] **4.4** Add pre-commit hook for proto changes
  ```bash
  # .git/hooks/pre-commit
  if git diff --cached --name-only | grep -q "packages/proto/schemas/"; then
      echo "Proto files changed, regenerating..."
      cd packages/proto && make generate
      git add packages/proto/gen/
  fi
  ```

#### Deliverables
- CI validates all proto changes
- Contract tests run BAS + test-genie together
- Documentation for contract change process
- Pre-commit hook keeps generated code in sync

---

### Phase 5: Expand to More Contracts (Ongoing)

**Goal:** Add contracts for other inter-scenario communication.

#### Candidates (in priority order)

1. **test-genie → scenario (any)** - Smoke phase UI checks
2. **deployment-manager → scenario (any)** - Deployment status
3. **ecosystem-manager → scenario (any)** - Scenario metadata
4. **prd-control-tower → scenario (any)** - Requirements data

#### Process for Each New Contract

1. Identify producer and consumer(s)
2. Document current implicit contract (what JSON is actually exchanged)
3. Write `.proto` file capturing that contract
4. Generate code
5. Update producer to use proto types
6. Update consumer(s) to use proto types
7. Add to CI contract tests

---

### Phase 6: Optional Schema Registry UI (Future)

**Goal:** Build a scenario for browsing/managing contracts.

```
scenarios/
└── schema-registry/
    ├── api/
    │   ├── main.go
    │   └── handlers/
    │       ├── schemas.go      # List/get schemas
    │       ├── consumers.go    # Who uses what
    │       └── breaking.go     # Breaking change history
    └── ui/
        └── src/
            ├── pages/
            │   ├── SchemaList.tsx
            │   ├── SchemaDetail.tsx
            │   └── DependencyGraph.tsx
            └── components/
                └── ProtoViewer.tsx
```

This is **nice-to-have** - the core value is in the package, not the UI.

---

## Technical Specifications

### Proto Style Guide

Follow [Buf's style guide](https://buf.build/docs/best-practices/style-guide/) with these additions:

```protobuf
// 1. Use snake_case for field names
string execution_id = 1;  // GOOD
string executionId = 1;   // BAD

// 2. Use SCREAMING_SNAKE_CASE for enum values, with prefix
enum Status {
  STATUS_UNSPECIFIED = 0;  // Always have UNSPECIFIED = 0
  STATUS_PENDING = 1;
  STATUS_RUNNING = 2;
}

// 3. Always add comments for public messages and fields
// ExecutionTimeline contains the full execution history.
// This is returned by GET /api/v1/executions/{id}/timeline.
message ExecutionTimeline {
  // Unique identifier for this execution (UUID format).
  string execution_id = 1;
}

// 4. Use optional for fields that may not be present
optional string error = 7;  // Only set on failure

// 5. Group related fields with comments
message TimelineFrame {
  // Identity
  int32 step_index = 1;
  string node_id = 2;

  // Status
  FrameStatus status = 3;
  bool success = 4;

  // Timing
  int32 duration_ms = 5;

  // Artifacts
  optional ScreenshotRef screenshot = 10;
}
```

### Versioning Strategy

- **Package versions**: `bas/v1`, `bas/v2`
- **When to bump version**:
  - Removing fields (breaking)
  - Changing field types (breaking)
  - Changing enum values (breaking)
- **Non-breaking changes** (stay in same version):
  - Adding optional fields
  - Adding enum values
  - Adding new messages

### JSON Field Naming

Use `protojson` with `UseProtoNames: true` to get snake_case:

```go
marshaler := protojson.MarshalOptions{
    UseProtoNames:   true,  // execution_id, not executionId
    EmitUnpopulated: false, // don't include empty fields
}
```

This matches existing BAS API field naming.

---

## Migration Strategy

### For Existing Endpoints

1. **Write proto matching current behavior** - Don't change the API, just formalize it
2. **Update producer** - Use proto types, serialize with protojson
3. **Test** - Verify JSON output is identical (or compatible)
4. **Update consumer** - Use proto types for parsing
5. **Remove old parsing code** - Delete manual struct definitions

### Backward Compatibility

- Adding optional fields is safe
- Adding enum values is safe (if consumers handle unknown values)
- Removing fields requires version bump
- Changing types requires version bump

### Rollback Plan

If proto migration causes issues:

1. Keep old parsing code during transition
2. Add feature flag: `USE_PROTO_TYPES=true`
3. Can revert to old code by setting `USE_PROTO_TYPES=false`
4. Remove flag and old code once stable

---

## Governance & Workflow

### Who Owns Contracts?

- **Producer owns the contract** - BAS team owns `bas/v1/*.proto`
- **Consumers propose changes** - test-genie team can propose changes via PR
- **Breaking changes require approval** - From both producer and affected consumers

### Change Review Process

1. **Non-breaking changes**: Normal PR review
2. **Breaking changes**:
   - Must update version (v1 → v2)
   - Must update all consumers in same PR
   - Requires approval from consumer teams

### Contract Deprecation

```protobuf
// Deprecated: Use execution_status instead. Will be removed in v3.
optional string status = 3 [deprecated = true];

ExecutionStatus execution_status = 15;
```

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Learning curve slows adoption | Medium | Medium | Good docs, examples, pairing sessions |
| Proto changes break consumers | Low | High | `buf breaking` in CI prevents this |
| Generated code bloat | Low | Low | Subpackages + tree-shaking |
| Toolchain complexity | Medium | Medium | Makefile abstracts buf commands |
| Drift during migration | Medium | Medium | Parallel old/new code, feature flags |

---

## Success Criteria

### Phase 1-3 (BAS ↔ test-genie)

- [ ] test-genie playbooks phase works correctly with BAS
- [ ] Contract violations produce clear error messages
- [ ] `buf breaking` catches breaking changes in CI
- [ ] Generated code is <100KB for BAS types

### Phase 4 (Governance)

- [ ] All proto changes go through CI validation
- [ ] Contract change process is documented
- [ ] At least one contract test runs in CI

### Long-term

- [ ] 10+ scenarios using proto contracts
- [ ] Zero production incidents from contract drift
- [ ] New scenario onboarding includes contract definition
- [ ] Schema registry UI (optional) provides visibility

---

## Appendix

### A. Buf CLI Commands Reference

```bash
# Generate code from protos
buf generate

# Lint proto files
buf lint

# Check for breaking changes vs main branch
buf breaking --against '.git#branch=main'

# Format proto files
buf format -w

# Show what would be generated
buf generate --dry-run
```

### B. Proto to JSON Mapping

| Proto Type | JSON Type | Example |
|------------|-----------|---------|
| `string` | `string` | `"abc"` |
| `int32` | `number` | `42` |
| `bool` | `boolean` | `true` |
| `optional string` | `string` or absent | `"abc"` or field missing |
| `repeated string` | `array` | `["a", "b"]` |
| `map<string, string>` | `object` | `{"key": "value"}` |
| `enum` | `string` | `"STATUS_COMPLETED"` |
| `google.protobuf.Timestamp` | `string` (RFC 3339) | `"2024-01-15T10:30:00Z"` |

### C. Example: Adding a New Field

```diff
// packages/proto/schemas/bas/v1/timeline.proto

message ExecutionTimeline {
  string execution_id = 1;
  ExecutionStatus status = 3;
  // ... existing fields ...

+ // Total execution duration in milliseconds.
+ // Added in v1.2 - optional for backward compatibility.
+ optional int64 total_duration_ms = 20;
}
```

```bash
cd packages/proto
make generate  # Regenerate code
make breaking  # Should pass (optional field is non-breaking)
```

### D. Related Documents

- [Buf Style Guide](https://buf.build/docs/best-practices/style-guide/)
- [Proto3 Language Guide](https://protobuf.dev/programming-guides/proto3/)
- [protojson Documentation](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson)

---

## Changelog

| Date | Author | Change |
|------|--------|--------|
| 2025-12-05 | Claude + Human | Initial draft |
