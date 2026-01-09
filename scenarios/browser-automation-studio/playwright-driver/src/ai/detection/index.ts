/**
 * Detection Module
 *
 * Exports detection utilities for identifying challenges that require human intervention.
 */

export {
  detectCaptcha,
  createMockCaptchaDetector,
  NO_CAPTCHA_RESULT,
  type CaptchaDetectionResult,
} from './captcha-detector';
