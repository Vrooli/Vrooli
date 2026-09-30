import { describe, expect, it } from '@jest/globals';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';

import { createPipeWireQualificationDevice, verifyBrowserCaptureDevice } from '../../../src/session/audio/device-evidence';
import type { Page } from 'rebrowser-playwright';

describe('PipeWire device evidence', () => {
  const deviceTest = process.env.VROOLI_DEVICE_EVIDENCE === '1' ? it : it.skip;
  deviceTest('creates and tears down a real host capture device', async () => {
    const device = await createPipeWireQualificationDevice();
    try {
      expect(device.evidence.strategy).toBe('host_device');
      expect(device.evidence.sourceVisible).toBe(true);
      expect(device.evidence.sinkVisible).toBe(true);
      const fixture = resolve(process.cwd(), '../../audio-tools/bas/fixtures/dictation-reference.wav');
      if (existsSync(fixture)) {
        const stopPlayback = device.startWavLoop(fixture);
        await new Promise((resolve) => setTimeout(resolve, 100));
        await stopPlayback();
      }
    } finally {
      await device.close();
    }
  });

  const optInNotice = process.env.VROOLI_DEVICE_EVIDENCE === '1' ? it.skip : it;
  optInNotice('is opt-in because it changes host audio topology for the test duration', () => {
    expect(true).toBe(true);
  });

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
