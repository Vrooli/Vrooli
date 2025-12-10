import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { HandlerRegistry } from '../handlers';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import type { CompiledInstruction } from '../types';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { captureScreenshot, captureDOMSnapshot, ConsoleLogCollector, NetworkCollector } from '../telemetry';
import { buildStepOutcome, toDriverOutcome } from '../domain';
import { logger } from '../utils';
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
    const instruction = (body as any).instruction as CompiledInstruction;

    logger.info('instruction: executing', {
      sessionId,
      type: instruction.type,
      stepIndex: instruction.index,
      nodeId: instruction.node_id,
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

    // Build step outcome using domain logic
    const completedAt = new Date();
    const outcome = buildStepOutcome({
      instruction,
      result,
      startedAt,
      completedAt,
      finalUrl: session.page.url(),
      screenshot: screenshot as Parameters<typeof buildStepOutcome>[0]['screenshot'],
      domSnapshot,
      consoleLogs,
      networkEvents,
    });

    logger.info('instruction: completed', {
      sessionId,
      type: instruction.type,
      stepIndex: instruction.index,
      success: result.success,
      durationMs: outcome.duration_ms,
      ...(result.error && { error: result.error.code }),
    });

    // Convert to driver wire format (flat fields for screenshot/dom)
    const screenshotData = screenshot as { base64?: string; media_type?: string; width?: number; height?: number } | undefined;
    const driverOutcome = toDriverOutcome(outcome, screenshotData, domSnapshot);

    sendJson(res, 200, driverOutcome);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/run`);

    // Record error metric
    appMetrics.instructionErrors.inc({
      type: 'unknown',
      error_kind: 'engine',
    });
  }
}
