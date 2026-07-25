import { composePatches } from '../../../src/browser-profile/patches';

describe('deterministic audio anti-detection composition', () => {
  const settings = { patch_audio_context: true } as never;
  const fingerprint = {} as never;

  it('excludes random AnalyserNode noise for fake microphone sessions', () => {
    expect(composePatches(settings, fingerprint, { deterministicAudio: true }))
      .not.toContain('AnalyserNode');
  });

  it('retains random AnalyserNode noise for ordinary sessions', () => {
    expect(composePatches(settings, fingerprint)).toContain('AnalyserNode');
  });
});
