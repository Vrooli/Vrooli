/**
 * Telemetry Orchestrator
 *
 * Coordinates telemetry collection across all collectors, providing a
 * unified interface for capturing screenshots, DOM snapshots, console logs,
 * and network events.
 *
 * DESIGN:
 * Previously, telemetry collection was scattered throughout routes/session-run.ts.
 * This orchestrator consolidates that logic into a single, reusable service that:
 * - Manages collector lifecycle (init, collect, dispose)
 * - Respects config-based enablement flags
 * - Provides a clean interface for routes and handlers
 *
 * USAGE:
 * ```typescript
 * const orchestrator = new TelemetryOrchestrator(page, config);
 * orchestrator.start();
 *
 * // After each instruction execution:
 * const telemetry = await orchestrator.collectForStep(handlerResult);
 *
 * // When done:
 * orchestrator.dispose();
 * ```
 *
 * @module telemetry/orchestrator
 */

import type { Page } from 'playwright';
import type { Config } from '../config';
import type { HandlerResult, Screenshot, DOMSnapshot, ConsoleLogEntry, NetworkEvent } from '../outcome';
import { ConsoleLogCollector, NetworkCollector } from './collector';
import { captureScreenshot } from './screenshot';
import { captureDOMSnapshot } from './dom';

// =============================================================================
// Types
// =============================================================================

/**
 * Telemetry data collected for a single step.
 */
export interface StepTelemetry {
  screenshot?: Screenshot;
  domSnapshot?: DOMSnapshot;
  consoleLogs?: ConsoleLogEntry[];
  networkEvents?: NetworkEvent[];
}

/**
 * Options for telemetry collection.
 */
export interface TelemetryCollectionOptions {
  /** Force screenshot capture even if handler already provided one */
  forceScreenshot?: boolean;
  /** Force DOM snapshot even if handler already provided one */
  forceDomSnapshot?: boolean;
  /** Include console logs in collection */
  includeConsoleLogs?: boolean;
  /** Include network events in collection */
  includeNetworkEvents?: boolean;
}

// =============================================================================
// TelemetryOrchestrator
// =============================================================================

/**
 * Orchestrates telemetry collection for browser automation sessions.
 *
 * Manages the lifecycle of telemetry collectors and provides a unified
 * interface for collecting screenshots, DOM snapshots, console logs,
 * and network events.
 *
 * @example
 * ```typescript
 * const orchestrator = new TelemetryOrchestrator(page, config);
 * orchestrator.start();
 *
 * for (const instruction of instructions) {
 *   const result = await handler.execute(instruction, context);
 *   const telemetry = await orchestrator.collectForStep(result);
 *   // telemetry now contains screenshot, dom, logs, network
 * }
 *
 * orchestrator.dispose();
 * ```
 */
export class TelemetryOrchestrator {
  private readonly page: Page;
  private readonly config: Config;
  private consoleCollector: ConsoleLogCollector | null = null;
  private networkCollector: NetworkCollector | null = null;
  private started = false;
  private disposed = false;

  constructor(page: Page, config: Config) {
    this.page = page;
    this.config = config;
  }

  /**
   * Start telemetry collection.
   *
   * Initializes console and network collectors if enabled in config.
   * Must be called before collectForStep().
   */
  start(): void {
    if (this.started || this.disposed) return;
    this.started = true;

    // Initialize collectors based on config
    if (this.config.telemetry.console.enabled) {
      this.consoleCollector = new ConsoleLogCollector(
        this.page,
        this.config.telemetry.console.maxEntries
      );
    }

    if (this.config.telemetry.network.enabled) {
      this.networkCollector = new NetworkCollector(
        this.page,
        this.config.telemetry.network.maxEvents
      );
    }
  }

  /**
   * Collect telemetry for a step.
   *
   * Captures screenshot, DOM snapshot, console logs, and network events
   * based on config settings. If the handler result already contains
   * telemetry data, it will be used instead of re-capturing (unless
   * force options are specified).
   *
   * @param handlerResult - Result from handler execution (may contain telemetry)
   * @param options - Collection options (force capture, include/exclude)
   * @returns StepTelemetry with all collected data
   */
  async collectForStep(
    handlerResult?: HandlerResult,
    options?: TelemetryCollectionOptions
  ): Promise<StepTelemetry> {
    if (!this.started) {
      throw new Error('TelemetryOrchestrator not started. Call start() first.');
    }

    if (this.disposed) {
      throw new Error('TelemetryOrchestrator has been disposed.');
    }

    const telemetry: StepTelemetry = {};

    // Screenshot: use handler result or capture if enabled
    if (!handlerResult?.screenshot || options?.forceScreenshot) {
      if (this.config.telemetry.screenshot.enabled) {
        telemetry.screenshot = await captureScreenshot(this.page, this.config);
      }
    } else {
      telemetry.screenshot = handlerResult.screenshot;
    }

    // DOM Snapshot: use handler result or capture if enabled
    if (!handlerResult?.domSnapshot || options?.forceDomSnapshot) {
      if (this.config.telemetry.dom.enabled) {
        telemetry.domSnapshot = await captureDOMSnapshot(this.page, this.config);
      }
    } else {
      telemetry.domSnapshot = handlerResult.domSnapshot;
    }

    // Console logs: collect from collector if enabled
    const includeConsole = options?.includeConsoleLogs ?? true;
    if (includeConsole && this.consoleCollector && this.config.telemetry.console.enabled) {
      telemetry.consoleLogs = handlerResult?.consoleLogs || this.consoleCollector.getAndClear();
    } else if (handlerResult?.consoleLogs) {
      telemetry.consoleLogs = handlerResult.consoleLogs;
    }

    // Network events: collect from collector if enabled
    const includeNetwork = options?.includeNetworkEvents ?? true;
    if (includeNetwork && this.networkCollector && this.config.telemetry.network.enabled) {
      telemetry.networkEvents = handlerResult?.networkEvents || this.networkCollector.getAndClear();
    } else if (handlerResult?.networkEvents) {
      telemetry.networkEvents = handlerResult.networkEvents;
    }

    return telemetry;
  }

  /**
   * Capture screenshot only.
   *
   * Useful for error screenshots or ad-hoc captures without full telemetry.
   */
  async captureScreenshot(): Promise<Screenshot | undefined> {
    if (!this.config.telemetry.screenshot.enabled) return undefined;
    return captureScreenshot(this.page, this.config);
  }

  /**
   * Capture DOM snapshot only.
   *
   * Useful for debugging or ad-hoc captures without full telemetry.
   */
  async captureDomSnapshot(): Promise<DOMSnapshot | undefined> {
    if (!this.config.telemetry.dom.enabled) return undefined;
    return captureDOMSnapshot(this.page, this.config);
  }

  /**
   * Get current console logs without clearing.
   */
  getConsoleLogs(): ConsoleLogEntry[] {
    return this.consoleCollector?.getLogs() ?? [];
  }

  /**
   * Get current network events without clearing.
   */
  getNetworkEvents(): NetworkEvent[] {
    return this.networkCollector?.getEvents() ?? [];
  }

  /**
   * Clear collected console logs.
   */
  clearConsoleLogs(): void {
    this.consoleCollector?.clear();
  }

  /**
   * Clear collected network events.
   */
  clearNetworkEvents(): void {
    this.networkCollector?.clear();
  }

  /**
   * Check if orchestrator is active.
   */
  isActive(): boolean {
    return this.started && !this.disposed;
  }

  /**
   * Dispose the orchestrator and clean up resources.
   *
   * Must be called when telemetry collection is no longer needed to
   * prevent memory leaks from event listeners.
   */
  dispose(): void {
    if (this.disposed) return;
    this.disposed = true;

    this.consoleCollector?.dispose();
    this.networkCollector?.dispose();

    this.consoleCollector = null;
    this.networkCollector = null;
  }
}

/**
 * Create a TelemetryOrchestrator instance.
 *
 * Factory function for creating orchestrator instances.
 * The orchestrator is NOT started automatically - call start() to begin collection.
 *
 * @param page - Playwright page to collect telemetry from
 * @param config - Application config with telemetry settings
 * @returns TelemetryOrchestrator instance (not yet started)
 */
export function createTelemetryOrchestrator(page: Page, config: Config): TelemetryOrchestrator {
  return new TelemetryOrchestrator(page, config);
}
