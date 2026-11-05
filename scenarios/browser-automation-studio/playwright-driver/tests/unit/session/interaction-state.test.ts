import type { Page } from 'rebrowser-playwright';

import {
  applyInteractionState,
  configureInteractionState,
  interactionStateForContext,
  InteractionStateApplicationError,
} from '../../../src/session/interaction-state';

function createPage(
  options: {
    targetVisible?: boolean;
    targetBoundingBox?: boolean;
    targetEvaluate?: boolean;
  } = {}
) {
  const target = {
    isVisible: jest.fn().mockResolvedValue(options.targetVisible ?? true),
    boundingBox: jest
      .fn()
      .mockResolvedValue(
        options.targetBoundingBox === false ? null : { x: 1, y: 1, width: 20, height: 20 }
      ),
    waitFor: jest.fn().mockResolvedValue(undefined),
    hover: jest.fn().mockResolvedValue(undefined),
    evaluate: jest.fn().mockResolvedValue(options.targetEvaluate ?? true),
  };
  const candidates = {
    count: jest.fn().mockResolvedValue(1),
    nth: jest.fn().mockReturnValue(target),
    first: jest.fn().mockReturnValue(target),
  };
  const page = {
    locator: jest.fn().mockReturnValue(candidates),
    keyboard: { press: jest.fn().mockResolvedValue(undefined) },
    mouse: {
      down: jest.fn().mockResolvedValue(undefined),
      up: jest.fn().mockResolvedValue(undefined),
    },
  } as unknown as Page;
  return { page, target, candidates };
}

describe('interaction-state applicator', () => {
  it.each([
    ['hover', 'hover'],
    ['focus-visible', 'focus-visible'],
    ['pressed', 'pressed'],
    ['disabled', 'disabled'],
  ] as const)('applies %s through native Playwright interaction', async (state, expected) => {
    const { page, target } = createPage();

    await applyInteractionState(page, state);

    expect(target.waitFor).toHaveBeenCalledWith({ state: 'visible', timeout: 10_000 });
    expect(target.evaluate).toHaveBeenCalled();
    if (expected === 'hover' || expected === 'pressed') {
      expect(target.hover).toHaveBeenCalledWith({ timeout: 10_000 });
    }
    if (expected === 'focus-visible') expect(page.keyboard.press).toHaveBeenCalledWith('Tab');
    if (expected === 'pressed') expect(page.mouse.down).toHaveBeenCalledTimes(1);
  });

  it('does not touch the page for rest', async () => {
    const { page } = createPage();

    await applyInteractionState(page, 'rest');

    expect(page.locator).not.toHaveBeenCalled();
  });

  it('fails with a typed error when no target can hold the state', async () => {
    const { page, candidates } = createPage();
    candidates.count.mockResolvedValue(0);

    const result = applyInteractionState(page, 'hover');

    await expect(result).rejects.toBeInstanceOf(InteractionStateApplicationError);
    await expect(result).rejects.toThrow('interaction state "hover"');
  });

  it('stores the state on the browser context used by navigation', () => {
    const context = {} as Parameters<typeof configureInteractionState>[0];

    configureInteractionState(context, 'pressed');

    expect(interactionStateForContext(context)).toBe('pressed');
  });
});
