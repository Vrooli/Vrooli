import {
  noopInstrumentation,
  resolveInstrumentation,
  safeInvoke,
  type Instrumentation,
  type InstructionInstrumentationContext,
} from '../../../src/instrumentation';

describe('instrumentation seam', () => {
  describe('noopInstrumentation', () => {
    it('has no hooks defined (inert by default)', () => {
      expect(noopInstrumentation.onSessionStart).toBeUndefined();
      expect(noopInstrumentation.onSessionClose).toBeUndefined();
      expect(noopInstrumentation.onInstructionStart).toBeUndefined();
      expect(noopInstrumentation.onInstructionEnd).toBeUndefined();
    });

    it('is frozen so it cannot be mutated', () => {
      expect(Object.isFrozen(noopInstrumentation)).toBe(true);
    });
  });

  describe('resolveInstrumentation', () => {
    it('returns the no-op singleton when given undefined', () => {
      expect(resolveInstrumentation(undefined)).toBe(noopInstrumentation);
    });

    it('returns the supplied instrumentation unchanged', () => {
      const custom: Instrumentation = { onSessionStart: () => {} };
      expect(resolveInstrumentation(custom)).toBe(custom);
    });
  });

  describe('safeInvoke', () => {
    it('resolves without calling anything when the hook is undefined', async () => {
      await expect(safeInvoke(undefined)).resolves.toBeUndefined();
    });

    it('invokes the hook with the provided arguments', async () => {
      const ctx: InstructionInstrumentationContext = {
        sessionId: 's1',
        type: 'navigate',
        index: 0,
        nodeId: 'n1',
      };
      const hook = jest.fn();
      await safeInvoke(hook, ctx);
      expect(hook).toHaveBeenCalledWith(ctx);
    });

    it('swallows synchronous throws (best-effort)', async () => {
      const hook = jest.fn(() => {
        throw new Error('boom');
      });
      await expect(safeInvoke(hook)).resolves.toBeUndefined();
      expect(hook).toHaveBeenCalled();
    });

    it('swallows async rejections (best-effort)', async () => {
      const hook = jest.fn(async () => {
        await Promise.resolve();
        throw new Error('boom-async');
      });
      await expect(safeInvoke(hook)).resolves.toBeUndefined();
      expect(hook).toHaveBeenCalled();
    });

    it('awaits async hooks before resolving', async () => {
      const order: string[] = [];
      const hook = jest.fn(async () => {
        await Promise.resolve();
        order.push('hook');
      });
      await safeInvoke(hook);
      order.push('after');
      expect(order).toEqual(['hook', 'after']);
    });
  });
});
