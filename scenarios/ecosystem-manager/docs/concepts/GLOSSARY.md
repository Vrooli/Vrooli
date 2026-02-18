# Glossary

## Task
A queue item representing scenario/resource generation or improvement.

## Operation
The work mode for a task: `generator` or `improver`.

## Type
The target class: `resource` or `scenario`.

## Auto Steer Term
Multi-phase steering profile used to evaluate and advance execution.

## Queue Processor
Background worker that picks pending items and executes them.

## Active Target
Resolved scenario/resource context used by execution and prompt assembly.

[CODE: api/pkg/tasks/types.go]
[CODE: api/pkg/queue/processor.go]
[CODE: api/pkg/autosteer/types.go]
