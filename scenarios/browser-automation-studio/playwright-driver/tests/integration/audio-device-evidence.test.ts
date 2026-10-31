import { describe, expect, it } from '@jest/globals';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { createPipeWireQualificationDevice } from '../../src/session/audio/device-evidence';

if (process.env.VROOLI_DEVICE_EVIDENCE !== '1') {
  throw new Error('Host-audio qualification requires explicit VROOLI_DEVICE_EVIDENCE=1 and host-audio admission');
}

describe('PipeWire device evidence', () => {
  it('creates and tears down a real host capture device', async () => {
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
});
