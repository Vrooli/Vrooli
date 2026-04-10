import type { Page } from 'rebrowser-playwright';
import {
  detectCaptcha,
  createMockCaptchaDetector,
  NO_CAPTCHA_RESULT,
  type CaptchaDetectionResult,
} from '../../../../src/ai/detection';

describe('CAPTCHA detector', () => {
  it('returns no detection when page.evaluate yields no results', async () => {
    const page = { evaluate: jest.fn().mockResolvedValue([]) } as unknown as Page;

    const result = await detectCaptcha(page);

    expect(result).toEqual({
      detected: false,
      type: null,
      confidence: 'low',
      reason: 'No CAPTCHA detected',
      instructions: '',
    });
  });

  it('selects the highest-confidence detection result', async () => {
    const results = [
      { type: 'generic_verification', confidence: 'low', matchedPattern: 'text: verify', selector: undefined },
      { type: 'recaptcha', confidence: 'high', matchedPattern: 'selector: .g-recaptcha', selector: '.g-recaptcha' },
    ];
    const page = { evaluate: jest.fn().mockResolvedValue(results) } as unknown as Page;

    const result = await detectCaptcha(page);

    expect(result.detected).toBe(true);
    expect(result.type).toBe('recaptcha');
    expect(result.confidence).toBe('high');
    expect(result.reason).toContain('reCAPTCHA');
    expect(result.selector).toBe('.g-recaptcha');
  });

  it('returns a safe fallback when no detection rules exist for a type', async () => {
    const results = [
      { type: 'unknown_captcha', confidence: 'high', matchedPattern: 'selector: .mystery', selector: '.mystery' },
    ];
    const page = { evaluate: jest.fn().mockResolvedValue(results) } as unknown as Page;

    const result = await detectCaptcha(page);

    expect(result.detected).toBe(false);
    expect(result.reason).toContain('No detection rules found for type');
    expect(result.selector).toBe('.mystery');
  });

  it('returns a safe result when evaluation throws', async () => {
    const page = { evaluate: jest.fn().mockRejectedValue(new Error('boom')) } as unknown as Page;

    const result = await detectCaptcha(page);

    expect(result.detected).toBe(false);
    expect(result.reason).toContain('Detection error');
  });

  it('creates a mock detector with a fixed result', async () => {
    const mockResult: CaptchaDetectionResult = {
      detected: true,
      type: 'hcaptcha',
      confidence: 'medium',
      reason: 'hCaptcha detected',
      instructions: 'Solve the challenge',
      selector: '.h-captcha',
    };

    const detector = createMockCaptchaDetector(mockResult);
    const result = await detector.detect({} as Page);

    expect(result).toEqual(mockResult);
  });

  it('exports NO_CAPTCHA_RESULT as a stable baseline', () => {
    expect(NO_CAPTCHA_RESULT.detected).toBe(false);
    expect(NO_CAPTCHA_RESULT.reason).toBe('No CAPTCHA detected');
  });
});
