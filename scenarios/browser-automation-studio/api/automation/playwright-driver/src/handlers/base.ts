import type { Page, BrowserContext } from 'playwright';
import type { CompiledInstruction, StepOutcome, Screenshot, DOMSnapshot, ConsoleLogEntry, NetworkEvent } from '../types';
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
 */
export abstract class BaseHandler implements InstructionHandler {
  abstract getSupportedTypes(): string[];
  abstract execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult>;

  /**
   * Build StepOutcome from handler result
   */
  protected buildOutcome(
    instruction: CompiledInstruction,
    result: HandlerResult,
    startedAt: Date
  ): StepOutcome {
    const completedAt = new Date();

    const outcome: StepOutcome = {
      schema_version: '1.0.0',
      payload_version: '1.0.0',
      step_index: instruction.step_index,
      success: result.success,
      started_at: startedAt.toISOString(),
      completed_at: completedAt.toISOString(),
    };

    // Add telemetry
    if (result.screenshot) {
      outcome.screenshot = result.screenshot;
    }
    if (result.domSnapshot) {
      outcome.dom_snapshot = result.domSnapshot;
    }
    if (result.consoleLogs && result.consoleLogs.length > 0) {
      outcome.console_logs = result.consoleLogs;
    }
    if (result.networkEvents && result.networkEvents.length > 0) {
      outcome.network = result.networkEvents;
    }

    // Add extracted data
    if (result.extracted_data) {
      outcome.extracted_data = result.extracted_data;
    }

    // Add focus
    if (result.focus) {
      outcome.focus = result.focus;
    }

    // Add failure
    if (!result.success && result.error) {
      outcome.failure = {
        kind: (result.error.kind as 'engine' | 'infra' | 'orchestration' | 'user' | 'timeout' | 'cancelled') || 'engine',
        code: result.error.code,
        message: result.error.message,
        retryable: result.error.retryable || false,
      };
    }

    return outcome;
  }

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
      return element.textContent() || '';
    }

    return page.textContent('body') || '';
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
