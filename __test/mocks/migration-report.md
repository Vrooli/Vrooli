# Mock Migration Report
Generated: Sun Aug 24 02:30:46 AM EDT 2025

## Executive Summary
- Tier 2 mocks: 24
- Legacy mocks: 28
- Migration progress: 85%

## Mock Status

| Mock | Tier 2 | Legacy | Lines Saved | Reduction |
|------|--------|--------|-------------|-----------|
| agent-s2 | ✅ | ✅ | 421 | 41% |
| browserless | ✅ | ✅ | 442 | 47% |
| claude-code | ✅ | ✅ | 327 | 41% |
| comfyui | ✅ | ✅ | 6 | 1% |
| dig | ❌ | ✅ | - | - |
| docker | ✅ | ✅ | 386 | 35% |
| filesystem | ✅ | ✅ | 1004 | 60% |
| helm | ✅ | ✅ | 426 | 36% |
| http | ✅ | ✅ | 732 | 59% |
| huginn | ✅ | ✅ | 312 | 42% |
| jq | ❌ | ✅ | - | - |
| judge0 | ✅ | ✅ | 96 | 19% |
| logs | ❌ | ✅ | - | - |
| minio | ✅ | ✅ | 429 | 39% |
| n8n | ✅ | ✅ | 464 | 44% |
| node-red | ✅ | ✅ | 776 | 67% |
| ollama | ✅ | ✅ | 586 | 52% |
| postgres | ✅ | ✅ | 606 | 53% |
| qdrant | ✅ | ✅ | 735 | 61% |
| questdb | ✅ | ✅ | 343 | 42% |
| redis | ✅ | ✅ | 834 | 59% |
| searxng | ✅ | ✅ | 472 | 55% |
| system | ✅ | ✅ | 1492 | 68% |
| unstructured-io | ✅ | ✅ | 481 | 50% |
| vault | ✅ | ✅ | 131 | 12% |
| verification | ❌ | ✅ | - | - |
| whisper | ✅ | ✅ | 809 | 55% |
| windmill | ✅ | ✅ | 341 | 37% |

## Next Steps

1. Complete migration of remaining legacy-only mocks
2. Update test files to use Tier 2 mocks
3. Remove legacy mocks after verification
4. Update documentation
