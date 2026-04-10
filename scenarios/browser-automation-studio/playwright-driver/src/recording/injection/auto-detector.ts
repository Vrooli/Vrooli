/**
 * Injection Strategy Auto-Detector
 *
 * Runtime detection of working injection strategies.
 *
 * ## How It Works
 *
 * The auto-detector:
 * 1. Creates a test page in the browser context
 * 2. Tries each strategy in order
 * 3. Verifies injection using verification markers
 * 4. Returns the first strategy that works
 *
 * ## When to Use
 *
 * - Initial setup when you're unsure which strategy will work
 * - Debugging injection issues
 * - Runtime environment detection
 *
 * ## Performance Note
 *
 * Auto-detection is EXPENSIVE - it requires:
 * - Creating test pages
 * - Trying multiple injection attempts
 * - Waiting for verification timeouts
 *
 * Cache the result and reuse the selected strategy for subsequent contexts.
 *
 * ## Usage
 *
 * ```typescript
 * const detector = new InjectionAutoDetector({
 *   logger: createLogger(),
 * });
 *
 * const result = await detector.detect(context);
 * if (result.strategy) {
 *   console.log('Selected strategy:', result.strategyName);
 *   // Use result.strategy for subsequent injections
 * } else {
 *   console.error('All strategies failed:', result.attempts);
 * }
 * ```
 *
 * @module recording/injection/auto-detector
 */

import type { BrowserContext, Page } from 'rebrowser-playwright';
import type winston from 'winston';
import {
  type InjectionStrategy,
  type InjectionStrategyName,
  type AutoDetectorOptions,
  type AutoDetectionResult,
  type InjectionStrategyOptions,
} from './types';
import { InjectionStrategyFactory, DEFAULT_STRATEGY_ORDER } from './factory';
import { waitForScriptReady } from '../validation/verification';
import { logger as defaultLogger, LogContext, scopedLog } from '../../utils';

/**
 * Default verification timeout in milliseconds.
 */
const DEFAULT_VERIFICATION_TIMEOUT_MS = 5000;

/**
 * Injection Strategy Auto-Detector
 *
 * Detects which injection strategy works in the current environment.
 */
export class InjectionAutoDetector {
  private readonly logger: winston.Logger;
  private readonly strategyOrder: InjectionStrategyName[];
  private readonly verificationTimeoutMs: number;
  private readonly factory: InjectionStrategyFactory;

  constructor(options: AutoDetectorOptions = {}) {
    this.logger = options.logger ?? defaultLogger;
    this.strategyOrder = options.strategyOrder ?? DEFAULT_STRATEGY_ORDER;
    this.verificationTimeoutMs = options.verificationTimeoutMs ?? DEFAULT_VERIFICATION_TIMEOUT_MS;
    this.factory = new InjectionStrategyFactory(this.logger);
  }

  /**
   * Detect a working injection strategy for a browser context.
   *
   * Tries each strategy in order and returns the first one that works.
   *
   * @param context - Browser context to test strategies on
   * @param strategyOptions - Options to pass to each strategy
   * @returns Detection result with the selected strategy
   */
  async detect(
    context: BrowserContext,
    strategyOptions: Omit<InjectionStrategyOptions, 'onFirstInjection'>
  ): Promise<AutoDetectionResult> {
    const startTime = Date.now();
    const attempts: AutoDetectionResult['attempts'] = [];

    this.logger.info(scopedLog(LogContext.INJECTION, 'starting strategy auto-detection'), {
      strategiesToTry: this.strategyOrder,
    });

    for (const strategyName of this.strategyOrder) {
      const attemptStartTime = Date.now();

      try {
        this.logger.debug(scopedLog(LogContext.INJECTION, `trying strategy: ${strategyName}`));

        const strategy = this.factory.createByName(strategyName);
        const success = await this.tryStrategy(context, strategy, strategyOptions);

        const durationMs = Date.now() - attemptStartTime;

        if (success) {
          this.logger.info(scopedLog(LogContext.INJECTION, 'strategy auto-detection complete'), {
            selectedStrategy: strategyName,
            totalDurationMs: Date.now() - startTime,
          });

          attempts.push({
            strategy: strategyName,
            success: true,
            durationMs,
          });

          return {
            strategy,
            strategyName,
            attempts,
            totalDurationMs: Date.now() - startTime,
          };
        }

        attempts.push({
          strategy: strategyName,
          success: false,
          durationMs,
        });
      } catch (error) {
        const durationMs = Date.now() - attemptStartTime;

        this.logger.warn(scopedLog(LogContext.INJECTION, `strategy ${strategyName} failed`), {
          error: error instanceof Error ? error.message : String(error),
          durationMs,
        });

        attempts.push({
          strategy: strategyName,
          success: false,
          error: error instanceof Error ? error.message : String(error),
          durationMs,
        });
      }
    }

    // All strategies failed
    this.logger.error(scopedLog(LogContext.INJECTION, 'all injection strategies failed'), {
      attempts,
      totalDurationMs: Date.now() - startTime,
    });

    return {
      strategy: null,
      strategyName: null,
      attempts,
      totalDurationMs: Date.now() - startTime,
    };
  }

  /**
   * Try a single strategy and return whether it works.
   */
  private async tryStrategy(
    context: BrowserContext,
    strategy: InjectionStrategy,
    strategyOptions: Omit<InjectionStrategyOptions, 'onFirstInjection'>
  ): Promise<boolean> {
    let testPage: Page | null = null;

    try {
      // Initialize the strategy
      await strategy.initialize(context, {
        ...strategyOptions,
        onFirstInjection: undefined, // Don't use callback during detection
      });

      // Create a test page
      testPage = await context.newPage();

      // Navigate to a test URL
      // Using data: URL to avoid network requests
      await testPage.goto('data:text/html,<!DOCTYPE html><html><head></head><body>Test</body></html>', {
        waitUntil: 'domcontentloaded',
      });

      // Wait for script to be ready
      const verification = await waitForScriptReady(
        testPage,
        this.verificationTimeoutMs
      );

      const success = verification.loaded && verification.ready && verification.inMainContext;

      if (strategyOptions.diagnosticsEnabled) {
        this.logger.debug(scopedLog(LogContext.INJECTION, 'strategy verification result'), {
          strategy: strategy.name,
          verification: {
            loaded: verification.loaded,
            ready: verification.ready,
            inMainContext: verification.inMainContext,
            handlersCount: verification.handlersCount,
          },
          success,
        });
      }

      return success;
    } finally {
      // Clean up test page
      if (testPage) {
        await testPage.close().catch(() => {});
      }

      // Clean up strategy
      await strategy.cleanup().catch(() => {});
    }
  }

  /**
   * Detect a working strategy with caching.
   *
   * First checks if a previously detected strategy still works,
   * then falls back to full detection if needed.
   *
   * @param context - Browser context to test
   * @param strategyOptions - Options for strategies
   * @param cachedStrategy - Previously detected strategy to try first
   * @returns Detection result
   */
  async detectWithCache(
    context: BrowserContext,
    strategyOptions: Omit<InjectionStrategyOptions, 'onFirstInjection'>,
    cachedStrategy?: InjectionStrategyName
  ): Promise<AutoDetectionResult> {
    // If we have a cached strategy, try it first
    if (cachedStrategy) {
      this.logger.debug(scopedLog(LogContext.INJECTION, 'trying cached strategy first'), {
        cachedStrategy,
      });

      const attemptStartTime = Date.now();

      try {
        const strategy = this.factory.createByName(cachedStrategy);
        const success = await this.tryStrategy(context, strategy, strategyOptions);

        if (success) {
          this.logger.info(scopedLog(LogContext.INJECTION, 'cached strategy still works'), {
            strategy: cachedStrategy,
            durationMs: Date.now() - attemptStartTime,
          });

          return {
            strategy,
            strategyName: cachedStrategy,
            attempts: [
              {
                strategy: cachedStrategy,
                success: true,
                durationMs: Date.now() - attemptStartTime,
              },
            ],
            totalDurationMs: Date.now() - attemptStartTime,
          };
        }

        this.logger.warn(scopedLog(LogContext.INJECTION, 'cached strategy no longer works'), {
          cachedStrategy,
        });
      } catch (error) {
        this.logger.warn(scopedLog(LogContext.INJECTION, 'cached strategy failed'), {
          cachedStrategy,
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }

    // Fall back to full detection
    return this.detect(context, strategyOptions);
  }
}

/**
 * Create an InjectionAutoDetector instance.
 */
export function createInjectionAutoDetector(options?: AutoDetectorOptions): InjectionAutoDetector {
  return new InjectionAutoDetector(options);
}

/**
 * Detect a working injection strategy.
 *
 * Convenience function that creates a detector and runs detection.
 *
 * @param context - Browser context to test
 * @param strategyOptions - Options for strategies
 * @param detectorOptions - Options for the detector
 * @returns Detection result
 */
export async function detectWorkingStrategy(
  context: BrowserContext,
  strategyOptions: Omit<InjectionStrategyOptions, 'onFirstInjection'>,
  detectorOptions?: AutoDetectorOptions
): Promise<AutoDetectionResult> {
  const detector = new InjectionAutoDetector(detectorOptions);
  return detector.detect(context, strategyOptions);
}
