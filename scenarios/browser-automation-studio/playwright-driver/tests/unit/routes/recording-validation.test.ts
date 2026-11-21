import { handleReplayPreview, handleValidateSelector } from '../../../src/routes/record-mode/recording-validation';
import type { SessionManager } from '../../../src/session';
import type { TimelineEntry } from '../../../src/proto/recording';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';

type PipelineManagerStub = {
  validateSelector: jest.Mock;
  replayPreview: jest.Mock;
};

function createSessionManager(pipelineManager?: PipelineManagerStub): SessionManager {
  return {
    getSession: jest.fn(() => ({ pipelineManager }) as unknown as ReturnType<SessionManager['getSession']>),
  } as unknown as SessionManager;
}

function createTimelineEntry(overrides?: Partial<TimelineEntry>): TimelineEntry {
  return {
    id: 'entry-1',
    sequenceNum: 1,
    action: { type: 2 },
    ...overrides,
  } as TimelineEntry;
}

describe('recording validation routes', () => {
  const config = createTestConfig({
    execution: {
      replayActionTimeoutMs: 12_345,
    },
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('handleValidateSelector', () => {
    it('requires a selector before calling the pipeline manager', async () => {
      const pipelineManager = {
        validateSelector: jest.fn(),
        replayPreview: jest.fn(),
      };
      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/session-123/record/validate-selector',
        body: {},
      });
      const res = createMockHttpResponse();

      await handleValidateSelector(req, res, 'session-123', createSessionManager(pipelineManager), config);

      expect(res.statusCode).toBe(400);
      expect(res.getJSON()).toEqual({
        error: 'MISSING_SELECTOR',
        message: 'selector field is required',
      });
      expect(pipelineManager.validateSelector).not.toHaveBeenCalled();
    });

    it('maps pipeline validation results to the public API response', async () => {
      const pipelineManager = {
        validateSelector: jest.fn().mockResolvedValue({
          valid: false,
          matchCount: 0,
          selector: '#missing',
          error: 'No elements matched selector',
        }),
        replayPreview: jest.fn(),
      };
      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/session-123/record/validate-selector',
        body: { selector: '#missing' },
      });
      const res = createMockHttpResponse();

      await handleValidateSelector(req, res, 'session-123', createSessionManager(pipelineManager), config);

      expect(pipelineManager.validateSelector).toHaveBeenCalledWith('#missing');
      expect(res.statusCode).toBe(200);
      expect(res.getJSON()).toEqual({
        valid: false,
        match_count: 0,
        selector: '#missing',
        error: 'No elements matched selector',
      });
    });
  });

  describe('handleReplayPreview', () => {
    it('requires a non-empty entries array before calling the pipeline manager', async () => {
      const pipelineManager = {
        validateSelector: jest.fn(),
        replayPreview: jest.fn(),
      };
      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/session-123/record/replay-preview',
        body: { entries: [] },
      });
      const res = createMockHttpResponse();

      await handleReplayPreview(req, res, 'session-123', createSessionManager(pipelineManager), config);

      expect(res.statusCode).toBe(400);
      expect(res.getJSON()).toEqual({
        error: 'MISSING_ENTRIES',
        message: 'entries field is required and must be a non-empty array',
      });
      expect(pipelineManager.replayPreview).not.toHaveBeenCalled();
    });

    it('passes default replay options from config and maps failures to snake_case', async () => {
      const entry = createTimelineEntry();
      const pipelineManager = {
        validateSelector: jest.fn(),
        replayPreview: jest.fn().mockResolvedValue({
          success: false,
          totalActions: 1,
          passedActions: 0,
          failedActions: 1,
          results: [
            {
              entryId: 'entry-1',
              sequenceNum: 1,
              actionType: 2,
              success: false,
              durationMs: 42,
              error: {
                message: 'Selector not found',
                code: 'SELECTOR_NOT_FOUND',
                matchCount: 0,
                selector: '#missing',
              },
              screenshotOnError: 'data:image/png;base64,abc',
            },
          ],
          totalDurationMs: 42,
          stoppedEarly: true,
        }),
      };
      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/session-123/record/replay-preview',
        body: { entries: [entry], limit: 3 },
      });
      const res = createMockHttpResponse();

      await handleReplayPreview(req, res, 'session-123', createSessionManager(pipelineManager), config);

      expect(pipelineManager.replayPreview).toHaveBeenCalledWith({
        entries: [entry],
        limit: 3,
        stopOnFailure: true,
        actionTimeout: 12_345,
      });
      expect(res.statusCode).toBe(200);
      expect(res.getJSON()).toEqual({
        success: false,
        total_actions: 1,
        passed_actions: 0,
        failed_actions: 1,
        results: [
          {
            entry_id: 'entry-1',
            sequence_num: 1,
            action_type: 2,
            success: false,
            duration_ms: 42,
            error: {
              message: 'Selector not found',
              code: 'SELECTOR_NOT_FOUND',
              match_count: 0,
              selector: '#missing',
            },
            screenshot_on_error: 'data:image/png;base64,abc',
          },
        ],
        total_duration_ms: 42,
        stopped_early: true,
      });
    });

    it('honors explicit replay stop and timeout options', async () => {
      const entries = [createTimelineEntry({ id: 'entry-2', sequenceNum: 2 })];
      const pipelineManager = {
        validateSelector: jest.fn(),
        replayPreview: jest.fn().mockResolvedValue({
          success: true,
          totalActions: 1,
          passedActions: 1,
          failedActions: 0,
          results: [
            {
              entryId: 'entry-2',
              sequenceNum: 2,
              actionType: 2,
              success: true,
              durationMs: 8,
            },
          ],
          totalDurationMs: 8,
          stoppedEarly: false,
        }),
      };
      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/session-123/record/replay-preview',
        body: {
          entries,
          stop_on_failure: false,
          action_timeout: 250,
        },
      });
      const res = createMockHttpResponse();

      await handleReplayPreview(req, res, 'session-123', createSessionManager(pipelineManager), config);

      expect(pipelineManager.replayPreview).toHaveBeenCalledWith({
        entries,
        limit: undefined,
        stopOnFailure: false,
        actionTimeout: 250,
      });
      expect(res.statusCode).toBe(200);
      expect(res.getJSON()).toMatchObject({
        success: true,
        total_actions: 1,
        passed_actions: 1,
        failed_actions: 0,
        stopped_early: false,
      });
    });
  });
});
