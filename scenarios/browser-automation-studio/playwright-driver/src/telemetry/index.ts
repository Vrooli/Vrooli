/**
 * Telemetry Module
 *
 * STABILITY: VOLATILE EDGE
 *
 * Captures observability data during instruction execution:
 * - Screenshot: Full-page or viewport captures with caching
 * - DOM: HTML snapshots for debugging and AI analysis
 * - Console/Network Collectors: Capture logs and requests
 *
 * Telemetry is attached to StepOutcome for debugging and
 * provides context for AI-assisted error analysis.
 *
 * CHANGE AXIS: New Telemetry Types
 * Primary extension point: Create new collector file
 *
 * When adding a new telemetry type:
 * 1. Create collector file (e.g., `telemetry/performance.ts`)
 * 2. Define type in `types/contracts.ts` (coordinate with Go API)
 * 3. Add to StepOutcome interface if needed
 * 4. Update `outcome/outcome-builder.ts` to include in output
 * 5. Add config options in `config.ts` under telemetry section
 * 6. Use collector in `routes/session-run.ts`
 *
 * Current telemetry types: screenshot, dom, console, network
 * Planned: HAR (enabled via config), video, tracing
 */

export * from './collector';
export * from './screenshot';
export * from './dom';
