# Decisions — Browser Automation Studio

This document records durable decisions and tradeoffs future agents should not
accidentally relitigate.

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-10 | Only the inner vision call moves to ai-gateway. | The navigation loop, element annotation, loop detection, and human intervention are caller-owned behavior. | browser-automation-studio sends ordered turns and attachments to ai-gateway; it retains the agent loop and browser-specific interpretation. | Revisit if ai-gateway adds a reviewed agent/tool contract that can replace caller-owned navigation behavior without losing browser semantics. |
| 2026-08-10 | The legacy Claude computer-use client is an explicit conformance exception during this migration. | ai-gateway does not yet carry the reviewed tool definitions and coordinate contract required by Claude computer-use, so the client remains temporarily isolated while all standard vision routes use AI Gateway. | The exception is narrow, documented, and must not expand to new provider paths; the Claude Code navigator resolves through resource-claude-code. | Revisit when ai-gateway ships the Claude computer-use request contract with tool definitions, coordinate normalization, and evidence semantics. |

`ai-gateway-exception owner=browser-automation-studio reason=Claude computer-use tool contract is not yet represented by the gateway action schema expires=2026-12-31 replacement=ai-gateway-computer-use-contract`

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |
