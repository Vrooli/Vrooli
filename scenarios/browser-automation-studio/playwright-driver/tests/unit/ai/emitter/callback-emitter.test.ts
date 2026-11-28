import { createCallbackEmitter } from '../../../../src/ai/emitter/callback-emitter';
import type { NavigationStep } from '../../../../src/ai/vision-agent/types';
import { fetchJsonResponse, installFetchMock } from '../../../helpers';

const mockFetch = installFetchMock();

describe('callback emitter inference receipts', () => {
  beforeEach(() => mockFetch.mockReset());

  it('forwards the Gateway receipt without changing exact money strings', async () => {
    mockFetch.mockResolvedValueOnce(fetchJsonResponse({ received: true }));
    const step: NavigationStep = {
      navigationId: 'nav-1',
      stepNumber: 1,
      action: { type: 'click' },
      reasoning: 'Click the button',
      screenshot: Buffer.from('png'),
      currentUrl: 'https://example.com/',
      tokensUsed: { promptTokens: 12, completionTokens: 4, totalTokens: 16 },
      inferenceReceipt: {
        rootOperationId: 'operation-1',
        source: 'vrooli-credits',
        provider: 'provider-a',
        model: 'model-a',
        providerCallsMade: 1,
        inputTokens: 12,
        outputTokens: 4,
        costMicros: '90071992547409931',
        creditsCharged: '17',
      },
      durationMs: 100,
      goalAchieved: false,
    };

    await createCallbackEmitter({ retries: 0 }).emit(step, 'http://localhost/callback');

    const event = JSON.parse(mockFetch.mock.calls[0][1]?.body as string);
    expect(event.inferenceReceipt).toEqual(step.inferenceReceipt);
  });
});
