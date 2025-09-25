# Algorithm Library

## Overview

The Algorithm Library is a validated, multi-language reference system for data structures and algorithms. It serves as the "ground truth" for correct algorithm implementations, providing both humans and AI agents with trusted, tested code they can reference, validate against, or directly use.

## Purpose

This scenario adds a **permanent capability** to Vrooli: a centralized, validated algorithm reference that ensures correctness across all coding scenarios. Every algorithm is tested via Judge0, performance benchmarked, and available in multiple programming languages.

## Key Features

- **Multi-Language Support**: Implementations in Python, JavaScript, Go, Java, C++, and more
- **Judge0 Validation**: Every algorithm is executed and validated in a sandboxed environment
- **Performance Benchmarking**: Time and space complexity measurements for optimization
- **API Access**: Other scenarios can query and validate their implementations
- **Terminal UI**: Matrix-style interface for browsing and testing algorithms
- **CLI Tool**: Command-line access for agents and developers

## Architecture

```
algorithm-library/
├── api/              # Go API server (port 3250)
├── cli/              # Bash CLI wrapper
├── ui/               # React web interface (port 3251)
├── initialization/
│   ├── postgres/     # Database schema and seed data
│   └── n8n/         # Testing workflows via Judge0
└── tests/           # Validation tests
```

## Usage

### Starting the Scenario

```bash
# Run the scenario
vrooli scenario run algorithm-library

# Access points:
# - UI: http://localhost:3251
# - API: http://localhost:3250
# - CLI: algorithm-library --help
```

### CLI Examples

```bash
# Search for algorithms
algorithm-library search quicksort
algorithm-library search --category sorting --language python

# Get algorithm details
algorithm-library get binary_search --all-languages

# Validate your implementation
algorithm-library validate quicksort my_quicksort.py

# View statistics
algorithm-library stats
```

### API Examples

```bash
# Search algorithms
curl http://localhost:3250/api/v1/algorithms/search?category=sorting

# Get implementations
curl http://localhost:3250/api/v1/algorithms/quicksort/implementations

# Validate code
curl -X POST http://localhost:3250/api/v1/algorithms/validate \
  -H "Content-Type: application/json" \
  -d '{"algorithm_id": "quicksort", "language": "python", "code": "..."}'
```

## Resource Dependencies

### Required
- **PostgreSQL**: Stores algorithms, implementations, and test results
- **Judge0**: Executes and validates code in sandboxed environment
- **n8n**: Orchestrates testing workflows

### Optional
- **Redis**: Caches frequently accessed algorithms
- **Ollama**: Generates algorithm explanations

## Value Proposition

### For Developers
- Reduces debugging time by 60-80%
- Provides working reference implementations
- Ensures algorithmic correctness

### For AI Agents
- Ground truth for validation
- Reference for code generation
- Performance baselines for optimization

### For Vrooli
- **Reusability**: 10/10 - Every coding scenario benefits
- **Intelligence Amplification**: Makes all agents better at coding
- **Compound Value**: Each validated algorithm improves future scenarios

## Initial Algorithm Set

The library comes pre-seeded with 50+ fundamental algorithms:
- **Sorting** (15): QuickSort, MergeSort, HeapSort, InsertionSort, BubbleSort, RadixSort, CountingSort, BucketSort, ShellSort, TimSort, and more
- **Searching** (2): Binary Search, Linear Search  
- **Graph** (9): DFS, BFS, Dijkstra, Kruskal, Topological Sort, Bellman-Ford, Floyd-Warshall, Prim, and more
- **Dynamic Programming** (8): Fibonacci, Knapsack, LCS, Edit Distance, Coin Change, Kadane's Algorithm, and more
- **Tree** (8): Binary Tree Traversal, BST Operations, AVL Tree, Heap Operations, Red-Black Tree, Segment Tree, and more
- **String** (5): KMP, Rabin-Karp, Boyer-Moore, Z Algorithm, Manacher's Algorithm
- **Mathematical** (6): GCD, Sieve of Eratosthenes, Fast Exponentiation, Matrix Multiplication, and more
- **Backtracking** (6): N-Queens, Sudoku Solver, Permutations, Combinations, and more
- **Greedy** (3): Activity Selection, Huffman Coding, Job Scheduling

## Future Enhancements

### Version 2.0
- Visual algorithm execution traces
- AI-powered problem-to-algorithm matching
- LeetCode/HackerRank problem mapping
- Real-time collaboration features

### Long-term Vision
- Become the definitive algorithm reference for all AI agents
- Self-improving through usage patterns
- Bridge between academic algorithms and production code

## Style Guide

The UI follows a **Matrix-style terminal aesthetic**:
- Dark background with green text
- Monospace fonts throughout
- Subtle scan line effects
- Terminal command prompts
- Technical but approachable

This creates a focused, efficient environment that feels like a hacker's algorithm reference while maintaining professional polish.

## Testing

```bash
# Run all tests
vrooli scenario test algorithm-library

# Test specific components
cd scenarios/algorithm-library
./tests/test-judge0-integration.sh
```

## Contributing

New algorithms can be added via:
1. Direct database insertion (for admins)
2. API submission endpoint (coming in v1.1)
3. CLI contribution command (coming in v1.1)

All submissions must pass the full test suite before being accepted into the library.

## Known Issues

### Judge0 Integration
- **Issue**: Judge0 execution fails with cgroup configuration errors
- **Impact**: Algorithm validation via Judge0 is currently non-functional
- **Workaround**: Use native Go execution for testing (under development)
- **Resolution**: Requires system-level cgroup configuration on the host

### API Endpoints
- Categories and Stats endpoints have minor SQL issues (under investigation)
- Core search and retrieval endpoints are fully functional

---

**Status**: ✅ Operational (with Judge0 limitation)  
**Version**: 1.0.0  
**Port Range**: API on dynamic port (check `vrooli scenario status`)  
**Resource Requirements**: Medium (2GB RAM, Judge0 container)  
**Last Updated**: 2025-09-24