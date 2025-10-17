# Workflow Usage

Default to direct API or CLI calls to interact with other resources and scenarios; workflow engines (e.g. n8n) add latency and extra failure modes without delivering additional value for most scenarios.

## When Workflows Make Sense
Use n8n/windmill-style workflows only when the scenario's core purpose is workflow creation or management. All other integrations should expose APIs or CLIs that other scenarios can call directly.

## Migration Reminder
If you inherit a workflow-based integration, plan to replace it with API/CLI calls: document the existing flow, build the equivalent endpoint, switch callers, then remove the workflow configuration.
