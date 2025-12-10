/**
 * Types Module
 *
 * Central export for type definitions and contracts.
 *
 * STABILITY GUIDE:
 *
 * STABLE CONTRACT (contracts.ts):
 *   - Wire format between Playwright driver and Go API
 *   - Changes require coordinated updates to Go contracts package
 *   - Adding fields is safe (Go uses omitempty)
 *   - Removing/renaming fields is a BREAKING CHANGE
 *
 * STABLE CORE (session.ts):
 *   - Session lifecycle types
 *   - Only additive changes expected
 *
 * VOLATILE (instruction.ts):
 *   - Zod schemas for instruction parameters
 *   - CHANGE AXIS: Adding new instruction types
 *   - Each schema is independent, safe to add new ones
 */

export * from './contracts';
export * from './session';
export * from './instruction';
