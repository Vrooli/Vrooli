/**
 * Execution feature - DEPRECATED LOCATION
 * Re-exports from domains/executions and domains/exports for backwards compatibility
 */

// Re-export executions domain
export * from '@/domains/executions';

// Re-export ReplayPlayer from exports domain for backwards compatibility
export { default as ReplayPlayer } from '@/domains/exports/replay/ReplayPlayer';
