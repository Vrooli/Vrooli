import { describe, expect, it } from 'vitest';
import { create } from '@bufbuild/protobuf';
import { AIInferenceReceiptSchema } from '@vrooli/proto-types/browser-automation-studio/v1/ai/ai_pb';
import { parseNavigationEvent, statusStepToAINavigationStep } from './navigationEvents';

const receiptWire = {
  rootOperationId: 'operation-1',
  billingOperationId: 'lpbs-op-123',
  source: 'vrooli-credits',
  provider: 'provider-a',
  model: 'model-a',
  providerCallsMade: 1,
  inputTokens: 12,
  outputTokens: 4,
  costMicros: '90071992547409931',
  creditsCharged: '17',
  accountingStatus: 'settled',
};

describe('navigation inference receipt projection', () => {
  it('preserves exact usage strings from live WebSocket events', () => {
    const event = parseNavigationEvent({
      type: 'ai_navigation_step',
      navigationId: 'nav-1',
      sessionId: 'session-1',
      stepNumber: 1,
      action: { type: 'click' },
      tokensUsed: { promptTokens: 12, completionTokens: 4, totalTokens: 16 },
      inferenceReceipt: receiptWire,
    });

    expect(event).not.toBeNull();
    if (!event || event.type !== 'ai_navigation_step') throw new Error('step event was not parsed');
    expect(event.inferenceReceipt?.costMicros).toBe('90071992547409931');
    expect(event.inferenceReceipt?.creditsCharged).toBe('17');
    expect(event.inferenceReceipt?.billingOperationId).toBe('lpbs-op-123');
  });

  it('retains the Gateway receipt when recovering steps from status', () => {
    const step = statusStepToAINavigationStep({
      index: 1,
      actionType: 'click',
      selector: '#submit',
      value: '',
      url: 'https://example.com',
      description: 'Click submit',
      success: true,
      error: '',
      inferenceReceipt: create(AIInferenceReceiptSchema, {
        rootOperationId: 'operation-1',
        billingOperationId: 'lpbs-op-123',
        source: 'vrooli-credits',
        inputTokens: '12',
        outputTokens: '4',
        costMicros: '90071992547409931',
        creditsCharged: '17',
      }),
    });

    expect(step.inferenceReceipt?.costMicros).toBe('90071992547409931');
    expect(step.inferenceReceipt?.billingOperationId).toBe('lpbs-op-123');
  });
});
