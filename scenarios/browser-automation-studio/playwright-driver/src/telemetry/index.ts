/**
 * Telemetry Module
 *
 * STABILITY: VOLATILE EDGE
 *
 * Captures observability data during instruction execution:
 * - Screenshot: Full-page or viewport captures with caching
 * - DOM: HTML snapshots for debugging and AI analysis
 * - Console/Network Collectors: Capture logs and requests
 * - Orchestrator: Unified telemetry collection service
 *
 * Telemetry is attached to StepOutcome for debugging and
 * provides context for AI-assisted error analysis.
 *
 * CHANGE AXIS: New Telemetry Types
 * Primary extension point: Create new collector file
 *
 * When adding a new telemetry type:
 * 1. Create collector file (e.g., `telemetry/performance.ts`)
 * 2. Define type in proto schema (driver.proto) and regenerate
 * 3. Add to StepOutcome interface if needed
 * 4. Update `outcome/outcome-builder.ts` to include in output
 * 5. Add config options in `config.ts` under telemetry section
 * 6. Add to TelemetryOrchestrator if needed
 *
 * Current telemetry types: screenshot, dom, console, network
 * Planned: HAR (enabled via config), video, tracing
 */

export * from './collector';
export * from './screenshot';
export * from './dom';
export * from './orchestrator';
export * from './element-context';
