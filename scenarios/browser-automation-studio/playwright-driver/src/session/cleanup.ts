import type { SessionManager } from './manager';
import type { Config } from '../config';
import { logger } from '../utils';

/**
 * Background cleanup task for idle sessions
 *
 * Runs periodically to remove sessions that have exceeded idle timeout.
 * This is essential for resource management and prevents memory leaks.
 *
 * Temporal hardening:
 * - Cleanup operations are guarded by isStopped flag to prevent work during shutdown
 * - In-flight cleanup is tracked to allow graceful shutdown
 * - stop() returns a promise that resolves when in-flight cleanup completes
 */
export class SessionCleanup {
  private manager: SessionManager;
  private config: Config;
  private intervalId: NodeJS.Timeout | null = null;
  /** Flag to prevent new cleanup work after stop() is called */
  private isStopped = false;
  /** Track if cleanup is currently in progress */
  private isCleanupInProgress = false;
  /** Resolve function for waiting on in-flight cleanup during shutdown */
  private cleanupCompleteResolver: (() => void) | null = null;

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

    if (this.isStopped) {
      logger.warn('cleanup: cannot start after stop', {
        hint: 'Cleanup task was stopped and cannot be restarted',
      });
      return;
    }

    logger.info('cleanup: task started', {
      intervalMs: this.config.session.cleanupIntervalMs,
      idleTimeoutMs: this.config.session.idleTimeoutMs,
    });

    this.intervalId = setInterval(async () => {
      // Guard: Don't start cleanup if stopped or another cleanup is in progress
      if (this.isStopped) {
        return;
      }

      if (this.isCleanupInProgress) {
        logger.debug('cleanup: skipping, previous cleanup still in progress');
        return;
      }

      this.isCleanupInProgress = true;
      try {
        await this.manager.cleanupIdleSessions();
      } catch (error) {
        logger.error('cleanup: task error', {
          error: error instanceof Error ? error.message : String(error),
          hint: 'Cleanup may have failed to close some sessions',
        });
      } finally {
        this.isCleanupInProgress = false;
        // Signal any waiting stop() call that cleanup is complete
        if (this.cleanupCompleteResolver) {
          this.cleanupCompleteResolver();
          this.cleanupCompleteResolver = null;
        }
      }
    }, this.config.session.cleanupIntervalMs);

    // Don't keep the process alive just for cleanup
    if (this.intervalId.unref) {
      this.intervalId.unref();
    }
  }

  /**
   * Stop background cleanup task
   *
   * Sets isStopped flag first to prevent any in-progress or pending cleanup
   * from continuing, then clears the interval.
   *
   * Temporal hardening: If cleanup is in progress, returns a promise that
   * resolves when the in-flight cleanup completes. This ensures graceful
   * shutdown doesn't terminate while cleanup is mid-operation.
   *
   * @returns Promise that resolves when cleanup task is fully stopped
   */
  async stop(): Promise<void> {
    // Set stopped flag first to prevent any pending cleanup from starting
    this.isStopped = true;

    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }

    // If cleanup is in progress, wait for it to complete
    if (this.isCleanupInProgress) {
      logger.info('cleanup: waiting for in-flight cleanup to complete');
      await new Promise<void>((resolve) => {
        this.cleanupCompleteResolver = resolve;
      });
    }

    logger.info('cleanup: task stopped');
  }

  /**
   * Check if cleanup task is currently running a cleanup operation.
   * Useful for graceful shutdown to wait for cleanup to complete.
   */
  isRunningCleanup(): boolean {
    return this.isCleanupInProgress;
  }
}
