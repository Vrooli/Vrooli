/**
 * AI Module
 *
 * STABILITY: STABLE CONTRACT
 *
 * This module provides AI-driven browser navigation capabilities.
 * It exports all sub-modules for the vision agent system.
 *
 * Architecture:
 * - action: Action types, parsing, and execution
 * - vision-client: Vision model client interface and implementations
 * - vision-agent: Main navigation orchestrator
 * - screenshot: Screenshot capture and annotation
 * - emitter: Event emission to API
 */

// Action module
export * from './action';

// Vision client module
export * from './vision-client';

// Vision agent module
export * from './vision-agent';

// Screenshot module
export * from './screenshot';

// Emitter module
export * from './emitter';
