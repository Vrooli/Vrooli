import { measureRealtimeAudio } from '../../../src/session/audio-capability';

describe('measureRealtimeAudio', () => {
  it('reports available only when the clock and render callbacks advance', async () => {
    const page = { evaluate: jest.fn().mockResolvedValue({
      supported: true, currentTimeDelta: 1.9, callbackCount: 20, outputLatency: 0.03, state: 'running',
    }) };
    await expect(measureRealtimeAudio(page as never)).resolves.toMatchObject({ available: true });
  });

  it('names the silent real-time audio capability gap', async () => {
    const page = { evaluate: jest.fn().mockResolvedValue({
      supported: true, currentTimeDelta: 0, callbackCount: 0, outputLatency: 0, state: 'running',
    }) };
    await expect(measureRealtimeAudio(page as never)).resolves.toMatchObject({
      available: false,
      finding: 'realtime_audio_unavailable: audio clock or render callbacks did not advance',
    });
  });
});
