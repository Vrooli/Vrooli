# Performance

## Targets

The PRD targets:

- API data queries under 500 ms.
- Initial UI load under 3 s.
- Prediction generation under 5 s per region.
- Map rendering under 2 s for overlay updates.

## Tests

Performance checks live in [CODE: api/performance_test.go] and cover response latency, concurrent requests, database pool behavior, throughput, and error handling overhead.

## Known Caveat

Prediction latency depends on Ollama availability and model response time. The fallback path keeps prediction responses bounded when Ollama is unavailable.
