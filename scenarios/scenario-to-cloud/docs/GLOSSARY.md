# Glossary

Key terms and concepts used in Scenario-to-Cloud.

## Bundle

A self-contained package containing a minimal Vrooli installation plus your scenario files. Bundles are created locally and transferred to the VPS via SCP.

## Caddy

An automatic HTTPS server. Scenario-to-Cloud uses Caddy to provide SSL certificates via Let's Encrypt and reverse proxy to your scenario services.

## Deployment

A tracked instance of a scenario running on a VPS. Deployments have status, history, and can be inspected, stopped, or redeployed.

## Edge

The public-facing configuration including domain and HTTPS settings. The "edge" is what users see when accessing your deployed scenario.

## Manifest

A JSON configuration file that describes what to deploy and where. The manifest includes scenario ID, VPS connection details, port mappings, and dependencies.

## Mini-Vrooli

A stripped-down Vrooli installation optimized for production deployments. Contains only the scripts and resources needed to run your scenario.

## Preflight

Pre-deployment checks that verify your VPS is ready. Includes SSH connectivity, disk space, and tool availability.

## Resource

A shared service that scenarios depend on, such as PostgreSQL, Redis, or Ollama. Resources are managed by Vrooli and can be shared across scenarios.

## Scenario

A complete application built on Vrooli. Scenarios can include APIs, UIs, CLIs, and depend on resources and other scenarios.

## VPS (Virtual Private Server)

A remote server where your scenario is deployed. Scenario-to-Cloud supports any Linux VPS with SSH access.
