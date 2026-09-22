import { streamRecordingEntry } from '../../../src/routes/record-mode/callback-streaming';
import { create } from '@bufbuild/protobuf';
import { TimelineEntrySchema } from '../../../src/proto/recording';
import { initRecordingBuffer, bufferTimelineEntry, getTimelineEntries, acknowledgeTimelineEntries, removeRecordingBuffer } from '../../../src/recording';
import { handleRecordNavigate, handleRecordActions, handleRecordActionsAck, handleRecordStop } from '../../../src/routes/record-mode';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';
import type { SessionManager } from '../../../src/session/manager';

// Minimal session manager stub to avoid spinning up Playwright
const mockPage = {
  on: jest.fn(),
  goto: jest.fn().mockResolvedValue(undefined),
  screenshot: jest.fn().mockResolvedValue(Buffer.from('image-bytes')),
  url: jest.fn().mockReturnValue('https://example.com'),
  title: jest.fn().mockResolvedValue('Example'),
};
const mockSessionManager: Pick<SessionManager, 'getSession'> = {
  getSession: () => ({ page: mockPage } as unknown as ReturnType<SessionManager['getSession']>),
};

describe('Record Mode Routes', () => {
  const config = createTestConfig();

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('handles navigate requests without crashing when parsing body', async () => {
    const sessionId = 'session-123';

    const mockReq = createMockHttpRequest({
      method: 'POST',
      url: `/session/${sessionId}/record/navigate`,
      body: { url: 'https://example.com', capture: true },
    });
    const mockRes = createMockHttpResponse();

    await handleRecordNavigate(mockReq, mockRes, sessionId, mockSessionManager as SessionManager, config);

    expect(mockRes.statusCode).toBe(200);
    const payload = mockRes.getJSON();
    expect(payload.url).toBe('https://example.com');
    expect(payload.screenshot).toContain('data:image/jpeg;base64,');
    expect(mockPage.goto).toHaveBeenCalledWith('https://example.com', { waitUntil: 'load', timeout: config.execution.navigationTimeoutMs });
  });
  it('requires an explicit acknowledgement after non-destructive reads', async () => {
    const sessionId = 'pull-actions';
    initRecordingBuffer(sessionId);
    bufferTimelineEntry(sessionId, create(TimelineEntrySchema, { id: 'pending', sequenceNum: 0 }));
    try {
      const read = (query = '') => {
        const res = createMockHttpResponse();
        handleRecordActions(createMockHttpRequest({ url: `/session/${sessionId}/record/actions${query}` }), res, sessionId, mockSessionManager as SessionManager);
        return res;
      };
      expect(read().getJSON().count).toBe(1);
      expect(read('?clear=true').statusCode).toBeGreaterThanOrEqual(400);
      expect(read().getJSON().count).toBe(1);
      const ack = createMockHttpResponse();
      await handleRecordActionsAck(createMockHttpRequest({ method: 'POST', body: { entry_ids: ['pending'] } }), ack, sessionId, mockSessionManager as SessionManager, config);
      expect(ack.statusCode).toBe(200);
      expect(ack.getJSON().entry_ids).toEqual(['pending']);
      expect(read().getJSON().count).toBe(0);
      const retry = createMockHttpResponse();
      await handleRecordActionsAck(createMockHttpRequest({ method: 'POST', body: { entry_ids: ['pending'] } }), retry, sessionId, mockSessionManager as SessionManager, config);
      expect(retry.statusCode).toBe(200);
    } finally {
      acknowledgeTimelineEntries(sessionId, getTimelineEntries(sessionId).map((entry) => entry.id));
      removeRecordingBuffer(sessionId);
    }
  });

  it.each([
    [503, { status: 'ok', entry_id: 'delivery' }],
    [200, { status: 'ok', entry_id: 'another-observation' }],
    [200, { status: 'ok' }],
  ])('requires the callback to acknowledge this exact observation (%s)', async (status, receipt) => {
    const entry = create(TimelineEntrySchema, { id: 'delivery' });
    const fetch = jest.spyOn(globalThis, 'fetch');
    try {
      fetch.mockImplementation(async () => new Response(JSON.stringify(receipt), { status }));
      await expect(streamRecordingEntry('http://fixture.test/commit', entry)).rejects.toThrow();
      fetch.mockImplementation(async () => new Response(JSON.stringify({ status: 'ok', entry_id: entry.id }), { status: 200 }));
      await expect(streamRecordingEntry('http://fixture.test/commit', entry)).resolves.toBeUndefined();
      expect(fetch).toHaveBeenCalledTimes(2);
    } finally { fetch.mockRestore(); }
  });

  it('returns the retained terminal receipt when stop is retried', async () => {
    const stoppedAt = '2026-09-22T00:00:00.000Z';
    const stop = jest.fn();
    const session = {
      phase: 'ready', page: mockPage,
      pipelineManager: {
        isRecording: () => false, getRecordingId: () => 'recording',
        getRecordingData: () => ({ actionCount: 7, stoppedAt }), stopRecording: stop,
      },
    };
    const manager = { getSession: () => session, setSessionPhase: jest.fn() } as unknown as SessionManager;
    const response = createMockHttpResponse();
    await handleRecordStop(createMockHttpRequest(), response, 'stopped', manager);
    expect(response.statusCode).toBe(200);
    expect(response.getJSON()).toMatchObject({ action_count: 7, recording_id: 'recording', stopped_at: stoppedAt });
    expect(stop).not.toHaveBeenCalled();
  });

});
