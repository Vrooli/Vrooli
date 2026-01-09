/**
 * Service Worker Controller
 *
 * Low-level CDP-based service worker management.
 * Provides direct access to Chrome's ServiceWorker domain.
 *
 * CDP Methods Used:
 * - ServiceWorker.enable() - Start receiving SW events
 * - ServiceWorker.disable() - Stop receiving SW events
 * - ServiceWorker.unregister(scopeURL) - Unregister specific SW
 * - ServiceWorker.stopAllWorkers() - Stop all running workers
 *
 * CDP Events:
 * - ServiceWorker.workerRegistrationUpdated - Registration changes
 * - ServiceWorker.workerVersionUpdated - Version changes
 * - ServiceWorker.workerErrorReported - SW errors
 */

import type { CDPSession, BrowserContext, Page } from 'rebrowser-playwright';
import type {
  ServiceWorkerInfo,
  ServiceWorkerControl,
} from '../types/service-worker';
import { getCachedCDPSession } from '../session/cdp-session';
import { logger, scopedLog, LogContext } from '../utils';

/**
 * CDP ServiceWorker.workerRegistrationUpdated event payload
 */
interface CDPServiceWorkerRegistration {
  registrationId: string;
  scopeURL: string;
  isDeleted: boolean;
}

/**
 * CDP ServiceWorker.workerVersionUpdated event payload
 */
interface CDPServiceWorkerVersion {
  versionId: string;
  registrationId: string;
  scriptURL: string;
  runningStatus: 'stopped' | 'starting' | 'running' | 'stopping';
  status: 'new' | 'installing' | 'installed' | 'activating' | 'activated' | 'redundant';
}

/**
 * Controller for CDP-based service worker operations.
 *
 * Responsibilities:
 * - Enable/disable ServiceWorker domain monitoring
 * - Track registrations and versions via CDP events
 * - Unregister service workers by scope URL
 * - Inject blocking scripts for domain-based control
 */
export class ServiceWorkerController {
  private sessionId: string;
  private registrations: Map<string, CDPServiceWorkerRegistration> = new Map();
  private versions: Map<string, CDPServiceWorkerVersion> = new Map();
  private enabled = false;
  private cdpSession: CDPSession | null = null;
  private control: ServiceWorkerControl;

  constructor(sessionId: string, control: ServiceWorkerControl) {
    this.sessionId = sessionId;
    this.control = control;
  }

  /**
   * Initialize SW monitoring via CDP.
   */
  async enable(page: Page): Promise<void> {
    if (this.enabled) return;

    try {
      this.cdpSession = await getCachedCDPSession(page);

      // Enable ServiceWorker domain
      await this.cdpSession.send('ServiceWorker.enable');
      this.enabled = true;

      // Setup event listeners
      this.cdpSession.on('ServiceWorker.workerRegistrationUpdated', (event: { registrations: CDPServiceWorkerRegistration[] }) => {
        this.handleRegistrationUpdate(event.registrations);
      });

      this.cdpSession.on('ServiceWorker.workerVersionUpdated', (event: { versions: CDPServiceWorkerVersion[] }) => {
        this.handleVersionUpdate(event.versions);
      });

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      this.cdpSession.on('ServiceWorker.workerErrorReported', (event: any) => {
        logger.warn(scopedLog(LogContext.SESSION, 'service worker error'), {
          sessionId: this.sessionId,
          errorMessage: event.errorMessage?.message || event.errorMessage || 'unknown error',
        });
      });

      logger.debug(scopedLog(LogContext.SESSION, 'service worker monitoring enabled'), {
        sessionId: this.sessionId,
        mode: this.control.mode,
      });
    } catch (error) {
      logger.warn(scopedLog(LogContext.SESSION, 'failed to enable SW monitoring'), {
        sessionId: this.sessionId,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  /**
   * Disable SW monitoring.
   */
  async disable(): Promise<void> {
    if (!this.enabled || !this.cdpSession) return;

    try {
      await this.cdpSession.send('ServiceWorker.disable');
      this.enabled = false;
      this.registrations.clear();
      this.versions.clear();
      logger.debug(scopedLog(LogContext.SESSION, 'service worker monitoring disabled'), {
        sessionId: this.sessionId,
      });
    } catch {
      // CDP session may already be detached - this is fine
    }
  }

  /**
   * Unregister a specific service worker by scope URL.
   */
  async unregister(scopeURL: string): Promise<boolean> {
    if (!this.cdpSession) return false;

    try {
      await this.cdpSession.send('ServiceWorker.unregister', { scopeURL });
      logger.info(scopedLog(LogContext.SESSION, 'service worker unregistered'), {
        sessionId: this.sessionId,
        scopeURL,
      });
      return true;
    } catch (error) {
      logger.warn(scopedLog(LogContext.SESSION, 'failed to unregister SW'), {
        sessionId: this.sessionId,
        scopeURL,
        error: error instanceof Error ? error.message : String(error),
      });
      return false;
    }
  }

  /**
   * Unregister all service workers.
   */
  async unregisterAll(): Promise<number> {
    let count = 0;
    for (const reg of this.registrations.values()) {
      if (!reg.isDeleted) {
        const success = await this.unregister(reg.scopeURL);
        if (success) count++;
      }
    }
    if (count > 0) {
      logger.info(scopedLog(LogContext.SESSION, 'all service workers unregistered'), {
        sessionId: this.sessionId,
        count,
      });
    }
    return count;
  }

  /**
   * Stop all running service workers.
   */
  async stopAll(): Promise<void> {
    if (!this.cdpSession) return;

    try {
      await this.cdpSession.send('ServiceWorker.stopAllWorkers');
      logger.info(scopedLog(LogContext.SESSION, 'all service workers stopped'), {
        sessionId: this.sessionId,
      });
    } catch (error) {
      logger.warn(scopedLog(LogContext.SESSION, 'failed to stop all SWs'), {
        sessionId: this.sessionId,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  /**
   * Get list of all registered service workers.
   */
  getWorkers(): ServiceWorkerInfo[] {
    const workers: ServiceWorkerInfo[] = [];

    for (const reg of this.registrations.values()) {
      if (reg.isDeleted) continue;

      // Find version for this registration
      const version = [...this.versions.values()].find(
        (v) => v.registrationId === reg.registrationId
      );

      workers.push({
        registrationId: reg.registrationId,
        scopeURL: reg.scopeURL,
        scriptURL: version?.scriptURL || '',
        status: this.mapStatus(version?.runningStatus),
        versionId: version?.versionId,
      });
    }

    return workers;
  }

  /**
   * Get current control settings.
   */
  getControl(): ServiceWorkerControl {
    return this.control;
  }

  /**
   * Check if a domain should have SWs blocked.
   */
  shouldBlockDomain(domain: string): boolean {
    // Check domain overrides first (highest priority)
    if (this.control.domainOverrides) {
      for (const override of this.control.domainOverrides) {
        if (this.matchDomain(domain, override.domain)) {
          return override.mode === 'block';
        }
      }
    }

    // Check blockedDomains for block-on-domain mode
    if (this.control.mode === 'block-on-domain' && this.control.blockedDomains) {
      return this.control.blockedDomains.some((d) => this.matchDomain(domain, d));
    }

    // Session-wide mode
    return this.control.mode === 'block';
  }

  /**
   * Setup script injection to block SW registration for specific domains.
   */
  async setupBlockingForContext(context: BrowserContext): Promise<void> {
    if (this.control.mode === 'allow' && !this.control.domainOverrides?.length) {
      return; // Nothing to block
    }

    // Inject script that overrides navigator.serviceWorker.register
    const blockScript = this.generateBlockingScript();

    await context.addInitScript(blockScript);

    logger.debug(scopedLog(LogContext.SESSION, 'SW blocking script injected'), {
      sessionId: this.sessionId,
      mode: this.control.mode,
      blockedDomains: this.control.blockedDomains,
      overrideCount: this.control.domainOverrides?.length || 0,
    });
  }

  // Private helpers

  private handleRegistrationUpdate(registrations: CDPServiceWorkerRegistration[]): void {
    for (const reg of registrations) {
      if (reg.isDeleted) {
        this.registrations.delete(reg.registrationId);
        logger.debug(scopedLog(LogContext.SESSION, 'SW registration removed'), {
          sessionId: this.sessionId,
          scopeURL: reg.scopeURL,
        });
      } else {
        this.registrations.set(reg.registrationId, reg);

        // Auto-unregister if this domain should be blocked
        try {
          const url = new URL(reg.scopeURL);
          if (this.shouldBlockDomain(url.hostname)) {
            logger.debug(scopedLog(LogContext.SESSION, 'auto-unregistering blocked SW'), {
              sessionId: this.sessionId,
              scopeURL: reg.scopeURL,
              hostname: url.hostname,
            });
            // Fire and forget - don't await in event handler
            this.unregister(reg.scopeURL);
          }
        } catch {
          // Invalid URL - skip
        }
      }
    }
  }

  private handleVersionUpdate(versions: CDPServiceWorkerVersion[]): void {
    for (const ver of versions) {
      this.versions.set(ver.versionId, ver);
    }
  }

  private mapStatus(cdpStatus?: string): ServiceWorkerInfo['status'] {
    switch (cdpStatus) {
      case 'running':
        return 'running';
      case 'starting':
        return 'activating';
      case 'stopped':
      case 'stopping':
      default:
        return 'stopped';
    }
  }

  /**
   * Match a domain against a pattern.
   * Supports wildcard patterns like "*.google.com"
   */
  private matchDomain(actual: string, pattern: string): boolean {
    if (pattern.startsWith('*.')) {
      const suffix = pattern.slice(2);
      return actual === suffix || actual.endsWith('.' + suffix);
    }
    return actual === pattern;
  }

  /**
   * Generate the blocking script to inject into pages.
   * This script overrides navigator.serviceWorker.register to block
   * registrations for configured domains.
   */
  private generateBlockingScript(): string {
    const blockedDomains = this.control.blockedDomains || [];
    const overrides = this.control.domainOverrides || [];
    const mode = this.control.mode;

    return `
      (function() {
        const originalRegister = navigator.serviceWorker?.register;
        if (!originalRegister) return;

        const blockedDomains = ${JSON.stringify(blockedDomains)};
        const overrides = ${JSON.stringify(overrides)};
        const mode = '${mode}';

        function matchDomain(actual, pattern) {
          if (pattern.startsWith('*.')) {
            const suffix = pattern.slice(2);
            return actual === suffix || actual.endsWith('.' + suffix);
          }
          return actual === pattern;
        }

        function shouldBlock(hostname) {
          // Check overrides first
          for (const override of overrides) {
            if (matchDomain(hostname, override.domain)) {
              return override.mode === 'block';
            }
          }
          // Check blockedDomains
          if (mode === 'block-on-domain') {
            return blockedDomains.some(d => matchDomain(hostname, d));
          }
          return mode === 'block';
        }

        navigator.serviceWorker.register = function(scriptURL, options) {
          const hostname = window.location.hostname;
          if (shouldBlock(hostname)) {
            console.warn('[playwright-driver] SW registration blocked for:', hostname);
            return Promise.reject(new DOMException('Service worker blocked by automation driver', 'SecurityError'));
          }
          return originalRegister.call(navigator.serviceWorker, scriptURL, options);
        };
      })();
    `;
  }
}
