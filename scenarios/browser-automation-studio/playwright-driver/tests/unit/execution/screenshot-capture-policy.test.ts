import { ScreenshotCapturePolicy } from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';

import { shouldCaptureStepScreenshot } from '../../../src/execution/instruction-executor';

/**
 * The capture policy is the only thing standing between a validation suite and
 * ~400MB of PNGs it never looks at, so it is worth pinning precisely. The cases
 * that matter most are the two that must NEVER regress: a failing step always
 * yields a frame, and an API that sends no directive keeps the old behavior.
 */
describe('shouldCaptureStepScreenshot', () => {
  describe('ON_FAILURE', () => {
    it('captures when the step failed', () => {
      expect(
        shouldCaptureStepScreenshot(ScreenshotCapturePolicy.ON_FAILURE, false)
      ).toBe(true);
    });

    it('skips when the step succeeded', () => {
      expect(
        shouldCaptureStepScreenshot(ScreenshotCapturePolicy.ON_FAILURE, true)
      ).toBe(false);
    });
  });

  describe('ALWAYS', () => {
    it.each([true, false])('captures regardless of outcome (succeeded=%s)', (ok) => {
      expect(shouldCaptureStepScreenshot(ScreenshotCapturePolicy.ALWAYS, ok)).toBe(true);
    });
  });

  describe('NEVER', () => {
    it('skips a successful step', () => {
      expect(shouldCaptureStepScreenshot(ScreenshotCapturePolicy.NEVER, true)).toBe(false);
    });

    // NEVER is an explicit "status only" request, so it outranks the
    // failure-evidence rule that ON_FAILURE relies on.
    it('skips even a failed step', () => {
      expect(shouldCaptureStepScreenshot(ScreenshotCapturePolicy.NEVER, false)).toBe(false);
    });
  });

  describe('backward compatibility', () => {
    // An API build that predates the directive sends nothing. Defaulting to
    // "skip" there would silently strip replay frames from product executions.
    it.each([true, false])('captures when no directive is sent (succeeded=%s)', (ok) => {
      expect(shouldCaptureStepScreenshot(undefined, ok)).toBe(true);
    });

    it.each([true, false])('captures on UNSPECIFIED (succeeded=%s)', (ok) => {
      expect(
        shouldCaptureStepScreenshot(ScreenshotCapturePolicy.UNSPECIFIED, ok)
      ).toBe(true);
    });
  });
});
