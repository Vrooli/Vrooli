# Known Issues & Deferred Work

## Open Issues

- **prd-control-tower requirements generation**: The tool experienced EOF errors during initial requirements generation. The requirements were eventually generated but this may recur. Workaround: Create requirements manually if the tool fails.

## Deferred Ideas

- **Conflict detection sophistication**: Simple path-based overlap detection may miss semantic conflicts (e.g., two skills expecting different patterns in the same file). More sophisticated analysis (AST-based, content comparison) is deferred to P1 conflict detection work.

- **Auto-config suggestions (P2)**: Analyzing SKILL.md content to suggest structural expectations requires NLP/pattern matching against free-form markdown. This is a significant effort deferred to P2.

- **Multi-template references**: Currently only react-vite template is planned. Adding CLI-only or landing-page templates requires defining which steer skills apply to each template type.

- **Concurrent CLI execution**: Running multiple CLI tool assertions sequentially can be slow. Parallel execution with goroutines is possible but needs careful resource management (port conflicts, CPU/memory).

## Tech Debt

- None yet (new scenario).
