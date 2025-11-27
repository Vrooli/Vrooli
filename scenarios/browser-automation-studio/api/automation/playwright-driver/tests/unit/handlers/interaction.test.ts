import { InteractionHandler } from '../../../src/handlers/interaction';
import type { CompiledInstruction, HandlerContext } from '../../../src/types';
import { createMockPage, createTestConfig } from '../../helpers';
import { logger, metrics } from '../../../src/utils';

describe('InteractionHandler', () => {
  let handler: InteractionHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new InteractionHandler();
    mockPage = createMockPage();

    const config = createTestConfig();
    context = {
      page: mockPage,
      context: {} as any,
      config,
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  describe('getSupportedTypes', () => {
    it('should support click, hover, and type instructions', () => {
      const types = handler.getSupportedTypes();

      expect(types).toEqual(['click', 'hover', 'type']);
    });
  });

  describe('execute - click', () => {
    it('should click element by selector', async () => {
      const instruction: CompiledInstruction = {
        type: 'click',
        params: { selector: '#button' },
        node_id: 'node-1',
      };

      const result = await handler.execute(instruction, context);

      expect(mockPage.click).toHaveBeenCalledWith('#button', expect.any(Object));
      expect(result.success).toBe(true);
    });

    it('should use custom timeout for click', async () => {
      const instruction: CompiledInstruction = {
        type: 'click',
        params: { selector: '#button', timeoutMs: 10000 },
        node_id: 'node-1',
      };

      await handler.execute(instruction, context);

      expect(mockPage.click).toHaveBeenCalledWith(
        '#button',
        expect.objectContaining({ timeout: 10000 })
      );
    });

    it('should handle click errors', async () => {
      mockPage.click.mockRejectedValue(new Error('Element not found'));

      const instruction: CompiledInstruction = {
        type: 'click',
        params: { selector: '#missing' },
        node_id: 'node-1',
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
    });
  });

  describe('execute - hover', () => {
    it('should hover over element by selector', async () => {
      const instruction: CompiledInstruction = {
        type: 'hover',
        params: { selector: '.item' },
        node_id: 'node-1',
      };

      const result = await handler.execute(instruction, context);

      expect(mockPage.hover).toHaveBeenCalledWith('.item', expect.any(Object));
      expect(result.success).toBe(true);
    });
  });

  describe('execute - type', () => {
    it('should type text into element', async () => {
      const instruction: CompiledInstruction = {
        type: 'type',
        params: { selector: '#input', text: 'Hello World' },
        node_id: 'node-1',
      };

      const result = await handler.execute(instruction, context);

      expect(mockPage.type).toHaveBeenCalledWith('#input', 'Hello World', expect.any(Object));
      expect(result.success).toBe(true);
    });

    it('should use custom delay for typing', async () => {
      const instruction: CompiledInstruction = {
        type: 'type',
        params: { selector: '#input', text: 'Test', delayMs: 100 },
        node_id: 'node-1',
      };

      await handler.execute(instruction, context);

      expect(mockPage.type).toHaveBeenCalledWith(
        '#input',
        'Test',
        expect.objectContaining({ delay: 100 })
      );
    });
  });
});
