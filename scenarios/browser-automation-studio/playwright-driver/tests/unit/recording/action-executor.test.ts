/**
 * Tests for Action Executor
 *
 * These tests verify the replay execution for each action type.
 * They serve as regression guards for the Recording Action Types change axis.
 */
import type { Page } from 'playwright';
import type { RecordedAction, SelectorValidation } from '../../../src/recording/types';
import type { ActionExecutorContext } from '../../../src/recording/action-executor';
import {
  registerActionExecutor,
  getActionExecutor,
  hasActionExecutor,
  getRegisteredActionTypes,
} from '../../../src/recording/action-executor';

// Mock Page object
const createMockPage = (): jest.Mocked<Page> => {
  return {
    click: jest.fn().mockResolvedValue(undefined),
    fill: jest.fn().mockResolvedValue(undefined),
    goto: jest.fn().mockResolvedValue(null),
    selectOption: jest.fn().mockResolvedValue([]),
    focus: jest.fn().mockResolvedValue(undefined),
    hover: jest.fn().mockResolvedValue(undefined),
    evaluate: jest.fn().mockResolvedValue(undefined),
    keyboard: {
      press: jest.fn().mockResolvedValue(undefined),
    },
  } as unknown as jest.Mocked<Page>;
};

// Create base action
const createBaseAction = (overrides: Partial<RecordedAction> = {}): RecordedAction => ({
  id: 'test-action-id',
  sessionId: 'test-session-id',
  sequenceNum: 0,
  timestamp: new Date().toISOString(),
  actionType: 'click',
  confidence: 0.9,
  url: 'https://example.com',
  selector: {
    primary: '[data-testid="test-button"]',
    candidates: [{ type: 'data-testid', value: '[data-testid="test-button"]', confidence: 0.98, specificity: 100 }],
  },
  ...overrides,
});

// Create base result
const createBaseResult = () => ({
  actionId: 'test-action-id',
  sequenceNum: 0,
  actionType: 'click' as const,
  success: false,
  durationMs: 0,
});

// Create executor context
const createContext = (
  page: Page,
  validationResult: SelectorValidation = { valid: true, matchCount: 1, selector: '[data-testid="test-button"]' }
): ActionExecutorContext => ({
  page,
  timeout: 5000,
  validateSelector: jest.fn().mockResolvedValue(validationResult),
});

describe('Action Executor Registry', () => {
  describe('Registry Operations', () => {
    it('should have executors registered for all standard action types', () => {
      const expectedTypes = ['click', 'type', 'navigate', 'scroll', 'select', 'keypress', 'focus', 'hover', 'blur'];

      for (const type of expectedTypes) {
        expect(hasActionExecutor(type as RecordedAction['actionType'])).toBe(true);
      }
    });

    it('should return registered action types', () => {
      const types = getRegisteredActionTypes();

      expect(types).toContain('click');
      expect(types).toContain('type');
      expect(types).toContain('navigate');
      expect(types.length).toBeGreaterThanOrEqual(9);
    });

    it('should return executor for registered action type', () => {
      const executor = getActionExecutor('click');

      expect(executor).toBeDefined();
      expect(typeof executor).toBe('function');
    });

    it('should return undefined for unregistered action type', () => {
      const executor = getActionExecutor('unknown-type' as RecordedAction['actionType']);

      expect(executor).toBeUndefined();
    });
  });

  describe('Click Executor', () => {
    it('should execute click action successfully', async () => {
      const page = createMockPage();
      const action = createBaseAction({ actionType: 'click' });
      const context = createContext(page);
      const baseResult = createBaseResult();

      const executor = getActionExecutor('click')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.click).toHaveBeenCalledWith('[data-testid="test-button"]', { timeout: 5000 });
    });

    it('should return error when selector is missing', async () => {
      const page = createMockPage();
      const action = createBaseAction({ actionType: 'click', selector: undefined });
      const context = createContext(page);
      const baseResult = createBaseResult();

      const executor = getActionExecutor('click')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.message).toContain('missing selector');
      expect(result.error?.code).toBe('UNKNOWN');
    });

    it('should return SELECTOR_NOT_FOUND when element not found', async () => {
      const page = createMockPage();
      const action = createBaseAction({ actionType: 'click' });
      const context = createContext(page, { valid: false, matchCount: 0, selector: '[data-testid="test-button"]' });
      const baseResult = createBaseResult();

      const executor = getActionExecutor('click')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('SELECTOR_NOT_FOUND');
      expect(result.error?.matchCount).toBe(0);
    });

    it('should return SELECTOR_AMBIGUOUS when multiple elements found', async () => {
      const page = createMockPage();
      const action = createBaseAction({ actionType: 'click' });
      const context = createContext(page, { valid: false, matchCount: 3, selector: '[data-testid="test-button"]' });
      const baseResult = createBaseResult();

      const executor = getActionExecutor('click')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('SELECTOR_AMBIGUOUS');
      expect(result.error?.matchCount).toBe(3);
    });
  });

  describe('Type Executor', () => {
    it('should execute type action successfully', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'type',
        payload: { text: 'Hello, World!' },
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'type' as const };

      const executor = getActionExecutor('type')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.fill).toHaveBeenCalledWith('[data-testid="test-button"]', 'Hello, World!', { timeout: 5000 });
    });

    it('should handle empty text', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'type',
        payload: {},
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'type' as const };

      const executor = getActionExecutor('type')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.fill).toHaveBeenCalledWith('[data-testid="test-button"]', '', { timeout: 5000 });
    });

    it('should return error when selector is missing', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'type',
        selector: undefined,
        payload: { text: 'test' },
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'type' as const };

      const executor = getActionExecutor('type')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.message).toContain('missing selector');
    });
  });

  describe('Navigate Executor', () => {
    it('should execute navigate action with targetUrl', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'navigate',
        payload: { targetUrl: 'https://example.com/page' },
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'navigate' as const };

      const executor = getActionExecutor('navigate')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.goto).toHaveBeenCalledWith('https://example.com/page', { timeout: 5000, waitUntil: 'networkidle' });
    });

    it('should fall back to action url when targetUrl not provided', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'navigate',
        url: 'https://example.com/fallback',
        payload: {},
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'navigate' as const };

      const executor = getActionExecutor('navigate')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.goto).toHaveBeenCalledWith('https://example.com/fallback', expect.any(Object));
    });

    it('should return NAVIGATION_FAILED on error', async () => {
      const page = createMockPage();
      page.goto = jest.fn().mockRejectedValue(new Error('Network error'));
      const action = createBaseAction({
        actionType: 'navigate',
        payload: { targetUrl: 'https://example.com/page' },
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'navigate' as const };

      const executor = getActionExecutor('navigate')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('NAVIGATION_FAILED');
      expect(result.error?.message).toContain('Network error');
    });

    it('should return error when URL is missing', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'navigate',
        url: '',
        payload: {},
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'navigate' as const };

      const executor = getActionExecutor('navigate')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.message).toContain('missing URL');
    });
  });

  describe('Scroll Executor', () => {
    it('should execute window scroll when no selector', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'scroll',
        selector: undefined,
        payload: { scrollX: 0, scrollY: 500 },
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'scroll' as const };

      const executor = getActionExecutor('scroll')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.evaluate).toHaveBeenCalledWith('window.scrollTo(0, 500)');
    });

    it('should execute element scroll when selector provided', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'scroll',
        selector: { primary: '.scroll-container', candidates: [] },
        payload: { scrollX: 100, scrollY: 200 },
      });
      const context = createContext(page, { valid: true, matchCount: 1, selector: '.scroll-container' });
      const baseResult = { ...createBaseResult(), actionType: 'scroll' as const };

      const executor = getActionExecutor('scroll')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.evaluate).toHaveBeenCalled();
    });

    it('should return error when element selector not found', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'scroll',
        selector: { primary: '.scroll-container', candidates: [] },
        payload: { scrollX: 0, scrollY: 500 },
      });
      const context = createContext(page, { valid: false, matchCount: 0, selector: '.scroll-container' });
      const baseResult = { ...createBaseResult(), actionType: 'scroll' as const };

      const executor = getActionExecutor('scroll')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('SELECTOR_NOT_FOUND');
    });
  });

  describe('Select Executor', () => {
    it('should execute select action successfully', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'select',
        payload: { value: 'option-1' },
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'select' as const };

      const executor = getActionExecutor('select')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.selectOption).toHaveBeenCalledWith('[data-testid="test-button"]', 'option-1', { timeout: 5000 });
    });

    it('should return error when selector is missing', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'select',
        selector: undefined,
        payload: { value: 'option-1' },
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'select' as const };

      const executor = getActionExecutor('select')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.message).toContain('missing selector');
    });
  });

  describe('Keypress Executor', () => {
    it('should execute keypress action successfully', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'keypress',
        payload: { key: 'Enter' },
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'keypress' as const };

      const executor = getActionExecutor('keypress')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.keyboard.press).toHaveBeenCalledWith('Enter');
    });

    it('should return error when key is missing', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'keypress',
        payload: {},
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'keypress' as const };

      const executor = getActionExecutor('keypress')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.message).toContain('missing key');
    });
  });

  describe('Focus Executor', () => {
    it('should execute focus action successfully', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'focus',
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'focus' as const };

      const executor = getActionExecutor('focus')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.focus).toHaveBeenCalledWith('[data-testid="test-button"]', { timeout: 5000 });
    });

    it('should return error when selector is missing', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'focus',
        selector: undefined,
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'focus' as const };

      const executor = getActionExecutor('focus')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.message).toContain('missing selector');
    });
  });

  describe('Hover Executor', () => {
    it('should execute hover action successfully', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'hover',
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'hover' as const };

      const executor = getActionExecutor('hover')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(true);
      expect(page.hover).toHaveBeenCalledWith('[data-testid="test-button"]', { timeout: 5000 });
    });

    it('should return SELECTOR_NOT_FOUND when element not found', async () => {
      const page = createMockPage();
      const action = createBaseAction({ actionType: 'hover' });
      const context = createContext(page, { valid: false, matchCount: 0, selector: '[data-testid="test-button"]' });
      const baseResult = { ...createBaseResult(), actionType: 'hover' as const };

      const executor = getActionExecutor('hover')!;
      const result = await executor(action, context, baseResult);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('SELECTOR_NOT_FOUND');
    });
  });

  describe('Blur Executor', () => {
    it('should execute blur action successfully (no-op)', async () => {
      const page = createMockPage();
      const action = createBaseAction({
        actionType: 'blur',
      });
      const context = createContext(page);
      const baseResult = { ...createBaseResult(), actionType: 'blur' as const };

      const executor = getActionExecutor('blur')!;
      const result = await executor(action, context, baseResult);

      // Blur is a no-op, just returns success
      expect(result.success).toBe(true);
    });
  });

  describe('Custom Executor Registration', () => {
    it('should allow registering custom executor', () => {
      const customExecutor = jest.fn().mockResolvedValue({ success: true });

      // Note: This would override existing executor in the global registry
      // In production, you'd want to use a scoped registry or clear after test
      const originalExecutor = getActionExecutor('click');

      registerActionExecutor('click', customExecutor);

      const executor = getActionExecutor('click');
      expect(executor).toBe(customExecutor);

      // Restore original
      if (originalExecutor) {
        registerActionExecutor('click', originalExecutor);
      }
    });
  });
});
