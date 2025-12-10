/**
 * Handler Base Types and Utilities
 *
 * STABILITY: STABLE CORE
 *
 * This module defines the foundational interfaces for all instruction handlers.
 * Changes here affect ALL handlers and should be made with care.
 *
 * - HandlerContext: Stable - only additive changes
 * - HandlerResult: Stable - only additive changes
 * - InstructionHandler: Stable - contract for all handlers
 * - BaseHandler: Stable - shared utilities
 *
 * To add a new instruction type, create a new handler file in handlers/
 * that implements InstructionHandler. Do NOT modify this file.
 */

import type { Page, BrowserContext, Frame } from 'playwright';
import type { CompiledInstruction, Screenshot, DOMSnapshot, ConsoleLogEntry, NetworkEvent } from '../types';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import winston from 'winston';

/**
 * Handler execution context
 *
 * Provides all necessary context for handler execution
 */
export interface HandlerContext {
  page: Page;
  context: BrowserContext;
  config: Config;
  logger: winston.Logger;
  metrics: Metrics;
  sessionId: string;

  // Multi-tab support (Phase 3)
  tabStack?: Page[];

  // Frame navigation support (Phase 3)
  // Tracks frame hierarchy for enter/exit operations
  frameStack?: Frame[];

  // Telemetry collectors (optional, handler can add if needed)
  screenshot?: Screenshot;
  domSnapshot?: DOMSnapshot;
  consoleLogs?: ConsoleLogEntry[];
  networkEvents?: NetworkEvent[];
}

/**
 * Handler result
 *
 * Result of handler execution with optional extracted data
 */
export interface HandlerResult {
  success: boolean;
  extracted_data?: Record<string, unknown>;
  focus?: {
    selector?: string;
    bounding_box?: {
      x: number;
      y: number;
      width: number;
      height: number;
    };
  };
  screenshot?: Screenshot;
  domSnapshot?: DOMSnapshot;
  consoleLogs?: ConsoleLogEntry[];
  networkEvents?: NetworkEvent[];
  error?: {
    message: string;
    code?: string;
    kind?: string;
    retryable?: boolean;
  };
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
}
