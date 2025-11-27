import { createTestInstruction } from '../../helpers';
import { handlerRegistry } from '../../../src/handlers/registry';
import { NavigationHandler, InteractionHandler, WaitHandler } from '../../../src/handlers';
import { UnsupportedInstructionError } from '../../../src/utils/errors';
import type { CompiledInstruction } from '../../../src/types';

describe('HandlerRegistry', () => {
  beforeEach(() => {
    // Clear registry before each test
    (handlerRegistry as any).handlers.clear();
  });

  describe('register', () => {
    it('should register a handler', () => {
      const handler = new NavigationHandler();

      handlerRegistry.register(handler);

      const types = handlerRegistry.getSupportedTypes();
      expect(types).toContain('navigate');
    });

    it('should register multiple handlers', () => {
      const navHandler = new NavigationHandler();
      const waitHandler = new WaitHandler();

      handlerRegistry.register(navHandler);
      handlerRegistry.register(waitHandler);

      const types = handlerRegistry.getSupportedTypes();
      expect(types).toContain('navigate');
      expect(types).toContain('wait');
    });

    it('should register handler supporting multiple types', () => {
      const interactionHandler = new InteractionHandler();

      handlerRegistry.register(interactionHandler);

      const types = handlerRegistry.getSupportedTypes();
      expect(types).toContain('click');
      expect(types).toContain('hover');
      expect(types).toContain('type');
    });

    it('should update count after registration', () => {
      expect(handlerRegistry.getHandlerCount()).toBe(0);

      handlerRegistry.register(new NavigationHandler());
      expect(handlerRegistry.getHandlerCount()).toBe(1);

      handlerRegistry.register(new WaitHandler());
      expect(handlerRegistry.getHandlerCount()).toBe(2);
    });

    it('should replace existing handler for same type', () => {
      const handler1 = new NavigationHandler();
      const handler2 = new NavigationHandler();

      handlerRegistry.register(handler1);
      handlerRegistry.register(handler2);

      // Should still have only 1 handler count for navigate
      const types = handlerRegistry.getSupportedTypes();
      expect(types.filter((t) => t === 'navigate')).toHaveLength(1);
    });
  });

  describe('getHandler', () => {
    beforeEach(() => {
      handlerRegistry.register(new NavigationHandler());
      handlerRegistry.register(new InteractionHandler());
      handlerRegistry.register(new WaitHandler());
    });

    it('should return handler for navigate instruction', () => {
      const instruction = createTestInstruction({
        type: 'navigate',
        params: {},
        node_id: 'node-1',
      });

      const handler = handlerRegistry.getHandler(instruction);

      expect(handler).toBeInstanceOf(NavigationHandler);
    });

    it('should return handler for click instruction', () => {
      const instruction = createTestInstruction({
        type: 'click',
        params: {},
        node_id: 'node-1',
      });

      const handler = handlerRegistry.getHandler(instruction);

      expect(handler).toBeInstanceOf(InteractionHandler);
    });

    it('should return handler for wait instruction', () => {
      const instruction = createTestInstruction({
        type: 'wait',
        params: {},
        node_id: 'node-1',
      });

      const handler = handlerRegistry.getHandler(instruction);

      expect(handler).toBeInstanceOf(WaitHandler);
    });

    it('should throw error for unsupported instruction type', () => {
      const instruction = createTestInstruction({
        type: 'unsupported-action',
        params: {},
        node_id: 'node-1',
      });

      expect(() => handlerRegistry.getHandler(instruction)).toThrow(UnsupportedInstructionError);
      expect(() => handlerRegistry.getHandler(instruction)).toThrow(
        'Unsupported instruction type: unsupported-action'
      );
    });
  });

  describe('getSupportedTypes', () => {
    it('should return empty array when no handlers registered', () => {
      const types = handlerRegistry.getSupportedTypes();

      expect(types).toEqual([]);
    });

    it('should return all supported types', () => {
      handlerRegistry.register(new NavigationHandler());
      handlerRegistry.register(new InteractionHandler());
      handlerRegistry.register(new WaitHandler());

      const types = handlerRegistry.getSupportedTypes();

      expect(types).toContain('navigate');
      expect(types).toContain('click');
      expect(types).toContain('hover');
      expect(types).toContain('type');
      expect(types).toContain('wait');
    });

    it('should return unique types', () => {
      handlerRegistry.register(new NavigationHandler());
      handlerRegistry.register(new NavigationHandler()); // Register again

      const types = handlerRegistry.getSupportedTypes();

      expect(types.filter((t) => t === 'navigate')).toHaveLength(1);
    });
  });

  describe('getHandlerCount', () => {
    it('should return 0 when no handlers registered', () => {
      expect(handlerRegistry.getHandlerCount()).toBe(0);
    });

    it('should count unique instruction types', () => {
      handlerRegistry.register(new NavigationHandler()); // 1 type
      expect(handlerRegistry.getHandlerCount()).toBe(1);

      handlerRegistry.register(new InteractionHandler()); // 3 types (click, hover, type)
      expect(handlerRegistry.getHandlerCount()).toBe(4);

      handlerRegistry.register(new WaitHandler()); // 1 type
      expect(handlerRegistry.getHandlerCount()).toBe(5);
    });
  });
});
