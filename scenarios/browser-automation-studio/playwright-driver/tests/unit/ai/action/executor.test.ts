/**
 * Action Executor Tests
 */

import { createActionExecutor, createMockActionExecutor } from '../../../../src/ai/action/executor';
import type { Page } from 'playwright';
import type { BrowserAction } from '../../../../src/ai/action/types';
import type { ElementLabel } from '../../../../src/ai/vision-client/types';

/**
 * Create a mock Playwright page for testing.
 */
function createMockPage(): {
  page: Page;
  mocks: {
    click: jest.Mock;
    keyboard: {
      type: jest.Mock;
      press: jest.Mock;
    };
    mouse: {
      click: jest.Mock;
      dblclick: jest.Mock;
      move: jest.Mock;
      wheel: jest.Mock;
    };
    goto: jest.Mock;
    hover: jest.Mock;
    selectOption: jest.Mock;
    waitForSelector: jest.Mock;
    url: jest.Mock;
    viewportSize: jest.Mock;
  };
} {
  const mocks = {
    click: jest.fn().mockResolvedValue(undefined),
    keyboard: {
      type: jest.fn().mockResolvedValue(undefined),
      press: jest.fn().mockResolvedValue(undefined),
    },
    mouse: {
      click: jest.fn().mockResolvedValue(undefined),
      dblclick: jest.fn().mockResolvedValue(undefined),
      move: jest.fn().mockResolvedValue(undefined),
      wheel: jest.fn().mockResolvedValue(undefined),
    },
    goto: jest.fn().mockResolvedValue(undefined),
    hover: jest.fn().mockResolvedValue(undefined),
    selectOption: jest.fn().mockResolvedValue(undefined),
    waitForSelector: jest.fn().mockResolvedValue(undefined),
    url: jest.fn().mockReturnValue('https://example.com'),
    viewportSize: jest.fn().mockReturnValue({ width: 1280, height: 720 }),
  };

  const page = {
    click: mocks.click,
    keyboard: mocks.keyboard,
    mouse: mocks.mouse,
    goto: mocks.goto,
    hover: mocks.hover,
    selectOption: mocks.selectOption,
    waitForSelector: mocks.waitForSelector,
    url: mocks.url,
    viewportSize: mocks.viewportSize,
  } as unknown as Page;

  return { page, mocks };
}

describe('createActionExecutor', () => {
  let executor: ReturnType<typeof createActionExecutor>;
  let page: Page;
  let mocks: ReturnType<typeof createMockPage>['mocks'];

  beforeEach(() => {
    const mockPage = createMockPage();
    page = mockPage.page;
    mocks = mockPage.mocks;
    executor = createActionExecutor();
  });

  describe('click action', () => {
    it('clicks element by ID using selector', async () => {
      const action: BrowserAction = {
        type: 'click',
        elementId: 5,
      };

      const result = await executor.execute(page, action);

      expect(result.success).toBe(true);
      expect(mocks.waitForSelector).toHaveBeenCalledWith(
        '[data-ai-label="5"]',
        expect.any(Object)
      );
      expect(mocks.click).toHaveBeenCalledWith(
        '[data-ai-label="5"]',
        expect.any(Object)
      );
    });

    it('uses element label selector when available', async () => {
      const action: BrowserAction = {
        type: 'click',
        elementId: 5,
      };

      const elementLabels: ElementLabel[] = [
        {
          id: 5,
          selector: '#login-button',
          tagName: 'button',
          bounds: { x: 100, y: 100, width: 80, height: 30 },
        },
      ];

      const result = await executor.execute(page, action, elementLabels);

      expect(result.success).toBe(true);
      expect(mocks.waitForSelector).toHaveBeenCalledWith(
        '#login-button',
        expect.any(Object)
      );
      expect(mocks.click).toHaveBeenCalledWith(
        '#login-button',
        expect.any(Object)
      );
    });

    it('clicks at coordinates', async () => {
      const action: BrowserAction = {
        type: 'click',
        coordinates: { x: 150, y: 300 },
      };

      const result = await executor.execute(page, action);

      expect(result.success).toBe(true);
      expect(mocks.mouse.click).toHaveBeenCalledWith(150, 300);
    });

    it('performs right click', async () => {
      const action: BrowserAction = {
        type: 'click',
        elementId: 5,
        variant: 'right',
      };

      await executor.execute(page, action);

      expect(mocks.click).toHaveBeenCalledWith(
        '[data-ai-label="5"]',
        expect.objectContaining({ button: 'right' })
      );
    });

    it('performs double click by element ID', async () => {
      const action: BrowserAction = {
        type: 'click',
        elementId: 5,
        variant: 'double',
      };

      await executor.execute(page, action);

      expect(mocks.click).toHaveBeenCalledWith(
        '[data-ai-label="5"]',
        expect.objectContaining({ clickCount: 2 })
      );
    });

    it('performs double click by coordinates', async () => {
      const action: BrowserAction = {
        type: 'click',
        coordinates: { x: 100, y: 200 },
        variant: 'double',
      };

      await executor.execute(page, action);

      expect(mocks.mouse.dblclick).toHaveBeenCalledWith(100, 200);
    });
  });

  describe('type action', () => {
    it('types into element by ID', async () => {
      const action: BrowserAction = {
        type: 'type',
        elementId: 3,
        text: 'hello@example.com',
      };

      const result = await executor.execute(page, action);

      expect(result.success).toBe(true);
      expect(mocks.click).toHaveBeenCalledWith('[data-ai-label="3"]');
      expect(mocks.keyboard.type).toHaveBeenCalledWith(
        'hello@example.com',
        expect.any(Object)
      );
    });

    it('clears field before typing when clearFirst is true', async () => {
      const action: BrowserAction = {
        type: 'type',
        elementId: 3,
        text: 'new value',
        clearFirst: true,
      };

      await executor.execute(page, action);

      // Triple click to select all
      expect(mocks.click).toHaveBeenCalledWith(
        '[data-ai-label="3"]',
        expect.objectContaining({ clickCount: 3 })
      );
      expect(mocks.keyboard.type).toHaveBeenCalledWith(
        'new value',
        expect.any(Object)
      );
    });

    it('types into currently focused element when no elementId', async () => {
      const action: BrowserAction = {
        type: 'type',
        text: 'just text',
      };

      await executor.execute(page, action);

      expect(mocks.click).not.toHaveBeenCalled();
      expect(mocks.keyboard.type).toHaveBeenCalledWith(
        'just text',
        expect.any(Object)
      );
    });
  });

  describe('scroll action', () => {
    it('scrolls down', async () => {
      const action: BrowserAction = {
        type: 'scroll',
        direction: 'down',
      };

      await executor.execute(page, action);

      // Default scroll is 80% of viewport height
      expect(mocks.mouse.wheel).toHaveBeenCalledWith(0, 576);
    });

    it('scrolls up', async () => {
      const action: BrowserAction = {
        type: 'scroll',
        direction: 'up',
      };

      await executor.execute(page, action);

      expect(mocks.mouse.wheel).toHaveBeenCalledWith(0, -576);
    });

    it('scrolls with custom amount', async () => {
      const action: BrowserAction = {
        type: 'scroll',
        direction: 'down',
        amount: 300,
      };

      await executor.execute(page, action);

      expect(mocks.mouse.wheel).toHaveBeenCalledWith(0, 300);
    });

    it('scrolls left and right', async () => {
      await executor.execute(page, { type: 'scroll', direction: 'left' });
      expect(mocks.mouse.wheel).toHaveBeenCalledWith(-1024, 0);

      await executor.execute(page, { type: 'scroll', direction: 'right' });
      expect(mocks.mouse.wheel).toHaveBeenCalledWith(1024, 0);
    });
  });

  describe('navigate action', () => {
    it('navigates to URL', async () => {
      const action: BrowserAction = {
        type: 'navigate',
        url: 'https://example.com/login',
      };

      const result = await executor.execute(page, action);

      expect(result.success).toBe(true);
      expect(mocks.goto).toHaveBeenCalledWith(
        'https://example.com/login',
        expect.any(Object)
      );
    });
  });

  describe('hover action', () => {
    it('hovers over element by ID', async () => {
      const action: BrowserAction = {
        type: 'hover',
        elementId: 7,
      };

      await executor.execute(page, action);

      expect(mocks.hover).toHaveBeenCalledWith('[data-ai-label="7"]');
    });

    it('hovers at coordinates', async () => {
      const action: BrowserAction = {
        type: 'hover',
        coordinates: { x: 200, y: 300 },
      };

      await executor.execute(page, action);

      expect(mocks.mouse.move).toHaveBeenCalledWith(200, 300);
    });
  });

  describe('select action', () => {
    it('selects option in dropdown', async () => {
      const action: BrowserAction = {
        type: 'select',
        elementId: 4,
        value: 'option-2',
      };

      await executor.execute(page, action);

      expect(mocks.selectOption).toHaveBeenCalledWith(
        '[data-ai-label="4"]',
        'option-2'
      );
    });
  });

  describe('wait action', () => {
    it('waits for selector', async () => {
      const action: BrowserAction = {
        type: 'wait',
        selector: '#loading-spinner',
      };

      await executor.execute(page, action);

      expect(mocks.waitForSelector).toHaveBeenCalledWith(
        '#loading-spinner',
        expect.any(Object)
      );
    });

    it('waits for specified milliseconds', async () => {
      const action: BrowserAction = {
        type: 'wait',
        ms: 100,
      };

      const start = Date.now();
      await executor.execute(page, action);
      const elapsed = Date.now() - start;

      expect(elapsed).toBeGreaterThanOrEqual(90); // Allow some tolerance
    });
  });

  describe('keypress action', () => {
    it('presses a single key', async () => {
      const action: BrowserAction = {
        type: 'keypress',
        key: 'Enter',
      };

      await executor.execute(page, action);

      expect(mocks.keyboard.press).toHaveBeenCalledWith('Enter');
    });

    it('presses key with modifiers', async () => {
      const action: BrowserAction = {
        type: 'keypress',
        key: 'a',
        modifiers: ['Control'],
      };

      await executor.execute(page, action);

      expect(mocks.keyboard.press).toHaveBeenCalledWith('Control+a');
    });

    it('presses key with multiple modifiers', async () => {
      const action: BrowserAction = {
        type: 'keypress',
        key: 's',
        modifiers: ['Control', 'Shift'],
      };

      await executor.execute(page, action);

      expect(mocks.keyboard.press).toHaveBeenCalledWith('Control+Shift+s');
    });
  });

  describe('done action', () => {
    it('returns success without executing anything', async () => {
      const action: BrowserAction = {
        type: 'done',
        success: true,
        result: 'Task completed',
      };

      const result = await executor.execute(page, action);

      expect(result.success).toBe(true);
      // No page methods should be called
      expect(mocks.click).not.toHaveBeenCalled();
      expect(mocks.goto).not.toHaveBeenCalled();
    });
  });

  describe('error handling', () => {
    it('returns error result when action fails', async () => {
      mocks.click.mockRejectedValueOnce(new Error('Element not found'));

      const action: BrowserAction = {
        type: 'click',
        elementId: 999,
      };

      const result = await executor.execute(page, action);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Element not found');
    });

    it('includes duration in error result', async () => {
      mocks.waitForSelector.mockImplementationOnce(async () => {
        await new Promise((resolve) => setTimeout(resolve, 50));
        throw new Error('Timeout');
      });

      const action: BrowserAction = {
        type: 'click',
        elementId: 5,
      };

      const result = await executor.execute(page, action);

      expect(result.success).toBe(false);
      expect(result.durationMs).toBeGreaterThanOrEqual(50);
    });
  });

  describe('result metadata', () => {
    it('includes duration and URL in successful result', async () => {
      const action: BrowserAction = {
        type: 'click',
        elementId: 5,
      };

      const result = await executor.execute(page, action);

      expect(result.success).toBe(true);
      expect(result.durationMs).toBeGreaterThanOrEqual(0);
      expect(result.newUrl).toBe('https://example.com');
    });
  });
});

describe('createMockActionExecutor', () => {
  it('records calls', async () => {
    const mock = createMockActionExecutor();
    const page = {} as Page;

    await mock.execute(page, { type: 'click', elementId: 5 });
    await mock.execute(page, { type: 'type', text: 'hello' });

    const calls = mock.getCalls();
    expect(calls).toHaveLength(2);
    expect(calls[0].action.type).toBe('click');
    expect(calls[1].action.type).toBe('type');
  });

  it('can be set to fail mode', async () => {
    const mock = createMockActionExecutor();
    const page = {} as Page;

    mock.setFailMode(true, 'Simulated failure');

    const result = await mock.execute(page, { type: 'click', elementId: 5 });

    expect(result.success).toBe(false);
    expect(result.error).toBe('Simulated failure');
  });

  it('clears calls', async () => {
    const mock = createMockActionExecutor();
    const page = {} as Page;

    await mock.execute(page, { type: 'click', elementId: 5 });
    expect(mock.getCalls()).toHaveLength(1);

    mock.clearCalls();
    expect(mock.getCalls()).toHaveLength(0);
  });
});
