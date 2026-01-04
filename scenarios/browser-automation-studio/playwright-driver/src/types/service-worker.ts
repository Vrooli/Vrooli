/**
 * Service Worker Control Types
 *
 * Configures how service workers are managed during browser sessions.
 * Used to prevent redirect loops and compatibility issues with sites
 * that use service workers aggressively (e.g., Google).
 */

/**
 * Service worker control mode.
 * - 'allow': Allow SWs (default), with monitoring and manual controls
 * - 'block': Block all SW registrations session-wide
 * - 'block-on-domain': Block SWs only for domains in blockedDomains list
 * - 'unregister-all': Unregister all existing SWs when session starts
 */
export type ServiceWorkerMode = 'allow' | 'block' | 'block-on-domain' | 'unregister-all';

/**
 * Per-domain service worker override configuration.
 */
export interface ServiceWorkerDomainOverride {
  /** Domain pattern (e.g., "google.com", "*.google.com") */
  domain: string;
  /** Mode to apply for this domain */
  mode: 'allow' | 'block';
}

/**
 * Service worker control settings for a session.
 */
export interface ServiceWorkerControl {
  /** Session-wide default mode */
  mode: ServiceWorkerMode;
  /** Per-domain overrides (take precedence over mode) */
  domainOverrides?: ServiceWorkerDomainOverride[];
  /** Domains to block when mode is 'block-on-domain' */
  blockedDomains?: string[];
  /** Whether to unregister existing SWs on session start */
  unregisterOnStart?: boolean;
}

/**
 * Runtime information about a registered service worker.
 */
export interface ServiceWorkerInfo {
  /** Registration scope URL */
  scopeURL: string;
  /** Script URL */
  scriptURL: string;
  /** Registration ID from CDP */
  registrationId: string;
  /** Current running status */
  status: 'stopped' | 'running' | 'activating' | 'installed';
  /** Version ID */
  versionId?: string;
}

/**
 * Service worker list response for API.
 */
export interface ServiceWorkerListResponse {
  sessionId: string;
  workers: ServiceWorkerInfo[];
  control: ServiceWorkerControl;
}
