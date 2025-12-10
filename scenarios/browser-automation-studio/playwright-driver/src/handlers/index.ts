/**
 * Handlers Module
 *
 * STABILITY: VOLATILE (by design)
 *
 * CHANGE AXIS: Adding New Instruction Types
 *
 * This module is the PRIMARY extension point for browser automation capabilities.
 * Instruction handlers execute browser automation commands via Playwright.
 *
 * To add a new instruction type:
 * 1. Create a new handler file (e.g., `handlers/my-action.ts`)
 * 2. Implement InstructionHandler interface (see base.ts for contract)
 * 3. Add Zod schema to `types/instruction.ts` for parameter validation
 * 4. Export handler from this index file
 * 5. Register handler in `server.ts:registerHandlers()`
 *
 * Handlers:
 * - Parse and validate instruction parameters (use Zod schemas)
 * - Execute Playwright actions
 * - Return HandlerResult (success/failure + extracted data)
 *
 * The registry provides handler lookup by instruction type.
 *
 * STABLE DEPENDENCIES:
 * - base.ts: Handler interface (InstructionHandler, HandlerContext, HandlerResult)
 * - registry.ts: Handler registration and lookup
 *
 * These interfaces are stable - handlers can evolve independently.
 */

// Core abstractions
export * from './base';
export * from './registry';

// Navigation & Frames
export * from './navigation';
export * from './frame';
export * from './tab';

// Interaction
export * from './interaction';
export * from './gesture';
export * from './keyboard';
export * from './select';

// Wait & Assert
export * from './wait';
export * from './assertion';

// Data
export * from './extraction';
export * from './screenshot';
export * from './cookie-storage';

// IO
export * from './upload';
export * from './download';
export * from './scroll';

// Network & Device
export * from './network';
export * from './device';
