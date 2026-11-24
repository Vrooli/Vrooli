import { handleSessionAudioDiagnostics, handleSessionAudioRestart, handleSessionAudioStop } from '../../../src/routes/session-audio';
import type { SessionManager } from '../../../src/session';
import type { BrowserPermissionProfile, SessionState } from '../../../src/types/session';
import { createMockHttpRequest, createMockHttpResponse } from '../../helpers';

function fixture(profile?: BrowserPermissionProfile) {
  const session = {
    spec: { permission_profile: profile },
    page: { url: () => 'https://audio.example.test/task' },
    createdAt: new Date(Date.now() - 1000),
    instructionCount: 0,
  } as unknown as SessionState;
  const manager = {
    getSessionForLease: jest.fn(() => session),
    restartAudioPlayback: jest.fn().mockResolvedValue(true),
    stopAudioPlayback: jest.fn().mockResolvedValue(true),
    readAudioDiagnostics: jest.fn().mockResolvedValue({ tracks: 1 }),
  } as unknown as SessionManager;
  return { session, manager };
}

function request(body: Record<string, unknown>) {
  return createMockHttpRequest({ method: 'POST', body });
}

describe('session audio permission routes', () => {
  it.each([
    ['restart', handleSessionAudioRestart],
    ['stop', handleSessionAudioStop],
  ] as const)('denies host playback %s before effect when control is revoked', async (_name, handler) => {
    const { manager } = fixture({
      effect: 'observe', approval: 'none', control: false, view: true,
      audio_mode: 'pipewire_qualification', url_allow: ['https://audio.example.test/*'],
    });
    const response = createMockHttpResponse();
    await handler(request({ execution_id: 'owner', lease_id: 'lease' }), response, 'session', manager);
    expect(response.statusCode).toBe(403);
    expect((manager as unknown as { restartAudioPlayback: jest.Mock }).restartAudioPlayback).not.toHaveBeenCalled();
    expect((manager as unknown as { stopAudioPlayback: jest.Mock }).stopAudioPlayback).not.toHaveBeenCalled();
  });

  it('allows an in-scope PipeWire grant and denies out-of-scope or missing audio authority before effect', async () => {
    const allowed = fixture({
      effect: 'act', approval: 'none', control: true, view: true,
      audio_mode: 'pipewire_qualification', url_allow: ['https://audio.example.test/*'],
    });
    const restart = createMockHttpResponse();
    await handleSessionAudioRestart(request({ execution_id: 'owner', lease_id: 'lease' }), restart, 'session', allowed.manager);
    expect(restart.statusCode).toBe(200);
    expect((allowed.manager as unknown as { restartAudioPlayback: jest.Mock }).restartAudioPlayback).toHaveBeenCalledTimes(1);

    const outOfScope = fixture({
      effect: 'act', approval: 'none', control: true, view: true,
      audio_mode: 'pipewire_qualification', url_allow: ['https://other.example.test/*'],
    });
    const deniedScope = createMockHttpResponse();
    await handleSessionAudioRestart(request({ execution_id: 'owner', lease_id: 'lease' }), deniedScope, 'session', outOfScope.manager);
    expect(deniedScope.statusCode).toBe(403);
    expect((outOfScope.manager as unknown as { restartAudioPlayback: jest.Mock }).restartAudioPlayback).not.toHaveBeenCalled();

    const missingCapability = fixture({
      effect: 'act', approval: 'none', control: true, view: true,
      url_allow: ['https://audio.example.test/*'],
    });
    const deniedCapability = createMockHttpResponse();
    await handleSessionAudioStop(request({ execution_id: 'owner', lease_id: 'lease' }), deniedCapability, 'session', missingCapability.manager);
    expect(deniedCapability.statusCode).toBe(403);
    expect((missingCapability.manager as unknown as { stopAudioPlayback: jest.Mock }).stopAudioPlayback).not.toHaveBeenCalled();
  });

  it('fails closed when approval cannot be supplied, and applies view authority to diagnostics', async () => {
    const approval = fixture({
      effect: 'act', approval: 'every_action', control: true, view: true,
      audio_mode: 'pipewire_qualification', url_allow: ['https://audio.example.test/*'],
    });
    const deniedApproval = createMockHttpResponse();
    await handleSessionAudioRestart(request({ execution_id: 'owner', lease_id: 'lease' }), deniedApproval, 'session', approval.manager);
    expect(deniedApproval.statusCode).toBe(403);
    expect((approval.manager as unknown as { restartAudioPlayback: jest.Mock }).restartAudioPlayback).not.toHaveBeenCalled();

    const noView = fixture({
      effect: 'act', approval: 'none', control: true, view: false,
      audio_mode: 'fake_microphone', url_allow: ['https://audio.example.test/*'],
    });
    const deniedDiagnostic = createMockHttpResponse();
    await handleSessionAudioDiagnostics(request({ execution_id: 'owner', lease_id: 'lease', surface: 'audio-tools' }), deniedDiagnostic, 'session', noView.manager);
    expect(deniedDiagnostic.statusCode).toBe(403);
    expect((noView.manager as unknown as { readAudioDiagnostics: jest.Mock }).readAudioDiagnostics).not.toHaveBeenCalled();

    const view = fixture({
      effect: 'observe', approval: 'none', control: false, view: true,
      audio_mode: 'fake_microphone', url_allow: ['https://audio.example.test/*'],
    });
    const allowedDiagnostic = createMockHttpResponse();
    await handleSessionAudioDiagnostics(request({ execution_id: 'owner', lease_id: 'lease', surface: 'audio-tools' }), allowedDiagnostic, 'session', view.manager);
    expect(allowedDiagnostic.statusCode).toBe(200);
    expect((view.manager as unknown as { readAudioDiagnostics: jest.Mock }).readAudioDiagnostics).toHaveBeenCalledTimes(1);
  });
});
