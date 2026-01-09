import { createTypedInstruction, createMockPage, createTestConfig } from '../../helpers';
import { InteractionHandler } from '../../../src/handlers/interaction';
import type { HandlerContext } from '../../../src/handlers/base';
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
    it('should support click, hover, type, focus, and blur instructions', () => {
      const types = handler.getSupportedTypes();

      expect(types).toEqual(['click', 'hover', 'type', 'focus', 'blur']);
    });
  });

  describe('execute - click', () => {
    it('should click element by selector', async () => {
      const instruction = createTypedInstruction('click', { selector: '#button' }, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      expect(mockPage.click).toHaveBeenCalledWith('#button', expect.any(Object));
      expect(result.success).toBe(true);
    });

    it('should use config default timeout for click', async () => {
      // Note: ClickParams in proto does NOT have timeoutMs field
      // Click timeout is controlled by config.execution.defaultTimeoutMs (30000 in test config)
      const instruction = createTypedInstruction('click', { selector: '#button' }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      // Uses config default timeout (30000ms from createTestConfig)
      expect(mockPage.click).toHaveBeenCalledWith(
        '#button',
        expect.objectContaining({ timeout: 30000 })
      );
    });

    it('should handle click errors', async () => {
      mockPage.click.mockRejectedValue(new Error('Element not found'));

      const instruction = createTypedInstruction('click', { selector: '#missing' }, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
    });
  });

  describe('execute - hover', () => {
    it('should hover over element by selector', async () => {
      const instruction = createTypedInstruction('hover', { selector: '.item' }, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      expect(mockPage.hover).toHaveBeenCalledWith('.item', expect.any(Object));
      expect(result.success).toBe(true);
    });
  });

  describe('execute - type', () => {
    it('should fill text into element', async () => {
      const instruction = createTypedInstruction('type', { selector: '#input', text: 'Hello World' }, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      expect(mockPage.fill).toHaveBeenCalledWith('#input', 'Hello World', expect.any(Object));
      expect(result.success).toBe(true);
    });

    it('should fill with value param when text not provided', async () => {
      const instruction = createTypedInstruction('type', { selector: '#input', value: 'Test' }, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      expect(mockPage.fill).toHaveBeenCalledWith('#input', 'Test', expect.any(Object));
      expect(result.success).toBe(true);
    });
  });
});
