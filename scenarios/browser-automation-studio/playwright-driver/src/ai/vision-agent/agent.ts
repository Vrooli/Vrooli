/**
 * Vision Agent
 *
 * STABILITY: STABLE CORE
 *
 * This module implements the vision-based AI navigation agent.
 * It orchestrates the observe-decide-act loop:
 *
 * 1. OBSERVE: Capture screenshot and extract interactive elements
 * 2. DECIDE: Send to vision model to determine next action
 * 3. ACT: Execute the action via Playwright
 * 4. EMIT: Send step event to callback URL
 * 5. REPEAT: Until goal achieved, max steps, or abort
 *
 * DESIGN PRINCIPLES:
 * - Dependency injection for all external services (testing seams)
 * - Clear separation between orchestration and execution
 * - Graceful handling of abort signals
 * - Comprehensive step tracking and emission
 */

import type { Page } from 'playwright';
import type {
  VisionAgent,
  VisionAgentDeps,
  NavigationConfig,
  NavigationStep,
  NavigationResult,
  NavigationStatus,
  LoggerInterface,
  ConversationMessageInterface,
} from './types';
import type { BrowserAction, DoneAction } from '../action/types';
import type { TokenUsage, ElementLabel } from '../vision-client/types';
import { extractInteractiveElements, formatElementLabelsForPrompt } from '../screenshot/annotate';

/**
 * Configuration for VisionAgent behavior.
 */
export interface VisionAgentConfig {
  /** Default max steps if not specified in NavigationConfig */
  defaultMaxSteps?: number;
  /** Delay between steps in ms (for rate limiting) */
  stepDelayMs?: number;
  /** Screenshot quality (1-100) */
  screenshotQuality?: number;
  /** Whether to capture full page screenshots */
  fullPageScreenshot?: boolean;
  /** Maximum conversation history messages to retain (for memory/context efficiency) */
  maxHistoryMessages?: number;
  /** Maximum element labels to include in context (for token efficiency) */
  maxElementLabels?: number;
}

/**
 * Default configuration values.
 */
const DEFAULTS: Required<VisionAgentConfig> = {
  defaultMaxSteps: 20,
  stepDelayMs: 0,
  screenshotQuality: 80,
  fullPageScreenshot: false,
  maxHistoryMessages: 20, // Keep last 20 messages to prevent context overflow
  maxElementLabels: 50, // Limit element labels to prevent token explosion
};

/**
 * Create a VisionAgent instance.
 *
 * TESTING SEAM: Accepts all dependencies via injection for testability.
 *
 * @param deps - Injected dependencies
 * @param config - Optional configuration
 * @returns VisionAgent instance
 */
export function createVisionAgent(
  deps: VisionAgentDeps,
  config: VisionAgentConfig = {}
): VisionAgent {
  const cfg = { ...DEFAULTS, ...config };
  let abortController: AbortController | null = null;
  let isNavigating = false;

  return {
    async navigate(navConfig: NavigationConfig): Promise<NavigationResult> {
      if (isNavigating) {
        return {
          navigationId: navConfig.navigationId,
          status: 'failed',
          totalSteps: 0,
          totalTokens: 0,
          totalDurationMs: 0,
          finalUrl: '',
          error: 'Navigation already in progress',
        };
      }

      isNavigating = true;
      abortController = new AbortController();

      const startTime = Date.now();
      let totalTokens = 0;
      let stepNumber = 0;
      let status: NavigationStatus = 'completed';
      let error: string | undefined;
      let summary: string | undefined;

      // Merge abort signals if external one provided
      const signal = navConfig.abortSignal
        ? mergeAbortSignals(navConfig.abortSignal, abortController.signal)
        : abortController.signal;

      const conversationHistory: ConversationMessageInterface[] = [];
      const maxSteps = navConfig.maxSteps ?? cfg.defaultMaxSteps;

      deps.logger.info('Vision navigation started', {
        navigationId: navConfig.navigationId,
        prompt: navConfig.prompt,
        model: navConfig.model,
        maxSteps,
      });

      try {
        // Main navigation loop
        while (stepNumber < maxSteps) {
          // Check for abort
          if (signal.aborted) {
            status = 'aborted';
            error = 'Navigation aborted by user';
            deps.logger.info('Navigation aborted', { navigationId: navConfig.navigationId });
            break;
          }

          stepNumber++;
          const stepStartTime = Date.now();

          deps.logger.debug('Starting step', {
            navigationId: navConfig.navigationId,
            stepNumber,
          });

          // Step delay (for rate limiting)
          if (cfg.stepDelayMs > 0 && stepNumber > 1) {
            await delay(cfg.stepDelayMs);
          }

          // OBSERVE: Capture screenshot and extract elements
          const { screenshot, elementLabels: rawElementLabels } = await observePhase(
            navConfig.page,
            deps,
            cfg
          );

          // PERFORMANCE: Limit element labels to prevent token explosion
          const elementLabels = rawElementLabels.slice(0, cfg.maxElementLabels);

          // Get current URL
          const currentUrl = navConfig.page.url();

          // Build conversation history for context
          updateConversationHistory(
            conversationHistory,
            stepNumber,
            navConfig.prompt,
            elementLabels
          );

          // PERFORMANCE: Trim old conversation history to prevent context overflow
          trimConversationHistory(conversationHistory, cfg.maxHistoryMessages);

          // DECIDE: Get action from vision model
          let analysisResult: {
            action: BrowserAction;
            reasoning: string;
            goalAchieved: boolean;
            confidence: number;
            tokensUsed: TokenUsage;
          };

          try {
            analysisResult = await deps.visionClient.analyze({
              screenshot,
              elementLabels,
              goal: navConfig.prompt,
              conversationHistory,
              currentUrl,
            });
          } catch (e) {
            const errMsg = e instanceof Error ? e.message : String(e);
            deps.logger.error('Vision model analysis failed', {
              navigationId: navConfig.navigationId,
              stepNumber,
              error: errMsg,
            });

            // Emit error step
            const errorStep = createErrorStep(
              navConfig.navigationId,
              stepNumber,
              screenshot,
              currentUrl,
              errMsg,
              Date.now() - stepStartTime
            );

            await safeEmit(deps, errorStep, navConfig.callbackUrl, navConfig.onStep);

            status = 'failed';
            error = `Vision model error: ${errMsg}`;
            break;
          }

          totalTokens += analysisResult.tokensUsed.totalTokens;

          // Add assistant response to history
          conversationHistory.push({
            role: 'assistant',
            content: `Action: ${JSON.stringify(analysisResult.action)}\nReasoning: ${analysisResult.reasoning}`,
          });

          deps.logger.debug('Vision model decision', {
            navigationId: navConfig.navigationId,
            stepNumber,
            action: analysisResult.action.type,
            goalAchieved: analysisResult.goalAchieved,
            confidence: analysisResult.confidence,
          });

          // Check if goal achieved (done action)
          if (analysisResult.goalAchieved || analysisResult.action.type === 'done') {
            const doneAction = analysisResult.action as DoneAction;
            summary = doneAction.type === 'done' ? doneAction.result : analysisResult.reasoning;

            // Emit final step
            const finalStep: NavigationStep = {
              navigationId: navConfig.navigationId,
              stepNumber,
              action: analysisResult.action,
              reasoning: analysisResult.reasoning,
              screenshot,
              currentUrl,
              tokensUsed: analysisResult.tokensUsed,
              durationMs: Date.now() - stepStartTime,
              goalAchieved: true,
              elementLabels,
            };

            await safeEmit(deps, finalStep, navConfig.callbackUrl, navConfig.onStep);

            status = 'completed';
            deps.logger.info('Goal achieved', {
              navigationId: navConfig.navigationId,
              totalSteps: stepNumber,
              summary,
            });
            break;
          }

          // ACT: Execute the action
          const executionResult = await deps.actionExecutor.execute(
            navConfig.page,
            analysisResult.action,
            elementLabels
          );

          if (!executionResult.success) {
            deps.logger.warn('Action execution failed', {
              navigationId: navConfig.navigationId,
              stepNumber,
              action: analysisResult.action.type,
              error: executionResult.error,
            });

            // Add failure to conversation for retry context
            conversationHistory.push({
              role: 'user',
              content: `The action failed with error: ${executionResult.error}. Please try a different approach.`,
            });
          }

          // EMIT: Send step event
          const step: NavigationStep = {
            navigationId: navConfig.navigationId,
            stepNumber,
            action: analysisResult.action,
            reasoning: analysisResult.reasoning,
            screenshot,
            currentUrl,
            tokensUsed: analysisResult.tokensUsed,
            durationMs: Date.now() - stepStartTime,
            goalAchieved: false,
            error: executionResult.error,
            elementLabels,
          };

          await safeEmit(deps, step, navConfig.callbackUrl, navConfig.onStep);

          // Small delay after action to let page settle
          await delay(100);
        }

        // Check if we hit max steps
        if (stepNumber >= maxSteps && status === 'completed') {
          status = 'max_steps_reached';
          error = `Maximum steps (${maxSteps}) reached without achieving goal`;
          deps.logger.warn('Max steps reached', {
            navigationId: navConfig.navigationId,
            maxSteps,
          });
        }
      } catch (e) {
        const errMsg = e instanceof Error ? e.message : String(e);
        deps.logger.error('Navigation failed unexpectedly', {
          navigationId: navConfig.navigationId,
          error: errMsg,
        });
        status = 'failed';
        error = errMsg;
      } finally {
        isNavigating = false;
        abortController = null;
      }

      const result: NavigationResult = {
        navigationId: navConfig.navigationId,
        status,
        totalSteps: stepNumber,
        totalTokens,
        totalDurationMs: Date.now() - startTime,
        finalUrl: navConfig.page.url(),
        error,
        summary,
      };

      deps.logger.info('Navigation completed', {
        ...result,
      });

      return result;
    },

    abort(): void {
      if (abortController) {
        abortController.abort();
        deps.logger.info('Abort signal sent');
      }
    },

    isNavigating(): boolean {
      return isNavigating;
    },
  };
}

/**
 * OBSERVE phase: Capture screenshot and extract interactive elements.
 */
async function observePhase(
  page: Page,
  deps: VisionAgentDeps,
  cfg: Required<VisionAgentConfig>
): Promise<{ screenshot: Buffer; elementLabels: ElementLabel[] }> {
  // Capture screenshot
  const screenshot = await deps.screenshotCapture.capture(page, {
    quality: cfg.screenshotQuality,
    fullPage: cfg.fullPageScreenshot,
  });

  // Extract interactive elements
  const elementLabels = await extractInteractiveElements(page);

  return { screenshot, elementLabels };
}

/**
 * Update conversation history with current step context.
 */
function updateConversationHistory(
  history: ConversationMessageInterface[],
  stepNumber: number,
  prompt: string,
  elementLabels: ElementLabel[]
): void {
  // Add system prompt on first step
  if (stepNumber === 1) {
    history.push({
      role: 'system',
      content: buildSystemPrompt(),
    });

    // Add user goal
    history.push({
      role: 'user',
      content: `Goal: ${prompt}`,
    });
  }

  // Add current state as user message
  const elementContext = formatElementLabelsForPrompt(elementLabels);
  history.push({
    role: 'user',
    content: `Current state (step ${stepNumber}):\n\n${elementContext}\n\nWhat action should I take next to achieve the goal?`,
  });
}

/**
 * PERFORMANCE: Trim conversation history to prevent context overflow.
 * Keeps the system message and goal, plus the most recent messages.
 */
function trimConversationHistory(
  history: ConversationMessageInterface[],
  maxMessages: number
): void {
  if (history.length <= maxMessages) {
    return;
  }

  // Find the indices of system message and goal (usually first 2 messages)
  let systemAndGoalEnd = 0;
  for (let i = 0; i < history.length && i < 3; i++) {
    if (history[i].role === 'system') {
      systemAndGoalEnd = i + 1;
    } else if (i === 1 && history[i].role === 'user' && history[i].content.startsWith('Goal:')) {
      systemAndGoalEnd = i + 1;
    }
  }

  // Calculate how many messages to keep after system/goal
  const messagesAfterSystemGoal = maxMessages - systemAndGoalEnd;
  if (messagesAfterSystemGoal <= 0) {
    return; // Can't trim without removing system/goal
  }

  // Remove old messages, keeping system/goal at the start and recent at the end
  const messagesToRemove = history.length - maxMessages;
  if (messagesToRemove > 0) {
    // Remove from after systemAndGoalEnd
    history.splice(systemAndGoalEnd, messagesToRemove);
  }
}

/**
 * Build system prompt for the vision model.
 */
function buildSystemPrompt(): string {
  return `You are a browser automation agent. Your task is to navigate a web page to achieve the user's goal.

You can see a screenshot of the current page and a list of interactive elements with numbered labels.

For each step, analyze the page and decide on ONE action to take. Respond with:
1. Your reasoning (what you observe and why you're taking this action)
2. The action to take

Available actions:
- click(elementId) - Click on element with given ID
- click(x, y) - Click at coordinates
- type(elementId, "text") - Type text into an input element
- type("text") - Type into the currently focused element
- scroll(direction) - Scroll up/down/left/right
- navigate("url") - Go to a specific URL
- hover(elementId) - Hover over an element
- select(elementId, "value") - Select option in dropdown
- wait(ms) - Wait for specified milliseconds
- keypress("key") - Press a keyboard key
- done(success, "result") - Task is complete

Respond in this format:
REASONING: [Your analysis of what you see and why you're taking this action]
ACTION: [action_name](parameters)

Example:
REASONING: I can see a login form with email and password fields. The email field is labeled [3]. I'll click on it to focus and then enter the email.
ACTION: click(3)

Important guidelines:
- Always analyze what you see before acting
- Take one step at a time
- If you encounter an error, try a different approach
- Use done(true, "result") when the goal is achieved
- Use done(false, "reason") if the goal cannot be achieved`;
}

/**
 * Create an error step for emission.
 */
function createErrorStep(
  navigationId: string,
  stepNumber: number,
  screenshot: Buffer,
  currentUrl: string,
  error: string,
  durationMs: number
): NavigationStep {
  return {
    navigationId,
    stepNumber,
    action: { type: 'wait', ms: 0 },
    reasoning: 'Error occurred during analysis',
    screenshot,
    currentUrl,
    tokensUsed: { promptTokens: 0, completionTokens: 0, totalTokens: 0 },
    durationMs,
    goalAchieved: false,
    error,
  };
}

/**
 * Safely emit a step, catching any emission errors.
 */
async function safeEmit(
  deps: VisionAgentDeps,
  step: NavigationStep,
  callbackUrl: string,
  onStep: (step: NavigationStep) => Promise<void>
): Promise<void> {
  try {
    // Emit to callback URL
    await deps.stepEmitter.emit(step, callbackUrl);
  } catch (e) {
    deps.logger.warn('Failed to emit step to callback URL', {
      error: e instanceof Error ? e.message : String(e),
      navigationId: step.navigationId,
      stepNumber: step.stepNumber,
    });
  }

  try {
    // Call onStep callback
    await onStep(step);
  } catch (e) {
    deps.logger.warn('onStep callback failed', {
      error: e instanceof Error ? e.message : String(e),
      navigationId: step.navigationId,
      stepNumber: step.stepNumber,
    });
  }
}

/**
 * Simple delay helper.
 */
function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Merge two abort signals into one.
 */
function mergeAbortSignals(
  signal1: AbortSignal,
  signal2: AbortSignal
): AbortSignal {
  const controller = new AbortController();

  function onAbort() {
    controller.abort();
  }

  if (signal1.aborted || signal2.aborted) {
    controller.abort();
  } else {
    signal1.addEventListener('abort', onAbort, { once: true });
    signal2.addEventListener('abort', onAbort, { once: true });
  }

  return controller.signal;
}

/**
 * Create a console logger for simple use cases.
 */
export function createConsoleLogger(): LoggerInterface {
  return {
    debug(message: string, meta?: Record<string, unknown>) {
      console.debug(`[DEBUG] ${message}`, meta ?? '');
    },
    info(message: string, meta?: Record<string, unknown>) {
      console.info(`[INFO] ${message}`, meta ?? '');
    },
    warn(message: string, meta?: Record<string, unknown>) {
      console.warn(`[WARN] ${message}`, meta ?? '');
    },
    error(message: string, meta?: Record<string, unknown>) {
      console.error(`[ERROR] ${message}`, meta ?? '');
    },
  };
}

/**
 * Create a no-op logger for testing.
 */
export function createNoopLogger(): LoggerInterface {
  return {
    debug() {},
    info() {},
    warn() {},
    error() {},
  };
}
