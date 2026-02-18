# Tunnel Manager Requirements

Technical requirements registry for the Tunnel Manager scenario.

## Modules

| # | Module | PRD Refs | Description |
|---|--------|----------|-------------|
| 01 | Route Manifest | OT-P0-001 | Declarative route manifest data model and management |
| 02 | Port Compliance | OT-P0-002 | Scenario `service.json` port auditing |
| 03 | Tunnel Health | OT-P0-003, OT-P1-004, OT-P1-009 | cloudflared health monitoring and metrics |
| 04 | Liveness Probes | OT-P0-004, OT-P0-005, OT-P1-008 | Internal and external route probing |
| 05 | Auto-Recovery | OT-P0-006, OT-P1-007 | Automatic tunnel recovery with backoff |
| 06 | CLI Interface | OT-P0-007..010 | CLI commands |
| 07 | Cloudflare API | OT-P1-001, OT-P1-003, OT-P1-006 | Remote tunnel configuration management |
| 08 | Local Config | OT-P1-002, OT-P1-003, OT-P1-006 | Local config YAML management |
| 09 | Web Dashboard | OT-P1-005 | React UI for route status and tunnel health |
| 10 | Observability | OT-P1-004, OT-P1-007, OT-P1-009 | Metrics, logging, event tracking |
