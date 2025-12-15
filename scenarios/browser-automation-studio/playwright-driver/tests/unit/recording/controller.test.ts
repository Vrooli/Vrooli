/**
 * RecordModeController Unit Tests
 *
 * Tests the core recording functionality including:
 * - Start/stop recording lifecycle
 * - Event normalization
 * - Selector validation
 * - State management
 */

import { RecordModeController, createRecordModeController } from '../../../src/recording/controller';
import { ActionType, type TimelineEntry } from '../../../src/recording/types';
import type { RawBrowserEvent, RawSelectorSet, RawElementMeta } from '../../../src/recording/types';
import { createMockPage } from '../../helpers';

// Helper to create a minimal valid RawBrowserEvent
function createRawEvent(overrides: Partial<RawBrowserEvent> = {}): RawBrowserEvent {
  const defaultSelector: RawSelectorSet = {
    primary: 'button',
    candidates: [],
  };

  const defaultElementMeta: RawElementMeta = {
    tagName: 'button',
    isVisible: true,
    isEnabled: true,
  };

  return {
    actionType: 'click',
    timestamp: Date.now(),
    url: 'https://example.com',
    selector: defaultSelector,
    elementMeta: defaultElementMeta,
    ...overrides,
  };
}

describe('RecordModeController', () => {
  let mockPage: ReturnType<typeof createMockPage>;
  let controller: RecordModeController;
  const sessionId = 'test-session-123';

  beforeEach(() => {
    mockPage = createMockPage();
    // Set up the page mock to support exposeFunction
    mockPage.exposeFunction = jest.fn().mockResolvedValue(undefined);
    mockPage.evaluate = jest.fn().mockResolvedValue(undefined);
    mockPage.on = jest.fn();
    mockPage.off = jest.fn();

    controller = createRecordModeController(mockPage, sessionId);
  });

  describe('createRecordModeController', () => {
    it('should create a controller instance', () => {
      expect(controller).toBeInstanceOf(RecordModeController);
    });

    it('should initialize with isRecording = false', () => {
      expect(controller.isRecording()).toBe(false);
    });

    it('should initialize with correct session ID', () => {
      const state = controller.getState();
      expect(state.sessionId).toBe(sessionId);
    });

    it('should initialize with actionCount = 0', () => {
      const state = controller.getState();
      expect(state.actionCount).toBe(0);
    });
  });

  describe('startRecording', () => {
    it('should start recording and return recording ID', async () => {
      const onEntry = jest.fn();
      const recordingId = await controller.startRecording({ sessionId, onEntry });

      expect(recordingId).toBeDefined();
      expect(typeof recordingId).toBe('string');
      expect(controller.isRecording()).toBe(true);
    });

    it('should use provided recording ID if specified', async () => {
      const onEntry = jest.fn();
      const customId = 'custom-recording-id';
      const recordingId = await controller.startRecording({
        sessionId,
        onEntry,
        recordingId: customId,
      });

      expect(recordingId).toBe(customId);
    });

    it('should expose __recordAction function to page', async () => {
      const onEntry = jest.fn();
      await controller.startRecording({ sessionId, onEntry });

      expect(mockPage.exposeFunction).toHaveBeenCalledWith(
        '__recordAction',
        expect.any(Function)
      );
    });

    it('should inject recording script into page', async () => {
      const onEntry = jest.fn();
      await controller.startRecording({ sessionId, onEntry });

      expect(mockPage.evaluate).toHaveBeenCalled();
    });

    it('should register load event handler for re-injection', async () => {
      const onEntry = jest.fn();
      await controller.startRecording({ sessionId, onEntry });

      expect(mockPage.on).toHaveBeenCalledWith('load', expect.any(Function));
    });

    it('should throw if already recording', async () => {
      const onEntry = jest.fn();
      await controller.startRecording({ sessionId, onEntry });

      await expect(controller.startRecording({ sessionId, onEntry })).rejects.toThrow(
        'Recording already in progress'
      );
    });

    it('should set startedAt timestamp', async () => {
      const onEntry = jest.fn();
      await controller.startRecording({ sessionId, onEntry });

      const state = controller.getState();
      expect(state.startedAt).toBeDefined();
    });
  });

  describe('stopRecording', () => {
    it('should stop recording and return result', async () => {
      const onEntry = jest.fn();
      await controller.startRecording({ sessionId, onEntry });

      const result = await controller.stopRecording();

      expect(result.recordingId).toBeDefined();
      // Initial navigation action is captured when recording starts (if page URL is not about:blank)
      expect(result.actionCount).toBe(1);
      expect(controller.isRecording()).toBe(false);
    });

    it('should remove load event handler', async () => {
      const onEntry = jest.fn();
      await controller.startRecording({ sessionId, onEntry });
      await controller.stopRecording();

      expect(mockPage.off).toHaveBeenCalledWith('load', expect.any(Function));
    });

    it('should run cleanup script', async () => {
      const onEntry = jest.fn();
      await controller.startRecording({ sessionId, onEntry });

      // Reset evaluate mock to track cleanup call
      mockPage.evaluate.mockClear();
      await controller.stopRecording();

      expect(mockPage.evaluate).toHaveBeenCalled();
    });

    it('should throw if not recording', async () => {
      await expect(controller.stopRecording()).rejects.toThrow(
        'No recording in progress'
      );
    });

    it('should reset state after stop', async () => {
      const onEntry = jest.fn();
      await controller.startRecording({ sessionId, onEntry });
      await controller.stopRecording();

      const state = controller.getState();
      expect(state.isRecording).toBe(false);
      expect(state.actionCount).toBe(0);
      expect(state.recordingId).toBeUndefined();
    });

    it('should cancel pending injection timeouts on stop', async () => {
      // This tests the temporal flow fix: pending timeouts should be cancelled
      // when recording stops to prevent work after stop
      const onEntry = jest.fn();
      const clearTimeoutSpy = jest.spyOn(global, 'clearTimeout');

      await controller.startRecording({ sessionId, onEntry });

      // Simulate navigation load event which triggers setTimeout for injection
      const loadHandler = (mockPage.on as jest.Mock).mock.calls.find(
        ([event]: [string, unknown]) => event === 'load'
      )?.[1] as (() => void) | undefined;

      if (loadHandler) {
        loadHandler();
      }

      // Stop recording - this should cancel any pending timeouts
      await controller.stopRecording();

      // Verify clearTimeout was called (at least once for the initial delay timeout)
      expect(clearTimeoutSpy).toHaveBeenCalled();

      clearTimeoutSpy.mockRestore();
    });
  });

  describe('validateSelector', () => {
    it('should validate CSS selector', async () => {
      mockPage.locator = jest.fn().mockReturnValue({
        count: jest.fn().mockResolvedValue(1),
      });

      const result = await controller.validateSelector('button.submit');

      expect(result.valid).toBe(true);
      expect(result.matchCount).toBe(1);
      expect(result.selector).toBe('button.submit');
    });

    it('should report invalid for multiple matches', async () => {
      mockPage.locator = jest.fn().mockReturnValue({
        count: jest.fn().mockResolvedValue(3),
      });

      const result = await controller.validateSelector('div.item');

      expect(result.valid).toBe(false);
      expect(result.matchCount).toBe(3);
    });

    it('should report invalid for no matches', async () => {
      mockPage.locator = jest.fn().mockReturnValue({
        count: jest.fn().mockResolvedValue(0),
      });

      const result = await controller.validateSelector('div.nonexistent');

      expect(result.valid).toBe(false);
      expect(result.matchCount).toBe(0);
    });

    it('should validate XPath selector', async () => {
      mockPage.evaluate = jest.fn().mockResolvedValue(1);

      const result = await controller.validateSelector('//button[@id="submit"]');

      expect(result.valid).toBe(true);
      expect(result.matchCount).toBe(1);
    });

    it('should handle XPath validation errors', async () => {
      mockPage.evaluate = jest.fn().mockResolvedValue(-1);

      const result = await controller.validateSelector('/invalid[xpath');

      expect(result.valid).toBe(false);
      expect(result.error).toBeDefined();
    });

    it('should handle CSS selector errors', async () => {
      mockPage.locator = jest.fn().mockReturnValue({
        count: jest.fn().mockRejectedValue(new Error('Invalid selector')),
      });

      const result = await controller.validateSelector('div::invalid');

      expect(result.valid).toBe(false);
      expect(result.error).toBeDefined();
    });
  });

  describe('getState', () => {
    it('should return immutable copy of state', async () => {
      const state1 = controller.getState();
      const state2 = controller.getState();

      // Should be equal but not same reference
      expect(state1).toEqual(state2);
      expect(state1).not.toBe(state2);
    });

    it('should reflect recording state changes', async () => {
      const onEntry = jest.fn();

      const stateBefore = controller.getState();
      expect(stateBefore.isRecording).toBe(false);

      await controller.startRecording({ sessionId, onEntry });

      const stateAfter = controller.getState();
      expect(stateAfter.isRecording).toBe(true);
      expect(stateAfter.recordingId).toBeDefined();
    });
  });

  describe('event handling', () => {
    let onEntry: jest.Mock;
    let exposedCallback: (event: RawBrowserEvent) => void;

    beforeEach(async () => {
      onEntry = jest.fn();

      // Capture the callback passed to exposeFunction
      mockPage.exposeFunction = jest.fn().mockImplementation((name, callback) => {
        if (name === '__recordAction') {
          exposedCallback = callback;
        }
        return Promise.resolve();
      });

      await controller.startRecording({ sessionId, onEntry });
    });

    it('should invoke onEntry callback for recorded events', () => {
      const rawEvent = createRawEvent({
        actionType: 'click',
        selector: {
          primary: 'button#submit',
          candidates: [
            { type: 'id', value: 'button#submit', confidence: 0.9, specificity: 100 },
          ],
        },
        elementMeta: {
          tagName: 'button',
          id: 'submit',
          isVisible: true,
          isEnabled: true,
        },
      });

      exposedCallback(rawEvent);

      // Call 0 is initial navigation, call 1 is our event
      const entry = onEntry.mock.calls[1][0] as TimelineEntry;
      expect(entry.action?.type).toBe(ActionType.CLICK);
      expect(entry.context?.origin.case).toBe('sessionId');
      expect(entry.context?.origin.value).toBe(sessionId);
    });

    it('should assign sequential sequence numbers', () => {
      const event1 = createRawEvent({ actionType: 'click' });
      const event2 = createRawEvent({ actionType: 'type', payload: { text: 'hello' } });

      exposedCallback(event1);
      exposedCallback(event2);

      // Note: Initial navigation action is captured when recording starts (call 0)
      // So these events are calls 1 and 2
      expect(onEntry).toHaveBeenCalledTimes(3);

      const firstEventCall = onEntry.mock.calls[1][0] as TimelineEntry;
      const secondEventCall = onEntry.mock.calls[2][0] as TimelineEntry;

      // Sequence numbers are 1 and 2 because 0 is the initial navigation
      expect(firstEventCall.sequenceNum).toBe(1);
      expect(secondEventCall.sequenceNum).toBe(2);
    });

    it('should normalize action types', () => {
      const events = [
        { actionType: 'CLICK', expected: ActionType.CLICK },
        { actionType: 'input', expected: ActionType.INPUT },
        { actionType: 'change', expected: ActionType.SELECT },
        { actionType: 'keydown', expected: ActionType.KEYBOARD },
      ];

      for (const { actionType, expected } of events) {
        onEntry.mockClear();
        exposedCallback(createRawEvent({ actionType }));

        const entry = onEntry.mock.calls[0][0] as TimelineEntry;
        expect(entry.action?.type).toBe(expected);
      }
    });

    it('should calculate confidence from selector candidates', () => {
      const rawEvent = createRawEvent({
        selector: {
          primary: '[data-testid="submit"]',
          candidates: [
            { type: 'data-testid', value: '[data-testid="submit"]', confidence: 0.98, specificity: 100 },
          ],
        },
      });

      exposedCallback(rawEvent);

      // Call 0 is initial navigation, call 1 is our event
      const entry = onEntry.mock.calls[1][0] as TimelineEntry;
      expect(entry.action?.metadata?.confidence).toBe(0.98);
    });

    it('should use default confidence when no selector candidates', () => {
      // Use 'click' action type (not 'navigate') since navigate returns 1 for no-selector case
      const rawEvent = createRawEvent({
        actionType: 'click',
        selector: { primary: '', candidates: [] },
      });

      exposedCallback(rawEvent);

      // Call 0 is initial navigation, call 1 is our event
      const entry = onEntry.mock.calls[1][0] as TimelineEntry;
      expect(entry.action?.metadata?.confidence).toBe(0.5);
    });

    it('should increment actionCount for each event', () => {
      exposedCallback(createRawEvent({ actionType: 'click' }));
      exposedCallback(createRawEvent({ actionType: 'type' }));
      exposedCallback(createRawEvent({ actionType: 'click' }));

      const state = controller.getState();
      // 1 initial navigation + 3 events = 4 total
      expect(state.actionCount).toBe(4);
    });

    it('should not process events after stop', async () => {
      await controller.stopRecording();

      onEntry.mockClear();
      exposedCallback(createRawEvent({ actionType: 'click' }));

      expect(onEntry).not.toHaveBeenCalled();
    });

    it('should call onError callback on processing errors', async () => {
      const onError = jest.fn();
      await controller.stopRecording();

      // Restart with error callback
      await controller.startRecording({ sessionId, onEntry, onError });

      // Make onEntry throw
      onEntry.mockImplementation(() => {
        throw new Error('Processing error');
      });

      // This should trigger error handling
      exposedCallback(createRawEvent({ actionType: 'click' }));

      expect(onError).toHaveBeenCalledWith(expect.any(Error));
    });
  });

  describe('re-injection on navigation', () => {
    it('should re-inject script on page load', async () => {
      let loadHandler: () => void;
      mockPage.on = jest.fn().mockImplementation((event, handler) => {
        if (event === 'load') {
          loadHandler = handler;
        }
      });

      await controller.startRecording({ sessionId, onEntry: jest.fn() });
      mockPage.evaluate.mockClear();

      // Simulate page load
      loadHandler!();

      // Wait for setTimeout in handler
      await new Promise((resolve) => setTimeout(resolve, 150));

      expect(mockPage.evaluate).toHaveBeenCalled();
    });

    it('should not re-inject after stop', async () => {
      let loadHandler: () => void;
      mockPage.on = jest.fn().mockImplementation((event, handler) => {
        if (event === 'load') {
          loadHandler = handler;
        }
      });

      await controller.startRecording({ sessionId, onEntry: jest.fn() });
      await controller.stopRecording();
      mockPage.evaluate.mockClear();

      // Simulate page load after stop
      loadHandler!();

      // Wait for setTimeout
      await new Promise((resolve) => setTimeout(resolve, 150));

      // Should not have re-injected
      expect(mockPage.evaluate).not.toHaveBeenCalled();
    });
  });
});
