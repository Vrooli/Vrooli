# Implementation Guide

This guide walks through how Scenario to MCP assembles, validates, and ships MCP endpoints.

## Discovery Pipeline

1. **Detector scan** – `lib/detector.js` inspects each scenario, infers whether MCP integration exists, and reports capabilities back to the API.
2. **Database sync** – active endpoints are persisted in `mcp.endpoints` so the registry and UI stay in sync across restarts.
3. **Health checks** – lifecycle hooks update `last_health_check` after each validation run.

## Adding a New MCP Endpoint

- Trigger the "Add MCP" flow from the dashboard.
- Provide the target scenario and allow auto-detection to scaffold the manifest.
- Once generation completes, restart the scenario so the registry can poll the new endpoint.

## Publishing Documentation

The UI exposes content from `docs/` alongside key scenario artifacts (`README.md`, `PRD.md`, `PROBLEMS.md`, and `TEST_IMPLEMENTATION_SUMMARY.md`). Use this folder for:

- Playbooks that explain end-to-end workflows.
- Troubleshooting and verification notes.
- Links to upstream MCP specifications or client libraries.

Remember to keep documents in plain Markdown so they render correctly in the built-in viewer.
