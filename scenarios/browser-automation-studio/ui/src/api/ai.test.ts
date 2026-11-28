import { describe, expect, it } from 'vitest';
import { ConnectError, Code } from '@connectrpc/connect';
import { aiRequestErrorMessage, newGatewayRootOperationID } from './ai';

describe('newGatewayRootOperationID', () => {
  it('returns a fresh 128-bit hexadecimal id compatible with AI Gateway', () => {
    const first = newGatewayRootOperationID();
    const second = newGatewayRootOperationID();

    expect(first).toMatch(/^[a-f0-9]{32}$/);
    expect(second).toMatch(/^[a-f0-9]{32}$/);
    expect(second).not.toBe(first);
  });
});

describe('aiRequestErrorMessage', () => {
  it('explains no-route and a safe next action without implying entitlement failure', () => {
    const error = new ConnectError('gateway no route', Code.FailedPrecondition, {
      'X-Vrooli-AI-Gateway-Error-Code': 'INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE',
      'X-Vrooli-AI-Source': 'local',
    });
    expect(aiRequestErrorMessage(error, 'fallback')).toBe(
      'No local route is eligible for this request. In Settings → Confidential data sharing, choose a configured remote source and grant consent, or ask the account owner to enable or upgrade an eligible route. No inference was dispatched, and funding did not change.',
    );
  });

  it('uses the existing source and settings owners for a safe next step', () => {
    const messageFor = (source: string) => aiRequestErrorMessage(
      new ConnectError('gateway no route', Code.FailedPrecondition, {
        'X-Vrooli-AI-Gateway-Error-Code': 'INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE',
        'X-Vrooli-AI-Source': source,
      }),
      'fallback',
    );

    expect(messageFor('own-key')).toContain('Check the key and route in Settings → AI Settings');
    expect(messageFor('own-key')).toContain('No inference was dispatched, and funding did not change.');
    expect(messageFor('vrooli-credits')).toContain('ask the account owner to enable or upgrade an eligible route');
    expect(messageFor('vrooli-credits')).toContain('choose your configured OpenRouter source');
    expect(messageFor('unknown')).toContain('Settings → Confidential data sharing');
  });

  it('leaves entitlement and unrelated failures unchanged', () => {
    const entitlement = new ConnectError('credits required', Code.FailedPrecondition, {
      'X-Vrooli-AI-Gateway-Error-Code': 'INFERENCE_ERROR_CODE_INSUFFICIENT_CREDITS',
    });
    expect(aiRequestErrorMessage(entitlement, 'fallback')).toBe('[failed_precondition] credits required');
    expect(aiRequestErrorMessage(new Error('network failed'), 'fallback')).toBe('network failed');
    expect(aiRequestErrorMessage(null, 'fallback')).toBe('fallback');
  });
});
