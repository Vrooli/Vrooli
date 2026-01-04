/**
 * Service Worker Module
 *
 * Provides configurable service worker control for browser automation.
 * Used to prevent redirect loops and compatibility issues with sites
 * that use service workers aggressively (e.g., Google).
 *
 * Features:
 * - CDP-based monitoring of service worker registrations
 * - Unregister specific or all service workers
 * - Block SW registration per-domain via script injection
 * - Domain pattern matching with wildcard support
 */

export { ServiceWorkerController } from './controller';

// Re-export types for convenience
export type {
  ServiceWorkerMode,
  ServiceWorkerControl,
  ServiceWorkerDomainOverride,
  ServiceWorkerInfo,
  ServiceWorkerListResponse,
} from '../types/service-worker';
