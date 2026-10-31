# Decisions

## Direct Weather And Prediction Flows

The PRD now says weather integration is handled through direct API/CLI flows, while Ollama prediction uses direct API calls. Do not reintroduce n8n workflow requirements without a new product decision.

## Lifecycle Ownership

Scenario start, stop, setup, and test behavior is owned by Vrooli lifecycle configuration. The Makefile remains a thin wrapper over lifecycle commands.

## Fallback Data

Read-only discovery endpoints may return sample data when PostgreSQL is unavailable. This supports degraded local exploration without making writes appear successful when persistence is unavailable.
