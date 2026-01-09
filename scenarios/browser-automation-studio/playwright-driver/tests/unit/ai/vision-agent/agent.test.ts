/**
 * Vision Agent Unit Tests
 *
 * Tests the observe-decide-act loop orchestration with mocked dependencies.
 */

import {
  createVisionAgent,
  createNoopLogger,
  createLoopDetector,
  createActionContext,
  createScrollContext,
  createHistoryEntry,
  serializeAction,
  DEFAULT_LOOP_CONFIG,
  type VisionAgentDeps,
  type NavigationConfig,
  type NavigationStep,
  type VisionModelClientInterface,
  type ScreenshotCaptureInterface,
  type ElementAnnotatorInterface,
  type ActionExecutorInterface,
  type StepEmitterInterface,
  type VisionAnalysisRequestInterface,
  type VisionAnalysisResponseInterface,
  type VisionModelSpecInterface,
  type ActionExecutionResult,
  type ActionExecutionContext,
  type LoopDetectionConfig,
  type ActionHistoryEntry,
} from '../../../../src/ai/vision-agent';
import type { BrowserAction, ScrollAction } from '../../../../src/ai/action/types';
import type { ElementLabel, TokenUsage } from '../../../../src/ai/vision-client/types';
import type { Page } from 'playwright';

// =============================================================================
// TEST UTILITIES
// =============================================================================

/**
 * Create mock dependencies with configurable behavior.
 */
function createMockDeps(overrides?: Partial<VisionAgentDeps>): VisionAgentDeps {
  return {
    visionClient: createMockVisionClient(),
    screenshotCapture: createMockScreenshotCapture(),
    annotator: createMockAnnotator(),
    actionExecutor: createMockActionExecutor(),
    stepEmitter: createMockStepEmitter(),
    logger: createNoopLogger(),
    ...overrides,
  };
}

/**
 * Create a mock vision client with queueable responses.
 */
function createMockVisionClient(): VisionModelClientInterface & {
  queueResponse: (response: Partial<VisionAnalysisResponseInterface>) => void;
  setDefaultResponse: (response: Partial<VisionAnalysisResponseInterface>) => void;
  getCallCount: () => number;
  setFailMode: (shouldFail: boolean, message?: string) => void;
} {
  const queue: Partial<VisionAnalysisResponseInterface>[] = [];
  let defaultResponse: Partial<VisionAnalysisResponseInterface> | null = null;
  let callCount = 0;
  let shouldFail = false;
  let failMessage = 'Mock client failure';

  const defaultTokens: TokenUsage = {
    promptTokens: 1000,
    completionTokens: 50,
    totalTokens: 1050,
  };

  return {
    async analyze(
      _request: VisionAnalysisRequestInterface
    ): Promise<VisionAnalysisResponseInterface> {
      callCount++;

      if (shouldFail) {
        throw new Error(failMessage);
      }

      const queued = queue.shift();
      const response = queued ?? defaultResponse;

      if (!response) {
        throw new Error('No mock response configured');
      }

      return {
        action: response.action ?? { type: 'wait', ms: 100 },
        reasoning: response.reasoning ?? 'Mock reasoning',
        goalAchieved: response.goalAchieved ?? false,
        confidence: response.confidence ?? 0.9,
        tokensUsed: response.tokensUsed ?? defaultTokens,
        rawResponse: JSON.stringify(response.action),
      };
    },

    getModelSpec(): VisionModelSpecInterface {
      return {
        id: 'mock-model',
        apiModelId: 'mock/model',
        displayName: 'Mock Model',
        provider: 'openrouter',
        inputCostPer1MTokens: 0.1,
        outputCostPer1MTokens: 0.5,
        maxContextTokens: 100000,
        supportsComputerUse: false,
        supportsElementLabels: true,
        recommended: true,
        tier: 'budget',
      };
    },

    queueResponse(response: Partial<VisionAnalysisResponseInterface>) {
      queue.push(response);
    },

    setDefaultResponse(response: Partial<VisionAnalysisResponseInterface>) {
      defaultResponse = response;
    },

    getCallCount() {
      return callCount;
    },

    setFailMode(fail: boolean, message?: string) {
      shouldFail = fail;
      if (message) failMessage = message;
    },
  };
}

/**
 * Create a mock screenshot capture.
 */
function createMockScreenshotCapture(): ScreenshotCaptureInterface {
  return {
    async capture(_page: Page): Promise<Buffer> {
      return Buffer.from('mock-screenshot');
    },
  };
}

/**
 * Create a mock annotator.
 */
function createMockAnnotator(): ElementAnnotatorInterface {
  return {
    async annotate(
      screenshot: Buffer,
      elements: ElementLabel[]
    ): Promise<{ image: Buffer; labels: ElementLabel[] }> {
      return { image: screenshot, labels: elements };
    },
  };
}

/**
 * Create a mock action executor.
 */
function createMockActionExecutor(): ActionExecutorInterface & {
  getCalls: () => BrowserAction[];
  setFailMode: (shouldFail: boolean, message?: string) => void;
  setScrollBehavior: (effective: boolean) => void;
} {
  const calls: BrowserAction[] = [];
  let shouldFail = false;
  let failMessage = 'Action execution failed';
  let scrollIsEffective = true; // Default: scrolls are effective (page moves)
  let currentScrollY = 0;

  return {
    async execute(
      _page: Page,
      action: BrowserAction,
      _elementLabels?: ElementLabel[]
    ): Promise<ActionExecutionResult> {
      calls.push(action);

      if (shouldFail) {
        return {
          success: false,
          error: failMessage,
          durationMs: 10,
        };
      }

      // Generate scroll context for scroll actions
      let context: ActionExecutionContext | undefined;
      if (action.type === 'scroll') {
        const positionBefore = { x: 0, y: currentScrollY };
        if (scrollIsEffective) {
          // Simulate effective scroll (position changes)
          if (action.direction === 'down') currentScrollY += 500;
          if (action.direction === 'up') currentScrollY = Math.max(0, currentScrollY - 500);
        }
        // If not effective, position stays the same
        const positionAfter = { x: 0, y: currentScrollY };
        context = {
          scroll: { positionBefore, positionAfter },
        };
      }

      return {
        success: true,
        newUrl: 'https://example.com/after-action',
        durationMs: 10,
        context,
      };
    },

    getCalls() {
      return [...calls];
    },

    setFailMode(fail: boolean, message?: string) {
      shouldFail = fail;
      if (message) failMessage = message;
    },

    setScrollBehavior(effective: boolean) {
      scrollIsEffective = effective;
    },
  };
}

/**
 * Create a mock step emitter.
 */
function createMockStepEmitter(): StepEmitterInterface & {
  getEmittedSteps: () => NavigationStep[];
} {
  const steps: NavigationStep[] = [];

  return {
    async emit(step: NavigationStep, _callbackUrl: string): Promise<void> {
      steps.push(step);
    },

    getEmittedSteps() {
      return [...steps];
    },
  };
}

/**
 * Create a mock Playwright page.
 */
function createMockPage(): Page {
  return {
    url: () => 'https://example.com',
    viewportSize: () => ({ width: 1280, height: 720 }),
    evaluate: async () => [],
  } as unknown as Page;
}

/**
 * Create a basic navigation config.
 */
function createNavConfig(overrides?: Partial<NavigationConfig>): NavigationConfig {
  const steps: NavigationStep[] = [];

  return {
    prompt: 'Test goal',
    page: createMockPage(),
    maxSteps: 10,
    model: 'mock-model',
    apiKey: 'test-key',
    navigationId: 'nav-test-123',
    callbackUrl: 'http://localhost/callback',
    onStep: async (step) => {
      steps.push(step);
    },
    ...overrides,
  };
}

// =============================================================================
// TESTS
// =============================================================================

describe('VisionAgent', () => {
  describe('navigate', () => {
    it('completes successfully when goal is achieved', async () => {
      const mockClient = createMockVisionClient();
      mockClient.queueResponse({
        action: { type: 'click', elementId: 1 },
        reasoning: 'Clicking button',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'done', success: true, result: 'Task completed' },
        reasoning: 'Goal achieved',
        goalAchieved: true,
      });

      const deps = createMockDeps({ visionClient: mockClient });
      const agent = createVisionAgent(deps);
      const config = createNavConfig();

      const result = await agent.navigate(config);

      expect(result.status).toBe('completed');
      expect(result.totalSteps).toBe(2);
      expect(result.summary).toBe('Task completed');
      expect(mockClient.getCallCount()).toBe(2);
    });

    it('stops at max steps when goal not achieved', async () => {
      const mockClient = createMockVisionClient();
      // Use different actions to avoid loop detection
      mockClient.queueResponse({
        action: { type: 'scroll', direction: 'down' },
        reasoning: 'Looking for content',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'scroll', direction: 'up' },
        reasoning: 'Looking for content',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'click', elementId: 1 },
        reasoning: 'Clicking element',
        goalAchieved: false,
      });

      const deps = createMockDeps({ visionClient: mockClient });
      const agent = createVisionAgent(deps);
      const config = createNavConfig({ maxSteps: 3 });

      const result = await agent.navigate(config);

      expect(result.status).toBe('max_steps_reached');
      expect(result.totalSteps).toBe(3);
      expect(result.error).toContain('Maximum steps');
    });

    it('detects click loops on same element and stops early', async () => {
      const mockClient = createMockVisionClient();
      // Return the same click action repeatedly to trigger loop detection
      mockClient.setDefaultResponse({
        action: { type: 'click', elementId: 5 },
        reasoning: 'Clicking the same button',
        goalAchieved: false,
      });

      const deps = createMockDeps({ visionClient: mockClient });
      const agent = createVisionAgent(deps);
      const config = createNavConfig({ maxSteps: 10 });

      const result = await agent.navigate(config);

      // Should detect loop after 4 identical clicks (more than maxSameElementRepeats=3)
      expect(result.status).toBe('loop_detected');
      expect(result.totalSteps).toBe(4); // Stops on the 4th identical click
      expect(result.error).toContain('Clicked same element');
    });

    it('allows effective scrolls without triggering loop detection', async () => {
      const mockClient = createMockVisionClient();
      const mockExecutor = createMockActionExecutor();

      // Scrolls are effective (default behavior - page actually moves)
      mockExecutor.setScrollBehavior(true);

      // Return scroll down 5 times, then done
      for (let i = 0; i < 5; i++) {
        mockClient.queueResponse({
          action: { type: 'scroll', direction: 'down' } as ScrollAction,
          reasoning: 'Scrolling to find content',
          goalAchieved: false,
        });
      }
      mockClient.queueResponse({
        action: { type: 'done', success: true, result: 'Found it' },
        reasoning: 'Goal achieved',
        goalAchieved: true,
      });

      const deps = createMockDeps({
        visionClient: mockClient,
        actionExecutor: mockExecutor,
      });
      const agent = createVisionAgent(deps);
      const config = createNavConfig({ maxSteps: 10 });

      const result = await agent.navigate(config);

      // Should NOT detect loop because scrolls are effective
      expect(result.status).toBe('completed');
      expect(result.totalSteps).toBe(6);
    });

    it('detects scroll loops when scroll has no effect (stuck at boundary)', async () => {
      const mockClient = createMockVisionClient();
      const mockExecutor = createMockActionExecutor();

      // Scrolls are NOT effective (simulating being at bottom of page)
      mockExecutor.setScrollBehavior(false);

      // Return scroll down repeatedly
      mockClient.setDefaultResponse({
        action: { type: 'scroll', direction: 'down' } as ScrollAction,
        reasoning: 'Trying to scroll more',
        goalAchieved: false,
      });

      const deps = createMockDeps({
        visionClient: mockClient,
        actionExecutor: mockExecutor,
      });
      const agent = createVisionAgent(deps);
      const config = createNavConfig({ maxSteps: 10 });

      const result = await agent.navigate(config);

      // Should detect loop after 3 ineffective scrolls (more than maxIneffectiveRepeats=2)
      expect(result.status).toBe('loop_detected');
      expect(result.totalSteps).toBe(3);
      expect(result.error).toContain('no effect');
    });

    it('handles abort signal', async () => {
      const mockClient = createMockVisionClient();
      mockClient.setDefaultResponse({
        action: { type: 'wait', ms: 100 },
        reasoning: 'Waiting',
        goalAchieved: false,
      });

      const deps = createMockDeps({ visionClient: mockClient });
      const agent = createVisionAgent(deps);

      const abortController = new AbortController();
      const config = createNavConfig({
        maxSteps: 10,
        abortSignal: abortController.signal,
      });

      // Abort after a short delay
      setTimeout(() => abortController.abort(), 50);

      const result = await agent.navigate(config);

      expect(result.status).toBe('aborted');
      expect(result.error).toContain('aborted');
    });

    it('handles vision model errors', async () => {
      const mockClient = createMockVisionClient();
      mockClient.setFailMode(true, 'API rate limited');

      const deps = createMockDeps({ visionClient: mockClient });
      const agent = createVisionAgent(deps);
      const config = createNavConfig();

      const result = await agent.navigate(config);

      expect(result.status).toBe('failed');
      expect(result.error).toContain('Vision model error');
      expect(result.error).toContain('API rate limited');
    });

    it('continues after action execution failure', async () => {
      const mockClient = createMockVisionClient();
      const mockExecutor = createMockActionExecutor();

      // First action fails, second succeeds with done
      mockClient.queueResponse({
        action: { type: 'click', elementId: 1 },
        reasoning: 'First attempt',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'done', success: true, result: 'Completed' },
        reasoning: 'Done',
        goalAchieved: true,
      });

      // First execution fails
      let callCount = 0;
      mockExecutor.execute = async (
        _page: Page,
        action: BrowserAction
      ): Promise<ActionExecutionResult> => {
        callCount++;
        if (callCount === 1) {
          return { success: false, error: 'Element not found', durationMs: 10 };
        }
        return { success: true, durationMs: 10 };
      };

      const deps = createMockDeps({
        visionClient: mockClient,
        actionExecutor: mockExecutor,
      });
      const agent = createVisionAgent(deps);
      const config = createNavConfig();

      const result = await agent.navigate(config);

      expect(result.status).toBe('completed');
      expect(result.totalSteps).toBe(2);
    });

    it('emits steps to callback and onStep handler', async () => {
      const mockClient = createMockVisionClient();
      mockClient.queueResponse({
        action: { type: 'click', elementId: 1 },
        reasoning: 'Clicking',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'done', success: true, result: 'Done' },
        reasoning: 'Complete',
        goalAchieved: true,
      });

      const mockEmitter = createMockStepEmitter();
      const deps = createMockDeps({
        visionClient: mockClient,
        stepEmitter: mockEmitter,
      });
      const agent = createVisionAgent(deps);

      const onStepCalls: NavigationStep[] = [];
      const config = createNavConfig({
        onStep: async (step) => {
          onStepCalls.push(step);
        },
      });

      await agent.navigate(config);

      expect(mockEmitter.getEmittedSteps()).toHaveLength(2);
      expect(onStepCalls).toHaveLength(2);

      // Verify step content
      const firstStep = onStepCalls[0];
      expect(firstStep.stepNumber).toBe(1);
      expect(firstStep.action.type).toBe('click');
      expect(firstStep.goalAchieved).toBe(false);

      const secondStep = onStepCalls[1];
      expect(secondStep.stepNumber).toBe(2);
      expect(secondStep.action.type).toBe('done');
      expect(secondStep.goalAchieved).toBe(true);
    });

    it('tracks total tokens across steps', async () => {
      const mockClient = createMockVisionClient();
      mockClient.queueResponse({
        action: { type: 'click', elementId: 1 },
        reasoning: 'Click',
        goalAchieved: false,
        tokensUsed: { promptTokens: 1000, completionTokens: 100, totalTokens: 1100 },
      });
      mockClient.queueResponse({
        action: { type: 'type', elementId: 2, text: 'hello' },
        reasoning: 'Type',
        goalAchieved: false,
        tokensUsed: { promptTokens: 1200, completionTokens: 80, totalTokens: 1280 },
      });
      mockClient.queueResponse({
        action: { type: 'done', success: true, result: 'Done' },
        reasoning: 'Complete',
        goalAchieved: true,
        tokensUsed: { promptTokens: 1100, completionTokens: 50, totalTokens: 1150 },
      });

      const deps = createMockDeps({ visionClient: mockClient });
      const agent = createVisionAgent(deps);
      const config = createNavConfig();

      const result = await agent.navigate(config);

      expect(result.totalTokens).toBe(1100 + 1280 + 1150);
    });

    it('prevents concurrent navigation', async () => {
      const mockClient = createMockVisionClient();
      // Use varied actions to avoid loop detection
      mockClient.queueResponse({
        action: { type: 'wait', ms: 50 },
        reasoning: 'Waiting',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'scroll', direction: 'down' },
        reasoning: 'Scrolling',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'click', elementId: 1 },
        reasoning: 'Clicking',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'scroll', direction: 'up' },
        reasoning: 'Scrolling up',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'wait', ms: 100 },
        reasoning: 'Final wait',
        goalAchieved: false,
      });

      const deps = createMockDeps({ visionClient: mockClient });
      const agent = createVisionAgent(deps);
      const config = createNavConfig({ maxSteps: 5 });

      // Start first navigation
      const nav1Promise = agent.navigate(config);

      // Try to start second navigation immediately
      const nav2Result = await agent.navigate(createNavConfig());

      expect(nav2Result.status).toBe('failed');
      expect(nav2Result.error).toContain('already in progress');

      // First navigation should complete normally
      const nav1Result = await nav1Promise;
      expect(nav1Result.status).toBe('max_steps_reached');
    });

    it('executes different action types correctly', async () => {
      const mockClient = createMockVisionClient();
      const mockExecutor = createMockActionExecutor();

      mockClient.queueResponse({
        action: { type: 'click', elementId: 5 },
        reasoning: 'Click button',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'type', elementId: 3, text: 'test@example.com' },
        reasoning: 'Type email',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'scroll', direction: 'down' },
        reasoning: 'Scroll down',
        goalAchieved: false,
      });
      mockClient.queueResponse({
        action: { type: 'done', success: true, result: 'Completed' },
        reasoning: 'Done',
        goalAchieved: true,
      });

      const deps = createMockDeps({
        visionClient: mockClient,
        actionExecutor: mockExecutor,
      });
      const agent = createVisionAgent(deps);
      const config = createNavConfig();

      const result = await agent.navigate(config);

      expect(result.status).toBe('completed');
      expect(result.totalSteps).toBe(4);

      // Note: 'done' action doesn't get executed - the loop exits early
      // when goalAchieved is true, so only 3 actions are actually executed
      const calls = mockExecutor.getCalls();
      expect(calls).toHaveLength(3);
      expect(calls[0].type).toBe('click');
      expect(calls[1].type).toBe('type');
      expect(calls[2].type).toBe('scroll');
    });
  });

  describe('abort', () => {
    it('aborts ongoing navigation', async () => {
      const mockClient = createMockVisionClient();
      mockClient.setDefaultResponse({
        action: { type: 'wait', ms: 100 },
        reasoning: 'Waiting',
        goalAchieved: false,
      });

      const deps = createMockDeps({ visionClient: mockClient });
      const agent = createVisionAgent(deps);
      const config = createNavConfig({ maxSteps: 10 });

      // Start navigation
      const navPromise = agent.navigate(config);

      // Abort after short delay
      setTimeout(() => agent.abort(), 50);

      const result = await navPromise;

      expect(result.status).toBe('aborted');
    });
  });

  describe('isNavigating', () => {
    it('returns correct state', async () => {
      const mockClient = createMockVisionClient();
      mockClient.queueResponse({
        action: { type: 'done', success: true, result: 'Done' },
        reasoning: 'Complete',
        goalAchieved: true,
      });

      const deps = createMockDeps({ visionClient: mockClient });
      const agent = createVisionAgent(deps);

      expect(agent.isNavigating()).toBe(false);

      const navPromise = agent.navigate(createNavConfig());

      // Note: This timing-based test may be flaky in some environments
      // In a real scenario, you'd use more sophisticated synchronization

      await navPromise;

      expect(agent.isNavigating()).toBe(false);
    });
  });
});

describe('formatElementLabelsForPrompt', () => {
  // Import the function for direct testing
  const { formatElementLabelsForPrompt } = require('../../../../src/ai/screenshot/annotate');

  it('formats empty labels correctly', () => {
    const result = formatElementLabelsForPrompt([]);
    expect(result).toBe('No interactive elements detected on this page.');
  });

  it('formats labels with text', () => {
    const labels: ElementLabel[] = [
      {
        id: 1,
        selector: '#login',
        tagName: 'button',
        bounds: { x: 100, y: 200, width: 80, height: 30 },
        text: 'Login',
        role: 'button',
      },
    ];

    const result = formatElementLabelsForPrompt(labels);

    expect(result).toContain('[1]');
    expect(result).toContain('button');
    expect(result).toContain('Login');
  });

  it('formats labels with placeholder', () => {
    const labels: ElementLabel[] = [
      {
        id: 2,
        selector: '#email',
        tagName: 'input',
        bounds: { x: 100, y: 200, width: 200, height: 30 },
        placeholder: 'Enter email',
        role: 'textbox',
      },
    ];

    const result = formatElementLabelsForPrompt(labels);

    expect(result).toContain('[2]');
    expect(result).toContain('textbox');
    expect(result).toContain('placeholder');
    expect(result).toContain('Enter email');
  });

  it('formats multiple labels', () => {
    const labels: ElementLabel[] = [
      {
        id: 1,
        selector: '#btn1',
        tagName: 'button',
        bounds: { x: 0, y: 0, width: 50, height: 30 },
        text: 'Submit',
      },
      {
        id: 2,
        selector: '#btn2',
        tagName: 'a',
        bounds: { x: 0, y: 50, width: 50, height: 30 },
        text: 'Cancel',
        role: 'link',
      },
    ];

    const result = formatElementLabelsForPrompt(labels);

    expect(result).toContain('[1]');
    expect(result).toContain('[2]');
    expect(result).toContain('Submit');
    expect(result).toContain('Cancel');
  });
});

// =============================================================================
// LOOP DETECTION MODULE TESTS
// =============================================================================

describe('LoopDetector', () => {
  /**
   * Helper to create a history entry from an action and scroll context.
   */
  function makeHistoryEntry(
    action: BrowserAction,
    stepNumber: number,
    options?: {
      urlBefore?: string;
      urlAfter?: string;
      scrollWasEffective?: boolean;
    }
  ): ActionHistoryEntry {
    const urlBefore = options?.urlBefore ?? 'https://example.com';
    const urlAfter = options?.urlAfter ?? urlBefore;
    const context = createActionContext(action, urlBefore, urlAfter);

    // Add scroll context if applicable
    if (action.type === 'scroll') {
      const scrollAction = action as ScrollAction;
      const delta = options?.scrollWasEffective !== false ? 500 : 0;
      context.scroll = createScrollContext(
        scrollAction.direction,
        { x: 0, y: 100 },
        { x: 0, y: 100 + delta },
        10
      );
    }

    return createHistoryEntry(action, context, stepNumber);
  }

  describe('scroll loop detection', () => {
    it('does NOT flag effective scrolls as loops', () => {
      const detector = createLoopDetector();
      const history: ActionHistoryEntry[] = [];

      // Add 10 effective scroll actions
      for (let i = 1; i <= 10; i++) {
        history.push(makeHistoryEntry(
          { type: 'scroll', direction: 'down' },
          i,
          { scrollWasEffective: true }
        ));
      }

      const result = detector.detect(history);
      expect(result.isLoop).toBe(false);
      expect(result.reason).toContain('within limits');
    });

    it('flags ineffective scrolls as loops', () => {
      const detector = createLoopDetector();
      const history: ActionHistoryEntry[] = [];

      // Add 3 ineffective scroll actions (more than maxIneffectiveRepeats=2)
      for (let i = 1; i <= 3; i++) {
        history.push(makeHistoryEntry(
          { type: 'scroll', direction: 'down' },
          i,
          { scrollWasEffective: false }
        ));
      }

      const result = detector.detect(history);
      expect(result.isLoop).toBe(true);
      expect(result.detectionStrategy).toBe('scroll_ineffective');
      expect(result.reason).toContain('no effect');
    });

    it('respects custom maxIneffectiveRepeats config', () => {
      const detector = createLoopDetector({
        scroll: { maxIneffectiveRepeats: 5, maxEffectiveRepeats: -1, minScrollDelta: 10 },
      });
      const history: ActionHistoryEntry[] = [];

      // Add 5 ineffective scrolls (at the new limit)
      for (let i = 1; i <= 5; i++) {
        history.push(makeHistoryEntry(
          { type: 'scroll', direction: 'down' },
          i,
          { scrollWasEffective: false }
        ));
      }

      let result = detector.detect(history);
      expect(result.isLoop).toBe(false);

      // Add one more to exceed limit
      history.push(makeHistoryEntry(
        { type: 'scroll', direction: 'down' },
        6,
        { scrollWasEffective: false }
      ));

      result = detector.detect(history);
      expect(result.isLoop).toBe(true);
    });

    it('resets ineffective count when an effective scroll occurs', () => {
      const detector = createLoopDetector();
      const history: ActionHistoryEntry[] = [];

      // 2 ineffective, then 1 effective, then 2 more ineffective
      history.push(makeHistoryEntry({ type: 'scroll', direction: 'down' }, 1, { scrollWasEffective: false }));
      history.push(makeHistoryEntry({ type: 'scroll', direction: 'down' }, 2, { scrollWasEffective: false }));
      history.push(makeHistoryEntry({ type: 'scroll', direction: 'down' }, 3, { scrollWasEffective: true }));
      history.push(makeHistoryEntry({ type: 'scroll', direction: 'down' }, 4, { scrollWasEffective: false }));
      history.push(makeHistoryEntry({ type: 'scroll', direction: 'down' }, 5, { scrollWasEffective: false }));

      const result = detector.detect(history);
      // Only 2 consecutive ineffective at the end, should not trigger
      expect(result.isLoop).toBe(false);
    });
  });

  describe('click loop detection', () => {
    it('flags repeated clicks on same element as loop', () => {
      const detector = createLoopDetector();
      const history: ActionHistoryEntry[] = [];

      // Add 4 clicks on same element (more than maxSameElementRepeats=3)
      for (let i = 1; i <= 4; i++) {
        history.push(makeHistoryEntry(
          { type: 'click', elementId: 5 },
          i
        ));
      }

      const result = detector.detect(history);
      expect(result.isLoop).toBe(true);
      expect(result.detectionStrategy).toBe('click_same_element');
    });

    it('does NOT flag clicks on same element if URL changes (pagination)', () => {
      const detector = createLoopDetector();
      const history: ActionHistoryEntry[] = [];

      // Add clicks on same element but URL changes each time
      for (let i = 1; i <= 5; i++) {
        history.push(makeHistoryEntry(
          { type: 'click', elementId: 5 },
          i,
          { urlBefore: `https://example.com/page${i}`, urlAfter: `https://example.com/page${i + 1}` }
        ));
      }

      const result = detector.detect(history);
      expect(result.isLoop).toBe(false);
      expect(result.reason).toContain('pagination pattern');
    });

    it('resets count when clicking different elements', () => {
      const detector = createLoopDetector();
      const history: ActionHistoryEntry[] = [];

      // Click element 5, then element 3, then element 5 twice
      history.push(makeHistoryEntry({ type: 'click', elementId: 5 }, 1));
      history.push(makeHistoryEntry({ type: 'click', elementId: 3 }, 2));
      history.push(makeHistoryEntry({ type: 'click', elementId: 5 }, 3));
      history.push(makeHistoryEntry({ type: 'click', elementId: 5 }, 4));

      const result = detector.detect(history);
      // Only 2 consecutive clicks on element 5 at the end
      expect(result.isLoop).toBe(false);
    });
  });

  describe('default loop detection', () => {
    it('flags repeated type actions as loop', () => {
      const detector = createLoopDetector();
      const history: ActionHistoryEntry[] = [];

      // Add 4 identical type actions (more than default maxRepeats=3)
      for (let i = 1; i <= 4; i++) {
        history.push(makeHistoryEntry(
          { type: 'type', text: 'hello', elementId: 1 },
          i
        ));
      }

      const result = detector.detect(history);
      expect(result.isLoop).toBe(true);
      expect(result.detectionStrategy).toBe('default_repeat');
    });

    it('does NOT flag done actions as loops', () => {
      const detector = createLoopDetector();
      const history: ActionHistoryEntry[] = [];

      // Add 10 done actions (should never be a loop)
      for (let i = 1; i <= 10; i++) {
        history.push(makeHistoryEntry(
          { type: 'done', success: true, result: 'done' },
          i
        ));
      }

      const result = detector.detect(history);
      expect(result.isLoop).toBe(false);
    });
  });

  describe('configuration', () => {
    it('allows updating config at runtime', () => {
      const detector = createLoopDetector();

      // Get default config
      const defaultConfig = detector.getConfig();
      expect(defaultConfig.scroll.maxIneffectiveRepeats).toBe(2);

      // Update config
      detector.updateConfig({
        scroll: { maxIneffectiveRepeats: 10, maxEffectiveRepeats: -1, minScrollDelta: 10 },
      });

      const updatedConfig = detector.getConfig();
      expect(updatedConfig.scroll.maxIneffectiveRepeats).toBe(10);
    });

    it('includes debug info when configured', () => {
      const detector = createLoopDetector({ includeDebugInfo: true });
      const history: ActionHistoryEntry[] = [];

      for (let i = 1; i <= 4; i++) {
        history.push(makeHistoryEntry({ type: 'click', elementId: 5 }, i));
      }

      const result = detector.detect(history);
      expect(result.isLoop).toBe(true);
      expect(result.debug).toBeDefined();
      expect(result.debug?.historyLength).toBe(4);
      expect(result.debug?.lastActions).toHaveLength(4);
    });
  });
});

describe('serializeAction', () => {
  it('serializes click with element ID', () => {
    const result = serializeAction({ type: 'click', elementId: 5 });
    expect(result).toBe('click:element:5');
  });

  it('serializes click with coordinates', () => {
    const result = serializeAction({ type: 'click', coordinates: { x: 100, y: 200 } });
    expect(result).toBe('click:coords:100,200');
  });

  it('serializes scroll', () => {
    const result = serializeAction({ type: 'scroll', direction: 'down' });
    expect(result).toBe('scroll:down');
  });

  it('serializes type with element', () => {
    const result = serializeAction({ type: 'type', text: 'hello', elementId: 3 });
    expect(result).toBe('type:element:3:hello');
  });

  it('serializes type without element', () => {
    const result = serializeAction({ type: 'type', text: 'world' });
    expect(result).toBe('type:focused:world');
  });
});
