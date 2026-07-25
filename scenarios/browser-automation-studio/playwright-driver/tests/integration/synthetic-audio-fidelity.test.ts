import { resolve } from 'path';
import * as http from 'http';
import { chromium } from 'rebrowser-playwright';
import { getAudioLaunchArgs, generateSilentSinkPatch } from '../../src/session/audio';

jest.setTimeout(60_000);

const fixture = resolve(__dirname, '../../../../audio-tools/bas/fixtures/dictation-reference.wav');

type Sample = { clockRate: number; peak: number; rms: number[] };
let origin = '';
let server: http.Server;

async function captureFakeMicrophone(): Promise<Sample> {
  const browser = await chromium.launch({
    headless: true,
    args: [
      ...getAudioLaunchArgs('synthetic_sink'),
      '--use-fake-device-for-media-stream',
      '--use-fake-ui-for-media-stream',
      `--use-file-for-fake-audio-capture=${fixture}`,
    ],
  });
  const context = await browser.newContext();
  await context.addInitScript(generateSilentSinkPatch());
  const page = await context.newPage();
  try {
    await page.goto(origin);
    await context.grantPermissions(['microphone'], { origin });
    return await page.evaluate(async (): Promise<Sample> => {
      const audio = new AudioContext();
      await audio.resume();
      const startedAt = performance.now();
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const source = audio.createMediaStreamSource(stream);
      const processor = audio.createScriptProcessor(1024, 1, 1);
      const silentGain = audio.createGain();
      silentGain.gain.value = 0;
      let peak = 0;
      const rms: number[] = [];
      const clockStart = audio.currentTime;
      processor.onaudioprocess = (event): void => {
        const samples = event.inputBuffer.getChannelData(0);
        let sum = 0;
        for (const sample of samples) {
          peak = Math.max(peak, Math.abs(sample));
          sum += sample * sample;
        }
        rms.push(Math.sqrt(sum / samples.length));
      };
      source.connect(processor);
      processor.connect(silentGain);
      silentGain.connect(audio.destination);
      await new Promise((resolve) => window.setTimeout(resolve, 2000));
      const elapsedSeconds = (performance.now() - startedAt) / 1000;
      const clockRate = (audio.currentTime - clockStart) / elapsedSeconds;
      stream.getTracks().forEach((track) => track.stop());
      source.disconnect();
      processor.disconnect();
      silentGain.disconnect();
      await audio.close();
      return { clockRate, peak, rms };
    });
  } finally {
    await page.close().catch(() => undefined);
    await context.close().catch(() => undefined);
    await browser.close().catch(() => undefined);
  }
}

describe('synthetic silent-sink audio fidelity', () => {
  beforeAll(async (): Promise<void> => {
    server = http.createServer((_req, res) => res.end('audio fixture origin'));
    await new Promise<void>((resolveServer) => server.listen(0, '127.0.0.1', resolveServer));
    const address = server.address();
    if (!address || typeof address === 'string')
      throw new Error('failed to allocate audio fixture origin');
    origin = `http://127.0.0.1:${address.port}`;
  });

  afterAll(async (): Promise<void> => {
    await new Promise<void>((resolveServer, rejectServer) =>
      server.close((error) => (error ? rejectServer(error) : resolveServer()))
    );
  });

  it('delivers the WAV at real-time pace and preserves its peak across repeated captures', async () => {
    const captures = [
      await captureFakeMicrophone(),
      await captureFakeMicrophone(),
      await captureFakeMicrophone(),
    ];

    for (const capture of captures) {
      expect(capture.clockRate).toBeGreaterThanOrEqual(0.9);
      expect(capture.peak).toBeGreaterThanOrEqual(0.4712);
      expect(capture.peak).toBeLessThanOrEqual(0.4912);
      expect(capture.rms.length).toBeGreaterThan(0);
    }

    expect(captures.map((capture) => capture.rms)).toEqual([
      captures[0].rms,
      captures[0].rms,
      captures[0].rms,
    ]);
  });
});
