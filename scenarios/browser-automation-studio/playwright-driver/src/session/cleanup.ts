import type { SessionManager } from './manager';
import type { Config } from '../config';
import { logger } from '../utils';

/**
 * Background cleanup task for idle sessions
 *
 * Runs periodically to remove sessions that have exceeded idle timeout.
 * This is essential for resource management and prevents memory leaks.
 */
export class SessionCleanup {
  private manager: SessionManager;
  private config: Config;
  private intervalId: NodeJS.Timeout | null = null;

  constructor(manager: SessionManager, config: Config) {
    this.manager = manager;
    this.config = config;
  }

  /**
   * Start background cleanup task
   */
  start(): void {
    if (this.intervalId) {
      logger.warn('cleanup: task already running', {
        hint: 'Cleanup task was started multiple times',
      });
      return;
    }

    logger.info('cleanup: task started', {
      intervalMs: this.config.session.cleanupIntervalMs,
      idleTimeoutMs: this.config.session.idleTimeoutMs,
    });

    this.intervalId = setInterval(async () => {
      try {
        await this.manager.cleanupIdleSessions();
      } catch (error) {
        logger.error('cleanup: task error', {
          error: error instanceof Error ? error.message : String(error),
          hint: 'Cleanup may have failed to close some sessions',
        });
      }
    }, this.config.session.cleanupIntervalMs);

    // Don't keep the process alive just for cleanup
    if (this.intervalId.unref) {
      this.intervalId.unref();
    }
  }

  /**
   * Stop background cleanup task
   */
  stop(): void {
    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
      logger.info('cleanup: task stopped');
    }
  }
}
