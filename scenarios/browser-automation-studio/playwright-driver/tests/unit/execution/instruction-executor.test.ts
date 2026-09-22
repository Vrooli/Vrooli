import type { HandlerInstruction } from '../../../src/proto';
import type { HandlerRegistry } from '../../../src/handlers';
import type { HandlerResult } from '../../../src/outcome';
import type { Metrics } from '../../../src/utils/metrics';
import type { ExecutionContext } from '../../../src/execution';
import type { Page, BrowserContext } from 'rebrowser-playwright';
import type winston from 'winston';

jest.mock('../../../src/telemetry', () => ({
  TelemetryOrchestrator: jest.fn(),
}));

jest.mock('../../../src/outcome', () => ({
  buildStepOutcome: jest.fn(),
  toDriverOutcome: jest.fn(),
}));

jest.mock('../../../src/proto', () => ({
	CompiledInstructionSchema: {},
	parseProtoLenient: jest.fn(),
	toHandlerInstruction: jest.fn(),
	getActionType: jest.fn().mockReturnValue('click'),
}));

import { TelemetryOrchestrator } from '../../../src/telemetry';
import { buildStepOutcome, toDriverOutcome } from '../../../src/outcome';
import { parseProtoLenient, toHandlerInstruction } from '../../../src/proto';
import { executeInstruction, validateInstruction } from '../../../src/execution';
import { createTestConfig } from '../../helpers/test-config';

const typedAction = {} as HandlerInstruction['action'];

describe('Instruction executor', () => {
  const observeInstructionDuration = jest.fn();
  const incrementInstructionErrors = jest.fn();
  const mockMetrics = {
    instructionDuration: { observe: observeInstructionDuration },
    instructionErrors: { inc: incrementInstructionErrors },
  } as unknown as Metrics;

  const mockPage = {
    url: jest.fn().mockReturnValue('https://example.com'),
  } as unknown as Page;

  const mockContext = {} as unknown as BrowserContext;

  const baseContext: ExecutionContext = {
    page: mockPage,
    browserContext: mockContext,
    config: createTestConfig(),
    logger: console as unknown as winston.Logger,
    metrics: mockMetrics,
    sessionId: 'session-123',
  };

  const mockTelemetryInstance = {
    start: jest.fn(),
    collectForStep: jest.fn().mockResolvedValue({
      screenshot: { base64: 'abc', media_type: 'image/png' },
      domSnapshot: { html: '<html></html>', preview: '<html></html>' },
    }),
    dispose: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    (TelemetryOrchestrator as jest.Mock).mockImplementation(() => mockTelemetryInstance);
  });

  describe('validateInstruction', () => {
    it('rejects invalid structures early', () => {
		const missingIndex = validateInstruction({ node_id: 'node-1', action: { type: 'ACTION_TYPE_CLICK' } });
      expect(missingIndex.valid).toBe(false);

		const missingNode = validateInstruction({ index: 0, action: { type: 'ACTION_TYPE_CLICK' } });
      expect(missingNode.valid).toBe(false);

		const missingType = validateInstruction({ index: 0, node_id: 'node-1' });
      expect(missingType.valid).toBe(false);

		const missingParams = validateInstruction({ index: 0, node_id: 'node-1', action: {} });
      expect(missingParams.valid).toBe(false);
    });

    it('parses valid instructions and returns handler format', () => {
      const handlerInstruction: HandlerInstruction = {
        index: 1,
        nodeId: 'node-1',
			action: typedAction,
      };

      (parseProtoLenient as jest.Mock).mockReturnValue({
        index: 1,
        nodeId: 'node-1',
			action: typedAction,
      });
      (toHandlerInstruction as jest.Mock).mockReturnValue(handlerInstruction);

      const result = validateInstruction({
        index: 1,
        node_id: 'node-1',
			action: { type: 'ACTION_TYPE_CLICK' },
      });

      expect(result.valid).toBe(true);
      if (result.valid) {
        expect(result.instruction).toBe(handlerInstruction);
      }
    });
  });

  describe('executeInstruction', () => {
    it('executes handler and returns outcomes on success', async () => {
      const instruction: HandlerInstruction = {
        index: 0,
        nodeId: 'node-1',
			action: typedAction,
      };

      const handlerResult: HandlerResult = { success: true };

      const executeHandler = jest.fn().mockResolvedValue(handlerResult);
      const handler = {
        execute: executeHandler,
      };

      const getHandler = jest.fn().mockReturnValue(handler);
      const handlerRegistry = {
        getHandler,
      } as unknown as HandlerRegistry;

      const outcome = { durationMs: 12 };
      const driverOutcome = { success: true };
      (buildStepOutcome as jest.Mock).mockReturnValue(outcome);
      (toDriverOutcome as jest.Mock).mockReturnValue(driverOutcome);

      const result = await executeInstruction(instruction, baseContext, handlerRegistry);

      expect(getHandler).toHaveBeenCalledWith(instruction);
      expect(executeHandler).toHaveBeenCalledWith(instruction, baseContext);
      expect(mockTelemetryInstance.start).toHaveBeenCalled();
      // This instruction carries no telemetry directive, so the capture policy
      // must resolve to "capture" — the pre-directive default.
      expect(mockTelemetryInstance.collectForStep).toHaveBeenCalledWith(handlerResult, {
        skipScreenshot: false,
      });
      expect(mockTelemetryInstance.dispose).toHaveBeenCalled();
      expect(result.outcome).toBe(outcome);
      expect(result.driverOutcome).toBe(driverOutcome);
      expect(observeInstructionDuration).toHaveBeenCalledWith(
        { type: 'click', success: 'true' },
        expect.any(Number)
      );
      expect(incrementInstructionErrors).not.toHaveBeenCalled();
    });

    it('records errors when handler fails', async () => {
      const instruction: HandlerInstruction = {
        index: 0,
        nodeId: 'node-1',
			action: typedAction,
      };

      const handlerResult: HandlerResult = {
        success: false,
        error: { code: 'TIMEOUT', message: 'timeout', kind: 'timeout' },
      };

      const executeHandler = jest.fn().mockResolvedValue(handlerResult);
      const handler = {
        execute: executeHandler,
      };

      const getHandler = jest.fn().mockReturnValue(handler);
      const handlerRegistry = {
        getHandler,
      } as unknown as HandlerRegistry;

      const outcome = { durationMs: 20 };
      (buildStepOutcome as jest.Mock).mockReturnValue(outcome);
      (toDriverOutcome as jest.Mock).mockReturnValue({ success: false });

      await executeInstruction(instruction, baseContext, handlerRegistry);

      expect(incrementInstructionErrors).toHaveBeenCalledWith({
        type: 'click',
        error_kind: 'timeout',
      });
    });

    it('captures failure evidence before releasing a thrown handler operation', async () => {
      const instruction: HandlerInstruction = {
        index: 0,
        nodeId: 'node-1',
			action: typedAction,
      };

      const executeHandler = jest.fn().mockRejectedValue(new Error('boom'));
      const handler = {
        execute: executeHandler,
      };

      const getHandler = jest.fn().mockReturnValue(handler);
      const handlerRegistry = {
        getHandler,
      } as unknown as HandlerRegistry;

      const result = await executeInstruction(instruction, baseContext, handlerRegistry);
      expect(result.success).toBe(false);
      expect(result.handlerResult.error).toMatchObject({
        code: 'INSTRUCTION_OUTCOME_UNCERTAIN', retryable: false, message: expect.stringContaining('boom'),
      });
      expect(mockTelemetryInstance.collectForStep).toHaveBeenCalledWith(result.handlerResult, { skipScreenshot: false });
      expect(result.telemetry.screenshot).toBeDefined();
      expect(executeHandler).toHaveBeenCalledTimes(1);
      expect(mockTelemetryInstance.dispose).toHaveBeenCalledTimes(1);
    });
    it.each(['start', 'collect', 'build', 'wire'])('releases collectors after a %s failure', async (site) => {
      const instruction = { index: 0, nodeId: 'fault', action: typedAction };
      const execute = jest.fn().mockResolvedValue({ success: true });
      const registry = { getHandler: () => ({ execute }) } as unknown as HandlerRegistry;
      const fail = () => { throw new Error(`${site} fault`); };
      if (site === 'start') mockTelemetryInstance.start.mockRejectedValueOnce(new Error('start fault'));
      if (site === 'collect') mockTelemetryInstance.collectForStep.mockImplementationOnce(fail);
      if (site === 'build') (buildStepOutcome as jest.Mock).mockImplementationOnce(fail);
      if (site === 'wire') (toDriverOutcome as jest.Mock).mockImplementationOnce(fail);
      await expect(executeInstruction(instruction, baseContext, registry)).rejects.toThrow(`${site} fault`);
      expect(mockTelemetryInstance.dispose).toHaveBeenCalledTimes(1);
      expect(execute).toHaveBeenCalledTimes(site === 'start' ? 0 : 1);
    });

    it('retains capture and the known result when a metrics observer throws', async () => {
      observeInstructionDuration.mockImplementationOnce(() => { throw new Error('metrics unavailable'); });
      const registry = { getHandler: () => ({ execute: async () => ({ success: true }) }) } as unknown as HandlerRegistry;
      const result = await executeInstruction({ index: 0, nodeId: 'metrics', action: typedAction }, baseContext, registry);
      expect(result.success).toBe(true);
      expect(result.telemetry.screenshot).toBeDefined();
      expect(mockTelemetryInstance.dispose).toHaveBeenCalledTimes(1);
    });

    it.each([true, false])('reports partial evidence while preserving a declared failure (handler success=%s)', async (success) => {
      const declaredError = { code: 'ASSERTION_FAILED', kind: 'engine', message: 'expected button absent', retryable: false };
      const registry = { getHandler: () => ({ execute: async () => ({ success, error: success ? undefined : declaredError }) }) } as unknown as HandlerRegistry;
      mockTelemetryInstance.collectForStep.mockResolvedValueOnce({ captureErrors: ['dom: transport failed'], consoleLogs: [{ text: 'available' }] });
      (buildStepOutcome as jest.Mock).mockReturnValueOnce({ durationMs: 10, notes: {} });
      const result = await executeInstruction({ index: 0, nodeId: 'partial', action: typedAction }, baseContext, registry);
      expect(result.success).toBe(false);
      expect(result.handlerResult.error).toMatchObject(success ? { code: 'INSTRUCTION_EVIDENCE_FAILED', retryable: false } : declaredError);
      expect(result.telemetry.consoleLogs).toEqual([{ text: 'available' }]);
      expect(result.outcome.notes.telemetry_errors).toBe('["dom: transport failed"]');
      expect(mockTelemetryInstance.dispose).toHaveBeenCalledTimes(1);
    });

  });
});
