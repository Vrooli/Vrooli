import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { HandlerRegistry } from '../handlers';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import type { CompiledInstruction, StepOutcome } from '../types';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { captureScreenshot, captureDOMSnapshot, ConsoleLogCollector, NetworkCollector } from '../telemetry';
import { logger, metrics as globalMetrics } from '../utils';
import winston from 'winston';

/**
 * Run instruction endpoint
 *
 * POST /session/:id/run
 */
export async function handleSessionRun(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  handlerRegistry: HandlerRegistry,
  config: Config,
  appLogger: winston.Logger,
  appMetrics: Metrics
): Promise<void> {
  const startedAt = new Date();

  try {
    // Get session
    const session = sessionManager.getSession(sessionId);

    // Parse instruction
    const body = await parseJsonBody(req, config);
    const instruction = body as CompiledInstruction;

    logger.info('Running instruction', {
      sessionId,
      type: instruction.type,
      stepIndex: instruction.step_index,
    });

    // Get handler
    const handler = handlerRegistry.getHandler(instruction);

    // Setup telemetry collectors
    const consoleCollector = new ConsoleLogCollector(session.page, config.telemetry.console.maxEntries);
    const networkCollector = new NetworkCollector(session.page, config.telemetry.network.maxEvents);

    // Execute handler
    const instructionStart = Date.now();
    const result = await handler.execute(instruction, {
      page: session.page,
      context: session.context,
      config,
      logger: appLogger,
      metrics: appMetrics,
      sessionId,
    });
    const instructionDuration = Date.now() - instructionStart;

    // Record metrics
    appMetrics.instructionDuration.observe(
      { type: instruction.type, success: String(result.success) },
      instructionDuration
    );

    if (!result.success) {
      appMetrics.instructionErrors.inc({
        type: instruction.type,
        error_kind: result.error?.kind || 'unknown',
      });
    }

    // Capture telemetry (if not already captured by handler)
    const screenshot = result.screenshot || (config.telemetry.screenshot.enabled
      ? await captureScreenshot(session.page, config)
      : undefined);

    const domSnapshot = result.domSnapshot || (config.telemetry.dom.enabled
      ? await captureDOMSnapshot(session.page, config)
      : undefined);

    const consoleLogs = result.consoleLogs || (config.telemetry.console.enabled
      ? consoleCollector.getAndClear()
      : undefined);

    const networkEvents = result.networkEvents || (config.telemetry.network.enabled
      ? networkCollector.getAndClear()
      : undefined);

    // Build step outcome
    const completedAt = new Date();
    const outcome: StepOutcome = {
      schema_version: '1.0.0',
      payload_version: '1.0.0',
      step_index: instruction.step_index,
      success: result.success,
      started_at: startedAt.toISOString(),
      completed_at: completedAt.toISOString(),
      duration_ms: completedAt.getTime() - startedAt.getTime(),
      final_url: session.page.url(),
      screenshot,
      dom_snapshot: domSnapshot,
      console_logs: consoleLogs,
      network: networkEvents,
      extracted_data: result.extracted_data,
      focus: result.focus,
    };

    if (!result.success && result.error) {
      outcome.failure = {
        kind: (result.error.kind as any) || 'engine',
        code: result.error.code,
        message: result.error.message,
        retryable: result.error.retryable || false,
      };
    }

    logger.info('Instruction complete', {
      sessionId,
      type: instruction.type,
      success: result.success,
      duration: outcome.duration_ms,
    });

    sendJson(res, 200, outcome);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/run`);

    // Record error metric
    appMetrics.instructionErrors.inc({
      type: 'unknown',
      error_kind: 'engine',
    });
  }
}
