import { afterEach, describe, expect, it, vi } from 'vitest';
import { RecordingApiService } from './RecordingApiService';

vi.mock('@/api/visionNavigation', () => ({ visionNavigationClient: {} }));
vi.mock('@/config', () => ({ getApiBase: () => 'http://fixture.invalid' }));

const receipt = {
  recording_id: '9c45d4a0-5333-4b36-8197-bdba6b6c8f36',
  session_id: 'record-session',
  started_at: '2026-09-22T12:00:00.123Z',
};
const response = (data: unknown, status = 200) => new Response(JSON.stringify(data), { status });

describe('recording receipt fidelity [REQ:BAS-RH-J24]', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('accepts the canonical completed receipt without losing identity or time', async () => {
    const terminal = { recording_id: receipt.recording_id, session_id: receipt.session_id, action_count: 7, completed_at: '2026-09-22T12:01:00.456Z' };
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(terminal)));
    expect(await new RecordingApiService().stopRecording(receipt.session_id)).toEqual({ success: true, data: terminal });
  });

  it('accepts the protobuf default when a completed recording has zero actions', async () => {
    const terminal = { recording_id: receipt.recording_id, session_id: receipt.session_id, completed_at: '2026-09-22T12:01:00Z' };
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(terminal)));
    expect(await new RecordingApiService().stopRecording(receipt.session_id)).toEqual({ success: true, data: { ...terminal, action_count: 0 } });
  });

  it('restores only the authoritative active recording after a known conflict', async () => {
    const fetch = vi.fn().mockResolvedValueOnce(response({ code: 'RECORDING_IN_PROGRESS' }, 409))
      .mockResolvedValueOnce(response({ ...receipt, is_recording: true, action_count: 7 }));
    vi.stubGlobal('fetch', fetch);
    const signal = new AbortController().signal;
    expect(await new RecordingApiService().startRecording(receipt.session_id, { signal })).toEqual({ success: true, data: receipt });
    expect(fetch).toHaveBeenCalledTimes(2);
    expect(fetch).toHaveBeenLastCalledWith('http://fixture.invalid/recordings/live/record-session/status', { signal });
  });

  it.each([{ code: 'RECORDING_START_SUPERSEDED' }, {}, { message: 'Unrelated conflict' }])('rejects an unknown conflict without inventing success: %j', async (error) => {
    const fetch = vi.fn().mockResolvedValue(response(error, 409)); vi.stubGlobal('fetch', fetch);
    expect((await new RecordingApiService().startRecording(receipt.session_id)).success).toBe(false);
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it.each([
    { ...receipt, is_recording: false },
    { session_id: receipt.session_id, is_recording: true },
    { ...receipt, session_id: 'another-session', is_recording: true },
    { ...receipt, recording_id: '', is_recording: true },
  ])('rejects absent or contradictory recovery evidence: %j', async (status) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(response({ code: 'RECORDING_IN_PROGRESS' }, 409)).mockResolvedValueOnce(response(status)));
    expect((await new RecordingApiService().startRecording(receipt.session_id)).success).toBe(false);
  });
});
