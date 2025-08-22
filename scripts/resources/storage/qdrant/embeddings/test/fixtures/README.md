# Embedding System Test Fixtures

This directory contains test fixtures for the Qdrant Semantic Knowledge System.

## Structure

```
fixtures/
├── sample-apps/           # Complete sample apps for end-to-end testing
│   ├── test-app-1/        # Simple app with basic content
│   ├── test-app-2/        # Complex app with all content types
│   └── empty-app/         # App with minimal content
├── workflows/             # Sample N8n workflow files
├── scenarios/             # Sample PRDs and scenario configs
├── docs/                  # Sample documentation with embedding markers
├── code/                  # Sample code files (bash, ts, js, etc.)
├── resources/             # Sample resource configurations
└── expected/              # Expected extraction outputs for validation
```

## Usage

```bash
# Run individual extractor tests
bats test/extractors/*.bats

# Run full integration tests  
bats test/integration/*.bats

# Run performance benchmarks
bash test/performance/benchmark.sh
```

## Test Data Guidelines

- All fixtures are **synthetic** - no real sensitive data
- Content designed to test edge cases and typical patterns
- Expected outputs pre-calculated for validation
- Covers all supported file types and formats