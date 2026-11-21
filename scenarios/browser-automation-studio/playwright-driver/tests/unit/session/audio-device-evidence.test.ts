import { describe, expect, it } from '@jest/globals';

import { verifyBrowserCaptureDevice } from '../../../src/session/audio/device-evidence';
import type { Page } from 'rebrowser-playwright';

describe('PipeWire device evidence', () => {
  it('retries a device probe when the qualification page navigates during evaluation', async () => {
    const evaluate = jest
      .fn()
      .mockRejectedValueOnce(new Error('page.evaluate: Execution context was destroyed, most likely because of a navigation'))
      .mockResolvedValueOnce({
        enumerated: true,
        selectedLabel: 'Vrooli Qualification Microphone',
        sampleRate: 48_000,
        channelCount: 1,
      });
    const page = {
      evaluate,
      waitForLoadState: jest.fn().mockResolvedValue(undefined),
    } as unknown as Page;

    await expect(verifyBrowserCaptureDevice(page, 'Vrooli_Qualification_Microphone')).resolves.toEqual({
      enumerated: true,
      selectedLabel: 'Vrooli Qualification Microphone',
      sampleRate: 48_000,
      channelCount: 1,
    });
    expect(evaluate).toHaveBeenCalledTimes(2);
  });
});
