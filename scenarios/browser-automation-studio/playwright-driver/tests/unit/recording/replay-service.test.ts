import { create } from '@bufbuild/protobuf';
import {
  TimelineEntrySchema,
  type TimelineEntry,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import type { Page } from 'rebrowser-playwright';

jest.mock('../../../src/recording/action-executor', () => ({
  executeTimelineEntry: jest.fn(),
}));

jest.mock('../../../src/recording/validation/selector-service', () => ({
  validateSelectorOnPage: jest.fn(),
}));

import { executeTimelineEntry } from '../../../src/recording/action-executor';
import { validateSelectorOnPage } from '../../../src/recording/validation/selector-service';
import {
  ReplayPreviewService,
  createReplayPreviewService,
} from '../../../src/recording/validation/replay-service';

const mockExecuteTimelineEntry = executeTimelineEntry as jest.MockedFunction<typeof executeTimelineEntry>;
const mockValidateSelectorOnPage = validateSelectorOnPage as jest.MockedFunction<
  typeof validateSelectorOnPage
>;

const createEntry = (id: string, sequenceNum: number): TimelineEntry =>
  create(TimelineEntrySchema, { id, sequenceNum });

const createMockPage = (): jest.Mocked<Page> =>
  ({
    screenshot: jest.fn().mockResolvedValue(Buffer.from('screenshot-bytes')),
  }) as unknown as jest.Mocked<Page>;

describe('ReplayPreviewService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('replays entries and stops on failure by default', async () => {
    const page = createMockPage();
    const service = new ReplayPreviewService(page);
    const entries = [createEntry('entry-1', 1), createEntry('entry-2', 2), createEntry('entry-3', 3)];

    mockExecuteTimelineEntry
      .mockResolvedValueOnce({ success: true, durationMs: 10 })
      .mockResolvedValueOnce({ success: false, durationMs: 20 })
      .mockResolvedValueOnce({ success: true, durationMs: 30 });

    const result = await service.replayPreview({ entries });

    expect(result.success).toBe(false);
    expect(result.totalActions).toBe(2);
    expect(result.passedActions).toBe(1);
    expect(result.failedActions).toBe(1);
    expect(result.stoppedEarly).toBe(true);
    expect(result.results[1]?.screenshotOnError).toBeDefined();
  });

  it('continues replay when stopOnFailure is false', async () => {
    const page = createMockPage();
    const service = createReplayPreviewService(page);
    const entries = [createEntry('entry-1', 1), createEntry('entry-2', 2), createEntry('entry-3', 3)];

    mockExecuteTimelineEntry
      .mockResolvedValueOnce({ success: false, durationMs: 10 })
      .mockResolvedValueOnce({ success: true, durationMs: 20 })
      .mockResolvedValueOnce({ success: true, durationMs: 30 });

    const result = await service.replayPreview({ entries, stopOnFailure: false });

    expect(result.totalActions).toBe(3);
    expect(result.failedActions).toBe(1);
    expect(result.stoppedEarly).toBe(false);
  });

  it('respects limit and does not execute additional entries', async () => {
    const page = createMockPage();
    const service = new ReplayPreviewService(page);
    const entries = [createEntry('entry-1', 1), createEntry('entry-2', 2)];

    mockExecuteTimelineEntry.mockResolvedValue({ success: true, durationMs: 10 });

    const result = await service.replayPreview({ entries, limit: 1 });

    expect(result.totalActions).toBe(1);
    expect(mockExecuteTimelineEntry).toHaveBeenCalledTimes(1);
  });

  it('deduplicates concurrent replay requests with identical entries', async () => {
    const page = createMockPage();
    const service = new ReplayPreviewService(page);
    const entries = [createEntry('entry-1', 1)];

    let resolveFirst: (value: { success: boolean; durationMs: number }) => void;
    const firstResult = new Promise<{ success: boolean; durationMs: number }>((resolve) => {
      resolveFirst = resolve;
    });

    mockExecuteTimelineEntry.mockReturnValueOnce(firstResult);

    const promiseA = service.replayPreview({ entries });
    const promiseB = service.replayPreview({ entries });

    await Promise.resolve();
    expect(mockExecuteTimelineEntry).toHaveBeenCalledTimes(1);

    resolveFirst!({ success: true, durationMs: 10 });

    const [resultA, resultB] = await Promise.all([promiseA, promiseB]);

    expect(resultA.totalActions).toBe(1);
    expect(resultB.totalActions).toBe(1);
    expect(mockExecuteTimelineEntry).toHaveBeenCalledTimes(1);
  });

  it('ignores screenshot failures on action errors', async () => {
    const page = createMockPage();
    page.screenshot.mockRejectedValueOnce(new Error('screenshot failed'));
    const service = new ReplayPreviewService(page);
    const entries = [createEntry('entry-1', 1)];

    mockExecuteTimelineEntry.mockResolvedValueOnce({ success: false, durationMs: 10 });

    const result = await service.replayPreview({ entries });

    expect(result.results[0]?.success).toBe(false);
    expect(result.results[0]?.screenshotOnError).toBeUndefined();
  });

  it('delegates selector validation to selector service', async () => {
    const page = createMockPage();
    const service = new ReplayPreviewService(page);

    mockValidateSelectorOnPage.mockResolvedValue({ valid: true, matchCount: 2, selector: '.cta' });

    const result = await service.validateSelector('.cta');

    expect(result).toEqual({ valid: true, matchCount: 2, selector: '.cta' });
    expect(mockValidateSelectorOnPage).toHaveBeenCalledWith(page, '.cta');
  });
});
