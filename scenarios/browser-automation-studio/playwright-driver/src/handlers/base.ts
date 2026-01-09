/**
 * Handler Base Types and Utilities
 *
 * STABILITY: STABLE CORE
 *
 * This module defines the foundational interfaces for all instruction handlers.
 * Changes here affect ALL handlers and should be made with care.
 *
 * - HandlerContext: Stable - only additive changes
 * - HandlerResult: Re-exported from outcome-builder (stable)
 * - InstructionHandler: Stable - contract for all handlers
 * - BaseHandler: Stable - shared utilities
 *
 * To add a new instruction type, create a new handler file in handlers/
 * that implements InstructionHandler. Do NOT modify this file.
 */

import type { Page, BrowserContext, Frame } from 'rebrowser-playwright';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import type winston from 'winston';

// Import types from proto and outcome-builder
import type { HandlerInstruction } from '../proto';
import type { HandlerResult, Screenshot, DOMSnapshot, ConsoleLogEntry, NetworkEvent } from '../outcome/outcome-builder';

// Re-export for handler use
export type { HandlerResult, Screenshot, DOMSnapshot, ConsoleLogEntry, NetworkEvent };

// HandlerInstruction is the primary instruction type for handlers
export type { HandlerInstruction };

/**
 * CompiledInstruction is an alias for HandlerInstruction.
 * Handlers use this type for instruction parameters.
 */
export type CompiledInstruction = HandlerInstruction;

/**
 * Handler execution context
 *
 * Provides all necessary context for handler execution.
 * This is the UNIFIED context type used across the execution pipeline.
 * Both handlers and the instruction executor use this same type.
 *
 * SEAM: Unified Execution Context
 * This interface defines the contract between the execution layer and handlers.
 * Adding new context fields should be done here, not in individual handlers.
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ FIELD LIFECYCLE:                                                        │
 * │                                                                         │
 * │   REQUIRED (always populated):                                          │
 * │     page, browserContext, config, logger, metrics, sessionId            │
 * │                                                                         │
 * │   OPTIONAL (populated when relevant):                                   │
 * │     tabStack - when multi-tab operations are used                       │
 * │     frameStack - when frame navigation is used                          │
 * │                                                                         │
 * │   OUTPUT (handlers may populate):                                       │
 * │     screenshot, domSnapshot, consoleLogs, networkEvents                 │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * NOTE: Uses `browserContext` (not `context`) to avoid confusion with
 * the overloaded term "context" in other parts of the codebase.
 */
export interface HandlerContext {
  // -------------------------------------------------------------------------
  // REQUIRED FIELDS - Always populated by execution layer
  // -------------------------------------------------------------------------

  /** The main Playwright page for browser interaction */
  page: Page;
  /** The Playwright browser context (for cookies, permissions, storage) */
  browserContext: BrowserContext;
  /** Application configuration */
  config: Config;
  /** Scoped logger instance */
  logger: winston.Logger;
  /** Metrics collector for observability */
  metrics: Metrics;
  /** Session identifier for logging and tracking */
  sessionId: string;

  // -------------------------------------------------------------------------
  // OPTIONAL FIELDS - Populated when relevant features are used
  // -------------------------------------------------------------------------

  /**
   * Multi-tab support - stack of open pages.
   * Populated when new-tab instruction opens additional tabs.
   * The current page is always at the top of the stack.
   */
  tabStack?: Page[];

  /**
   * Frame navigation support - stack of frame hierarchy.
   * Populated when enter-frame instruction navigates into iframes.
   * Used to track frame depth for exit-frame operations.
   */
  frameStack?: Frame[];

  // -------------------------------------------------------------------------
  // OUTPUT FIELDS - Handlers may populate for telemetry
  // -------------------------------------------------------------------------

  /**
   * Screenshot captured during handler execution.
   * If populated, the orchestrator will use this instead of capturing.
   */
  screenshot?: Screenshot;
  /** DOM snapshot captured during handler execution */
  domSnapshot?: DOMSnapshot;
  /** Console log entries observed during execution */
  consoleLogs?: ConsoleLogEntry[];
  /** Network events observed during execution */
  networkEvents?: NetworkEvent[];
}

/**
 * Base handler interface
 *
 * All instruction handlers must implement this interface
 */
export interface InstructionHandler {
  /**
   * Get instruction types this handler supports
   */
  getSupportedTypes(): string[];

  /**
   * Execute instruction
   */
  execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult>;
}

/**
 * Base handler implementation with common utilities
 *
 * Handlers are responsible for executing browser automation instructions.
 * Outcome building is delegated to domain/outcome-builder.ts - handlers
 * return HandlerResult which is transformed by the route layer.
 */
export abstract class BaseHandler implements InstructionHandler {
  abstract getSupportedTypes(): string[];
  abstract execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult>;

  /**
   * Wait for element with timeout
   */
  protected async waitForElement(
    page: Page,
    selector: string,
    timeoutMs?: number
  ): Promise<void> {
    await page.waitForSelector(selector, {
      timeout: timeoutMs,
      state: 'visible',
    });
  }

  /**
   * Get element bounding box
   */
  protected async getBoundingBox(
    page: Page,
    selector: string
  ): Promise<{ x: number; y: number; width: number; height: number } | null> {
    const element = await page.locator(selector).first();
    const box = await element.boundingBox();

    if (!box) {
      return null;
    }

    return {
      x: box.x,
      y: box.y,
      width: box.width,
      height: box.height,
    };
  }

  /**
   * Extract text from page
   */
  protected async extractText(
    page: Page,
    selector?: string
  ): Promise<string> {
    if (selector) {
      const element = await page.locator(selector).first();
      const text = await element.textContent();
      return text || '';
    }

    const bodyText = await page.textContent('body');
    return bodyText || '';
  }

  /**
   * Extract attribute from element
   */
  protected async extractAttribute(
    page: Page,
    selector: string,
    attribute: string
  ): Promise<string | null> {
    const element = await page.locator(selector).first();
    return element.getAttribute(attribute);
  }

  /**
   * Require typed params from instruction.action.
   * Throws an error if the instruction doesn't have a typed action - this indicates
   * an execution path that hasn't been migrated to populate the Action field.
   */
  protected requireTypedParams<T>(
    params: T | undefined,
    handlerType: string,
    nodeId: string
  ): T {
    if (!params) {
      throw new Error(
        `[${handlerType}] Missing typed action params for node ${nodeId}. ` +
        `This indicates an unmigrated execution path - all instructions should have action populated.`
      );
    }
    return params;
  }

  /**
   * Create a missing required parameter error result.
   */
  protected missingParamError(handlerType: string, paramName: string): HandlerResult {
    return {
      success: false,
      error: {
        message: `${handlerType} instruction missing ${paramName} parameter`,
        code: 'MISSING_PARAM',
        kind: 'orchestration',
        retryable: false,
      },
    };
  }
}
