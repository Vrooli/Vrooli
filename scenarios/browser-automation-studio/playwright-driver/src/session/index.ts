/**
 * Session Module
 *
 * STABILITY: STABLE CORE
 *
 * Manages browser session lifecycle:
 * - SessionManager: Create, retrieve, reset, close sessions
 * - SessionCleanup: Reap idle sessions automatically
 * - ContextBuilder: Configure Playwright BrowserContext
 *
 * A session encapsulates a Browser + BrowserContext + Page with
 * tracking for activity, phase, and capabilities (recording, tabs, etc.).
 *
 * CHANGE AXIS: Session Capabilities
 *
 * When adding new session capabilities (e.g., new browser features):
 * 1. Add capability flag to SessionInfo in `types/session.ts`
 * 2. Add configuration option in `config.ts`
 * 3. Update ContextBuilder to apply capability settings
 * 4. Update /session/start handler if user-configurable
 *
 * Core lifecycle operations (create/reset/close) are stable.
 */

export * from './manager';
export * from './cleanup';
export * from './context-builder';
