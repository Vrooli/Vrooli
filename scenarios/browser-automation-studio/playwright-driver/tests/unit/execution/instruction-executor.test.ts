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
import { executeInstruction, validateInstruction, createInstructionKey } from '../../../src/execution';
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

  describe('createInstructionKey', () => {
    it('builds a stable key from nodeId and index', () => {
      const key = createInstructionKey({
        index: 4,
        nodeId: 'node-xyz',
			action: typedAction,
      });

      expect(key).toBe('node-xyz:4');
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
      expect(mockTelemetryInstance.collectForStep).toHaveBeenCalledWith(handlerResult);
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

    it('disposes telemetry when handler throws', async () => {
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

      await expect(executeInstruction(instruction, baseContext, handlerRegistry)).rejects.toThrow('boom');
      expect(mockTelemetryInstance.dispose).toHaveBeenCalled();
    });
  });
});
