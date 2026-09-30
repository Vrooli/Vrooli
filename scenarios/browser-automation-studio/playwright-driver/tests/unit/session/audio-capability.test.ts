import { detectHostAudioCapability, generateSilentSinkPatch, measureRealtimeAudio, selectAudioStrategy } from '../../../src/session/audio';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import ts from 'typescript';

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
    // Follow the decision module's actual imports/exports. Optional qualification
    // devices are separate consumers, not dependencies of strategy selection.
    const pending = [path.join(audioSourceDirectory, 'index.ts')];
    const visited = new Set<string>();
    while (pending.length) {
      const file = pending.pop()!;
      if (visited.has(file)) continue;
      visited.add(file);
      const source = readFileSync(file, 'utf8');
      expect(source).not.toMatch(/process\.platform|pipewire|pactl|wpctl|darwin|node:child_process/i);
      for (const imported of ts.preProcessFile(source).importedFiles) {
        if (!imported.fileName.startsWith('.')) continue;
        const resolved = ts.resolveModuleName(imported.fileName, file, {}, ts.sys).resolvedModule;
        expect(resolved).toBeDefined();
        pending.push(resolved!.resolvedFileName);
      }
    }
  });
});
