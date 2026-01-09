# Model Context Protocol Overview

The Model Context Protocol (MCP) is Vrooli's standard for describing agent-facing capabilities in a way that is portable across scenarios. Each MCP endpoint exposes a manifest that lists supported tools, metadata that agents use when deciding how to engage, and the network transport that should be used for live sessions. By aligning on MCP, scenarios can share intelligence without copy-pasting integrations or re-implementing orchestration logic.

## Why MCP Exists

- **Repeatable onboarding** – once a scenario knows how to publish an MCP manifest it becomes instantly consumable by any compliant agent.
- **Discoverable capabilities** – the registry surfaces scenario functionality, making it easy to locate the right tool for a job.
- **Composable systems** – agents can chain MCP endpoints together because they all speak the same protocol surface area.

## Key Resources

- [OpenAI – Model Context Protocol](https://github.com/modelcontextprotocol/specification)
- [Vrooli Scenario Registry](../README.md)
- [Scenario PRD](../PRD.md)

The remainder of this documentation explains how the Scenario to MCP application automates discovery, validation, and publishing for these endpoints.
