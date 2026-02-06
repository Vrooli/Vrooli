# Glossary

Key terms and concepts used in Scenario-to-Cloud.

## Bundle

A self-contained package containing a minimal Vrooli installation plus your scenario files. Bundles are created locally and transferred to the VPS via SCP.

## Caddy

An automatic HTTPS server. Scenario-to-Cloud uses Caddy to provide SSL certificates via Let's Encrypt and reverse proxy to your scenario services.

## Deployment

A tracked instance of a scenario running on a VPS. Deployments have status, history, and can be inspected, stopped, or redeployed.

## Drift Detection

Comparing the expected state of a deployment (from its manifest and last-known configuration) against the actual live state on the VPS. Drift indicates that something changed on the server outside of the normal deployment pipeline.

## Edge

The public-facing configuration including domain and HTTPS settings. The "edge" is what users see when accessing your deployed scenario.

## Investigation

An autonomous debugging session run by the agent-manager against a deployment. Investigations analyze logs, inspect live state, and propose fixes. See also: Task.

## Live State

The real-time state of a deployment's VPS, gathered by running ~15 SSH commands in parallel. Includes processes, ports, disk, memory, CPU, uptime, scenario status, resource status, and Caddy configuration.

## Manifest

A JSON configuration file that describes what to deploy and where. The manifest includes scenario ID, VPS connection details, port mappings, and dependencies.

## Mini-Vrooli

A stripped-down Vrooli installation optimized for production deployments. Contains only the scripts and resources needed to run your scenario.

## Preflight

Pre-deployment checks that verify your VPS is ready. Includes SSH connectivity, disk space, and tool availability.

## Preflight Fix

An automated remediation action that resolves a preflight check failure. Examples: stopping services occupying required ports, opening firewall rules, cleaning up disk space, stopping stale scenario processes.

## Resource

A shared service that scenarios depend on, such as PostgreSQL, Redis, or Ollama. Resources are managed by Vrooli and can be shared across scenarios.

## Scenario

A complete application built on Vrooli. Scenarios can include APIs, UIs, CLIs, and depend on resources and other scenarios.

## SSE (Server-Sent Events)

A protocol for streaming real-time updates from the server to the browser over HTTP. Used by the progress endpoint (`GET /deployments/{id}/progress`) to push deployment step updates.

## StepConfig

Per-step execution parameters for VPS deployment operations. Controls command timeout, retry count, and retry delay for each pipeline step. See [configuration.md](../reference/configuration.md#step-configuration).

## TOFU (Trust On First Use)

SSH host key verification model used by scenario-to-cloud. On first connection to a new host, the key is accepted and saved. On subsequent connections, a changed key triggers a warning. Controlled by the `StrictHostKey` option.

## Tool Execution Protocol

A standardized interface for other agents and scenarios to programmatically invoke scenario-to-cloud capabilities. Tools are discovered via `GET /tools` and executed via `POST /tools/execute`.

## VPS (Virtual Private Server)

A remote server where your scenario is deployed. Scenario-to-Cloud supports any Linux VPS with SSH access.
