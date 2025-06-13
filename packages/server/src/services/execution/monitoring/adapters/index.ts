/**
 * Emergent Monitoring Architecture
 * 
 * Instead of hardcoded monitoring adapters, this system uses emergent agents
 * that learn from events and propose monitoring improvements.
 * 
 * See ../README.md for how to deploy intelligent monitoring agents.
 */

// Only keep the telemetry shim for external compatibility
export { TelemetryShimAdapter } from "./TelemetryShimAdapter.js";

// For emergent monitoring, use the AgentDeploymentService:
// import { AgentDeploymentService } from "../../cross-cutting/agents/agentDeploymentService.js";