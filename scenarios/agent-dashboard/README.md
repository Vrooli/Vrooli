# Agent Dashboard

## Purpose
Dashboard for monitoring and orchestrating all active AI agents in the Vrooli ecosystem, including Claude Code, Agent-S2, Huginn, n8n workers, and custom agents.

## Usefulness
- **Central Control**: Single interface to monitor all agents' health and performance
- **Smart Orchestration**: AI-powered decision making for optimal agent configuration
- **Resource Management**: Efficient allocation of compute resources across agents
- **Proactive Monitoring**: Automated health checks and remediation

## Key Features
- Real-time agent status monitoring
- Health checks with auto-remediation
- Multi-agent orchestration
- Performance metrics tracking
- Resource usage optimization

## Dependencies
- **Resources**: None (monitors existing resources)
- **Models**: llama3.2:3b for intelligent orchestration (optional)
- **Shared Workflows**: ollama.json for AI reasoning (optional)

## UX Style
Professional, dark-themed dashboard with real-time updates. Matrix-inspired green-on-black terminal aesthetic for system monitoring views.

## Integration Points
Other scenarios can query agent status and request orchestration via CLI:
```bash
vrooli scenario agent-dashboard status
vrooli scenario agent-dashboard orchestrate --task "complex multi-agent task"
```