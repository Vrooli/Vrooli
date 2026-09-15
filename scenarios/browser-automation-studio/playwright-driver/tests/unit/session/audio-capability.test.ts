import { detectHostAudioCapability, generateSilentSinkPatch, measureRealtimeAudio, selectAudioStrategy } from '../../../src/session/audio';
import { readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';

const audioSourceDirectory = path.resolve(__dirname, '../../../src/session/audio');

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
      finding: 'realtime_audio_host_output_unavailable: host audio capability was not recorded',
    });
  });

  it.each([
    ['device_available', 'host_device'],
    ['no_device', 'synthetic_sink'],
    ['detection_failed', 'synthetic_sink'],
  ] as const)('selects %s as %s', (outcome, strategy) => {
    expect(selectAudioStrategy({ outcome, currentTimeDelta: 0, durationMs: 1, reason: 'test' })).toBe(strategy);
  });

  it('builds a silent sink patch without discarding caller options', () => {
    const patch = generateSilentSinkPatch();
    expect(patch).toContain("sinkId: { type: 'none' }");
    expect(patch).toContain("setSinkId({ type: 'none' })");
    expect(patch).toContain('...(options || {})');
    expect(patch).toContain('webkitAudioContext');
  });

  it('bounds a blocked host-output probe and reports no_device', async () => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date('2026-01-01T00:00:00Z'));
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
      evaluate: jest.fn().mockReturnValue(new Promise(() => undefined)),
      close: jest.fn().mockResolvedValue(undefined),
    };
    const context = { newPage: jest.fn().mockResolvedValue(page), close: jest.fn().mockResolvedValue(undefined) };
    const probe = detectHostAudioCapability({ newContext: jest.fn().mockResolvedValue(context) } as never);
    await jest.advanceTimersByTimeAsync(1200);
    await expect(probe).resolves.toMatchObject({ outcome: 'no_device', durationMs: 1200 });
    jest.useRealTimers();
  });

  it('does not encode host or platform assumptions in the audio decision path', () => {
    const source = readdirSync(audioSourceDirectory)
      .filter((file) => file.endsWith('.ts'))
      .map((file) => readFileSync(path.join(audioSourceDirectory, file), 'utf8'))
      .join('\n');
    expect(source).not.toMatch(/process\.platform|pipewire|pactl|wpctl|darwin/i);
  });
});
