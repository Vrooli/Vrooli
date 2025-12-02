# gpu - GPU Detection and Requirements

The `gpu` package detects available GPU hardware and enforces GPU requirements for services.

## Overview

Some services (particularly AI/ML workloads) benefit from or require GPU acceleration. This package detects GPU availability across platforms and configures services appropriately, including fallback to CPU when GPUs aren't available.

## Key Types

| Type | Purpose |
|------|---------|
| `Detector` | Interface for GPU detection |
| `RealDetector` | Production implementation |
| `Status` | Detection result with availability and method |
| `Applier` | Applies GPU requirements to service environment |

## Detector Interface

```go
type Detector interface {
    // Detect checks for GPU availability
    Detect() Status
}

type Status struct {
    Available bool   // GPU is available
    Method    string // Detection method used
    Reason    string // Explanation (especially for failures)
}
```

## Detection Methods

| Platform | Method | Tool |
|----------|--------|------|
| Linux | NVIDIA | `nvidia-smi` |
| macOS | Metal | `system_profiler SPDisplaysDataType` |
| Windows | DirectX | `wmic path win32_VideoController` |

## Environment Override

For testing or forcing specific behavior:

```bash
export BUNDLE_GPU_OVERRIDE=true   # Force GPU available
export BUNDLE_GPU_OVERRIDE=false  # Force GPU unavailable
```

## GPU Requirements

Services declare GPU requirements in the manifest:

| Requirement | Behavior |
|-------------|----------|
| `required` | Error if no GPU detected |
| `optional_with_cpu_fallback` | Use GPU if available, CPU otherwise (silent) |
| `optional_but_warn` | Use GPU if available, CPU otherwise (logged) |

## Environment Variables Set

When applying GPU requirements, these variables are injected:

| Variable | Description |
|----------|-------------|
| `BUNDLE_GPU_AVAILABLE` | `"true"` or `"false"` |
| `BUNDLE_GPU_MODE` | `"gpu"` or `"cpu"` |

## Usage

```go
// Detection
detector := gpu.NewDetector(cmdRunner, envReader)
status := detector.Detect()

if status.Available {
    fmt.Printf("GPU available (detected via %s)\n", status.Method)
} else {
    fmt.Printf("No GPU: %s\n", status.Reason)
}

// Apply to service
applier := gpu.NewApplier(status, telemetry)
err := applier.Apply(envMap, service)
```

## Manifest Example

```json
{
  "services": [{
    "id": "ml-worker",
    "gpu_requirement": "optional_with_cpu_fallback"
  }]
}
```

## Dependencies

- **Depends on**: `infra` (CommandRunner, EnvReader), `manifest`, `telemetry`
- **Depended on by**: `bundleruntime`
