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
  /** Shared promise lets every shutdown caller join the same cleanup operation. */
  private cleanupInFlight: Promise<void> | null = null;
  /** Track when the last cleanup run completed */
  private lastRunAt: Date | null = null;

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

    this.intervalId = setInterval(() => {
      void this.runCleanup();
    }, this.config.session.cleanupIntervalMs);

    // Don't keep the process alive just for cleanup
    if (this.intervalId.unref) {
      this.intervalId.unref();
    }
  }

  private runCleanup(): Promise<void> {
    // Guard: Don't start cleanup if stopped or another cleanup is in progress
    if (this.isStopped) {
      return Promise.resolve();
    }

    if (this.cleanupInFlight) {
      logger.debug('cleanup: skipping, previous cleanup still in progress');
      return this.cleanupInFlight;
    }

    const operation = this.manager.cleanupIdleSessions().catch((error) => {
      logger.error('cleanup: task error', {
        error: error instanceof Error ? error.message : String(error),
        hint: 'Cleanup may have failed to close some sessions',
      });
    });
    const tracked = operation.finally(() => {
      this.lastRunAt = new Date();
      if (this.cleanupInFlight === tracked) this.cleanupInFlight = null;
    });
    this.cleanupInFlight = tracked;
    return tracked;
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

    // Every caller joins the same in-flight promise. Avoid a single resolver
    // slot that lets a later stop() overwrite an earlier shutdown waiter.
    const cleanup = this.cleanupInFlight;
    if (cleanup) {
      logger.info('cleanup: waiting for in-flight cleanup to complete');
      await cleanup;
    }

    logger.info('cleanup: task stopped');
  }

  /**
   * Check if cleanup task is currently running a cleanup operation.
   * Useful for graceful shutdown to wait for cleanup to complete.
   */
  isRunningCleanup(): boolean {
    return this.cleanupInFlight !== null;
  }

  /**
   * Get the timestamp of the last completed cleanup run.
   * Returns null if cleanup has never run.
   */
  getLastRunAt(): Date | null {
    return this.lastRunAt;
  }

  /**
   * Get the cleanup interval in milliseconds.
   */
  getIntervalMs(): number {
    return this.config.session.cleanupIntervalMs;
  }
}
