import type { SessionManager } from './manager';
import type { Config } from '../config';
import { logger } from '../utils';

/**
 * Background cleanup task for idle sessions
 *
 * Runs periodically to remove sessions that have exceeded idle timeout
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
      logger.warn('Cleanup task already running');
      return;
    }

    logger.info('Starting session cleanup task', {
      intervalMs: this.config.session.cleanupIntervalMs,
      idleTimeoutMs: this.config.session.idleTimeoutMs,
    });

    this.intervalId = setInterval(async () => {
      try {
        await this.manager.cleanupIdleSessions();
      } catch (error) {
        logger.error('Error in cleanup task', {
          error: error instanceof Error ? error.message : String(error),
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
      logger.info('Session cleanup task stopped');
    }
  }
}
