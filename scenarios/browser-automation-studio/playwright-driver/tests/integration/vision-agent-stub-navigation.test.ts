/**
 * The vision agent drives a real Chromium page through a stubbed model. This is a
 * driver-level check; the BAS-RH-J11 journey (API route with a stubbed model) is pending
 * because the driver reads its model gateway only from AI_GATEWAY_URL at startup.
 */
import { chromium } from 'rebrowser-playwright';
import { createActionExecutor } from '../../src/ai/action';
import { createElementAnnotator, createScreenshotCapture } from '../../src/ai/screenshot';
import { createNoopLogger, createVisionAgent, type VisionAgentDeps, type VisionAnalysisRequestInterface } from '../../src/ai/vision-agent';

const PAGE = '<button id="counter" onclick="document.querySelector(\'#count\').textContent=String(Number(document.querySelector(\'#count\').textContent)+1)">Click fixture</button><span id="count">0</span>';
const usage = { promptTokens: 1, completionTokens: 1, totalTokens: 2 };

describe('vision agent with a stubbed model on a real browser', () => {
  it('given an authorized task, when the stub picks the labelled control, then the task reaches the model and the effect happens once', async () => {
    const browser = await chromium.launch({ headless: true });
    try {
      const page = await browser.newPage();
      await page.goto(`data:text/html,${encodeURIComponent(PAGE)}`, { waitUntil: 'load' });
      const requests: VisionAnalysisRequestInterface[] = [];
      const visionClient: VisionAgentDeps['visionClient'] = {
        analyze(request) {
          requests.push(request);
          if (requests.length > 1) return Promise.resolve({ action: { type: 'done', success: true, result: 'clicked' }, reasoning: 'done', goalAchieved: true, confidence: 1, tokensUsed: usage });
          const label = request.elementLabels?.find(({ text }) => text?.includes('Click fixture'));
          if (!label) return Promise.reject(new Error('Stub model did not receive the fixture control'));
          return Promise.resolve({ action: { type: 'click', elementId: label.id }, reasoning: 'authorized click', goalAchieved: false, confidence: 1, tokensUsed: usage });
        },
        getModelSpec: () => ({ id: 'stub', displayName: 'Stub', provider: 'mock', supportsComputerUse: false, supportsElementLabels: true, recommended: true, tier: 'mock' }),
      };
      const agent = createVisionAgent({
        visionClient,
        screenshotCapture: createScreenshotCapture(),
        annotator: createElementAnnotator(),
        actionExecutor: createActionExecutor({ actionTimeout: 5000 }),
        stepEmitter: { emit: () => Promise.resolve() },
        logger: createNoopLogger(),
      }, { stepDelayMs: 0, postActionSettleMs: 0 });

      const result = await agent.navigate({
        prompt: 'Click the fixture button once to record the authorized action.', page, maxSteps: 3, model: 'stub',
        callbackUrl: '', navigationId: 'stub-navigation', effectPolicy: 'explicit', onStep: () => Promise.resolve(),
      });

      expect(result.status).toBe('completed');
      expect(requests[0]?.goal).toContain('authorized action');
      expect(requests[0]?.conversationHistory.some(({ content }) => content.includes('Goal:'))).toBe(true);
      expect(await page.locator('#count').textContent()).toBe('1');
    } finally {
      await browser.close();
    }
  }, 30000);
});
