/**
 * 🧪 Execution Architecture Test Fixtures
 * 
 * Comprehensive test fixtures for Vrooli's three-tier AI execution system,
 * demonstrating emergent capabilities, agent collaboration, and self-improving intelligence.
 * 
 * Directory Structure:
 * - tier1-coordination/: Dynamic swarm coordination through AI metacognition
 * - tier2-process/: Universal workflow execution supporting multiple formats
 * - tier3-execution/: Context-aware strategy execution with safety enforcement
 * - emergent-capabilities/: Cross-tier emergent behaviors and self-improvement
 * - integration-scenarios/: Complete system examples showing tier synergy
 * 
 * For detailed documentation, see:
 * - README.md in this directory
 * - docs/architecture/execution/ for architecture documentation
 */

// Tier 2: Process Intelligence
export * from "./tier2-process/index.js";

// Emergent Capabilities
export * from "./emergent-capabilities/index.js";

// Shared utilities
export * from "./testIdGenerator.js";

// Get all routine fixtures by evolution stage
import { getRoutinesByEvolutionStage } from "./tier2-process/index.js";

export const routinesByStage = getRoutinesByEvolutionStage();

// Get evolution metrics
import { evolutionMetrics } from "./emergent-capabilities/index.js";

export { evolutionMetrics };
