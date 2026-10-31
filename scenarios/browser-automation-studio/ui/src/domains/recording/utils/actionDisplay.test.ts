import { describe, expect, it } from 'vitest';
import type { BrowserAction } from '../ai-navigation/types';
import {
  REDACTED_ACTION_VALUE,
  displayActionDetail,
  displayActionPreview,
  redactActionText,
  redactActionUrl,
} from './actionDisplay';

describe('action display privacy', () => {
  it('redacts value and text for sensitive selectors without changing the action', () => {
    const action: BrowserAction = {
      type: 'type',
      selector: 'input[name="password"]',
      value: 'BAS_SYNTHETIC_PASSWORD_7f3e',
      text: 'BAS_SYNTHETIC_PASSWORD_7f3e',
    };

    expect(displayActionDetail(action, 'value', action.value!)).toBe(REDACTED_ACTION_VALUE);
    expect(displayActionDetail(action, 'text', action.text!)).toBe(REDACTED_ACTION_VALUE);
    expect(displayActionPreview(action)).toBe(REDACTED_ACTION_VALUE);
    expect(action.value).toBe('BAS_SYNTHETIC_PASSWORD_7f3e');
  });

  it('redacts credential assignments and sensitive URL query values', () => {
    expect(redactActionText('submitted token=BAS_SYNTHETIC_TOKEN_91ab')).toBe('submitted token=[REDACTED]');
    const url = redactActionUrl('https://example.test/login?token=BAS_SYNTHETIC_TOKEN_91ab&page=2');
    expect(url).not.toContain('BAS_SYNTHETIC_TOKEN_91ab');
    expect(url).toContain('page=2');
    expect(url).toContain('token=%5BREDACTED%5D');
  });

  it('redacts a raw secret echoed in a sensitive action result', () => {
    const secret = 'BAS_SYNTHETIC_PASSWORD_7f3e';
    const action: BrowserAction = {
      type: 'type',
      selector: 'input[name="password"]',
      value: secret,
      text: secret,
    };

    expect(displayActionDetail(action, 'result', `provider echoed ${secret}`))
      .toBe('provider echoed [REDACTED]');
    expect(displayActionDetail(action, 'result', 'field updated')).toBe('field updated');
  });
});
