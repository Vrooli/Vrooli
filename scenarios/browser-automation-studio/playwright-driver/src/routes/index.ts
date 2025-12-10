/**
 * Routes Module
 *
 * STABILITY: STABLE CORE (router) + VOLATILE EDGE (route handlers)
 *
 * HTTP transport layer - maps URLs to handlers:
 * - /health: Health check endpoint
 * - /session/start: Create new browser session
 * - /session/:id/run: Execute instruction (instruction handlers are volatile)
 * - /session/:id/reset: Reset session state
 * - /session/:id/close: Terminate session
 * - /session/:id/storage-state: Export session storage
 * - /session/:id/record/*: Record mode endpoints
 *
 * Routes orchestrate the flow between transport, session management,
 * handler execution, and outcome building.
 *
 * CHANGE AXIS: New HTTP Endpoints
 * Primary extension point: Create new route file
 *
 * When adding a new endpoint:
 * 1. Create route handler file (e.g., `routes/session-foo.ts`)
 * 2. Export handler from this index file
 * 3. Register route in `server.ts` with router.get/post()
 * 4. Add any request/response types to `types/session.ts`
 *
 * Core routes (session lifecycle) are stable.
 * Feature routes (record-mode) are expected to evolve.
 */

export * from './router';
export type { Router } from './router';
export * from './health';
export * from './session-start';
export * from './session-run';
export * from './session-reset';
export * from './session-close';
export * from './session-storage';
export * from './record-mode/index';
