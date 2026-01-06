/**
 * State Machine Tests
 *
 * Tests for the recording pipeline state machine.
 * Verifies transition rules, verification gate, error handling, and observability.
 */

import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import {
  createRecordingStateMachine,
  recordingReducer,
  createInitialState,
  isValidTransition,
  RECOVERABLE_ERRORS,
  type RecordingPipelineState,
  type RecordingTransition,
  type PipelineVerification,
} from '../../../src/recording/state-machine';

describe('RecordingStateMachine', () => {
  describe('createInitialState', () => {
    it('should create state with uninitialized phase', () => {
      const state = createInitialState();
      expect(state.phase).toBe('uninitialized');
      expect(state.totalGenerations).toBe(0);
      expect(state.initialization).toBeUndefined();
      expect(state.verification).toBeUndefined();
      expect(state.recording).toBeUndefined();
      expect(state.error).toBeUndefined();
    });
  });

  describe('isValidTransition', () => {
    it('should allow valid transitions', () => {
      expect(isValidTransition('uninitialized', 'initializing')).toBe(true);
      expect(isValidTransition('initializing', 'verifying')).toBe(true);
      expect(isValidTransition('initializing', 'error')).toBe(true);
      expect(isValidTransition('verifying', 'ready')).toBe(true);
      expect(isValidTransition('verifying', 'error')).toBe(true);
      expect(isValidTransition('ready', 'starting')).toBe(true);
      expect(isValidTransition('starting', 'capturing')).toBe(true);
      expect(isValidTransition('capturing', 'stopping')).toBe(true);
      expect(isValidTransition('stopping', 'ready')).toBe(true);
      expect(isValidTransition('error', 'ready')).toBe(true);
      expect(isValidTransition('error', 'uninitialized')).toBe(true);
    });

    it('should reject invalid transitions', () => {
      expect(isValidTransition('uninitialized', 'capturing')).toBe(false);
      expect(isValidTransition('ready', 'capturing')).toBe(false); // Must go through 'starting'
      expect(isValidTransition('capturing', 'ready')).toBe(false); // Must go through 'stopping'
      expect(isValidTransition('verifying', 'capturing')).toBe(false);
      expect(isValidTransition('stopping', 'uninitialized')).toBe(false);
    });
  });

  describe('recordingReducer', () => {
    describe('INITIALIZE transition', () => {
      it('should transition from uninitialized to initializing', () => {
        const state = createInitialState();
        const transition: RecordingTransition = { type: 'INITIALIZE', contextId: 'ctx-123' };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('initializing');
        expect(newState.initialization).toBeDefined();
        expect(newState.initialization?.contextId).toBe('ctx-123');
        expect(newState.initialization?.injectionAttempts).toBe(0);
        expect(newState.loopDetection).toBeDefined();
        expect(newState.loopDetection?.navigationHistory).toEqual([]);
      });

      it('should reject INITIALIZE from non-uninitialized phase', () => {
        const state: RecordingPipelineState = {
          phase: 'ready',
          totalGenerations: 0,
        };
        const transition: RecordingTransition = { type: 'INITIALIZE', contextId: 'ctx-123' };

        const newState = recordingReducer(state, transition);

        expect(newState).toBe(state); // No change
      });
    });

    describe('INJECTION_COMPLETE transition', () => {
      it('should transition to verifying on success', () => {
        const state: RecordingPipelineState = {
          phase: 'initializing',
          initialization: { startedAt: Date.now(), contextId: 'ctx', injectionAttempts: 0 },
          loopDetection: { navigationHistory: [], isBreakingLoop: false, lastCheckedUrl: null },
          totalGenerations: 0,
        };
        const transition: RecordingTransition = { type: 'INJECTION_COMPLETE', success: true };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('verifying');
        expect(newState.initialization?.injectionAttempts).toBe(1);
      });

      it('should transition to error on failure', () => {
        const state: RecordingPipelineState = {
          phase: 'initializing',
          initialization: { startedAt: Date.now(), contextId: 'ctx', injectionAttempts: 0 },
          loopDetection: { navigationHistory: [], isBreakingLoop: false, lastCheckedUrl: null },
          totalGenerations: 0,
        };
        const transition: RecordingTransition = {
          type: 'INJECTION_COMPLETE',
          success: false,
          error: 'Script injection failed',
        };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('error');
        expect(newState.error?.code).toBe('INJECTION_FAILED');
        expect(newState.error?.message).toBe('Script injection failed');
        expect(newState.error?.recoverable).toBe(true);
      });
    });

    describe('VERIFICATION_COMPLETE transition', () => {
      const goodVerification: PipelineVerification = {
        scriptLoaded: true,
        scriptReady: true,
        inMainContext: true,
        handlersCount: 7,
        eventRouteActive: true,
        verifiedAt: new Date().toISOString(),
        version: '2.2.0',
      };

      it('should transition to ready on successful verification', () => {
        const state: RecordingPipelineState = {
          phase: 'verifying',
          initialization: { startedAt: Date.now(), contextId: 'ctx', injectionAttempts: 1 },
          loopDetection: { navigationHistory: [], isBreakingLoop: false, lastCheckedUrl: null },
          totalGenerations: 0,
        };
        const transition: RecordingTransition = {
          type: 'VERIFICATION_COMPLETE',
          verification: goodVerification,
        };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('ready');
        expect(newState.verification).toEqual(goodVerification);
        expect(newState.error).toBeUndefined();
      });

      it('should transition to error if script not loaded', () => {
        const state: RecordingPipelineState = {
          phase: 'verifying',
          totalGenerations: 0,
        };
        const badVerification: PipelineVerification = {
          ...goodVerification,
          scriptLoaded: false,
        };
        const transition: RecordingTransition = {
          type: 'VERIFICATION_COMPLETE',
          verification: badVerification,
        };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('error');
        expect(newState.error?.code).toBe('SCRIPT_NOT_LOADED');
      });

      it('should transition to error if wrong context', () => {
        const state: RecordingPipelineState = {
          phase: 'verifying',
          totalGenerations: 0,
        };
        const badVerification: PipelineVerification = {
          ...goodVerification,
          inMainContext: false,
        };
        const transition: RecordingTransition = {
          type: 'VERIFICATION_COMPLETE',
          verification: badVerification,
        };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('error');
        expect(newState.error?.code).toBe('WRONG_CONTEXT');
        expect(newState.error?.recoverable).toBe(false);
      });

      it('should transition to error if no handlers', () => {
        const state: RecordingPipelineState = {
          phase: 'verifying',
          totalGenerations: 0,
        };
        const badVerification: PipelineVerification = {
          ...goodVerification,
          handlersCount: 0,
        };
        const transition: RecordingTransition = {
          type: 'VERIFICATION_COMPLETE',
          verification: badVerification,
        };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('error');
        expect(newState.error?.code).toBe('NO_HANDLERS');
      });
    });

    describe('START_RECORDING transition', () => {
      it('should transition from ready to starting', () => {
        const state: RecordingPipelineState = {
          phase: 'ready',
          verification: {
            scriptLoaded: true,
            scriptReady: true,
            inMainContext: true,
            handlersCount: 7,
            eventRouteActive: true,
            verifiedAt: new Date().toISOString(),
            version: '2.2.0',
          },
          loopDetection: { navigationHistory: [], isBreakingLoop: false, lastCheckedUrl: null },
          totalGenerations: 0,
        };
        const transition: RecordingTransition = {
          type: 'START_RECORDING',
          sessionId: 'session-123',
          recordingId: 'recording-456',
        };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('starting');
        expect(newState.recording?.sessionId).toBe('session-123');
        expect(newState.recording?.recordingId).toBe('recording-456');
        expect(newState.recording?.generation).toBe(1);
        expect(newState.recording?.actionCount).toBe(0);
        expect(newState.totalGenerations).toBe(1);
      });

      it('should increment generation on each recording start', () => {
        let state: RecordingPipelineState = {
          phase: 'ready',
          totalGenerations: 5,
          loopDetection: { navigationHistory: [], isBreakingLoop: false, lastCheckedUrl: null },
        };

        state = recordingReducer(state, {
          type: 'START_RECORDING',
          sessionId: 's1',
          recordingId: 'r1',
        });

        expect(state.recording?.generation).toBe(6);
        expect(state.totalGenerations).toBe(6);
      });

      it('should reject START_RECORDING from non-ready phase (verification gate)', () => {
        const state: RecordingPipelineState = {
          phase: 'verifying',
          totalGenerations: 0,
        };
        const transition: RecordingTransition = {
          type: 'START_RECORDING',
          sessionId: 'session-123',
          recordingId: 'recording-456',
        };

        const newState = recordingReducer(state, transition);

        expect(newState).toBe(state); // No change - verification gate enforced
      });
    });

    describe('RECORDING_STARTED transition', () => {
      it('should transition from starting to capturing', () => {
        const state: RecordingPipelineState = {
          phase: 'starting',
          recording: {
            sessionId: 's1',
            recordingId: 'r1',
            startedAt: '',
            generation: 1,
            actionCount: 0,
            lastUrl: null,
          },
          totalGenerations: 1,
        };
        const startedAt = new Date().toISOString();
        const transition: RecordingTransition = { type: 'RECORDING_STARTED', startedAt };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('capturing');
        expect(newState.recording?.startedAt).toBe(startedAt);
      });
    });

    describe('ACTION_CAPTURED transition', () => {
      it('should increment action count', () => {
        const state: RecordingPipelineState = {
          phase: 'capturing',
          recording: {
            sessionId: 's1',
            recordingId: 'r1',
            startedAt: new Date().toISOString(),
            generation: 1,
            actionCount: 5,
            lastUrl: null,
          },
          totalGenerations: 1,
        };
        const transition: RecordingTransition = { type: 'ACTION_CAPTURED', actionType: 'click' };

        const newState = recordingReducer(state, transition);

        expect(newState.recording?.actionCount).toBe(6);
      });

      it('should not change state if not capturing', () => {
        const state: RecordingPipelineState = {
          phase: 'ready',
          totalGenerations: 0,
        };
        const transition: RecordingTransition = { type: 'ACTION_CAPTURED', actionType: 'click' };

        const newState = recordingReducer(state, transition);

        expect(newState).toBe(state);
      });
    });

    describe('STOP_RECORDING transition', () => {
      it('should transition from capturing to stopping', () => {
        const state: RecordingPipelineState = {
          phase: 'capturing',
          recording: {
            sessionId: 's1',
            recordingId: 'r1',
            startedAt: new Date().toISOString(),
            generation: 1,
            actionCount: 10,
            lastUrl: 'https://example.com',
          },
          totalGenerations: 1,
        };
        const transition: RecordingTransition = { type: 'STOP_RECORDING' };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('stopping');
      });
    });

    describe('RECORDING_STOPPED transition', () => {
      it('should transition from stopping to ready', () => {
        const state: RecordingPipelineState = {
          phase: 'stopping',
          recording: {
            sessionId: 's1',
            recordingId: 'r1',
            startedAt: new Date().toISOString(),
            generation: 1,
            actionCount: 10,
            lastUrl: 'https://example.com',
          },
          totalGenerations: 1,
        };
        const transition: RecordingTransition = { type: 'RECORDING_STOPPED', actionCount: 15 };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('ready');
        expect(newState.recording?.actionCount).toBe(15); // Updated final count
      });
    });

    describe('ERROR transition', () => {
      it('should transition to error from any phase', () => {
        const state: RecordingPipelineState = {
          phase: 'capturing',
          recording: {
            sessionId: 's1',
            recordingId: 'r1',
            startedAt: new Date().toISOString(),
            generation: 1,
            actionCount: 5,
            lastUrl: null,
          },
          totalGenerations: 1,
        };
        const transition: RecordingTransition = {
          type: 'ERROR',
          code: 'PAGE_CLOSED',
          message: 'Page was closed',
        };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('error');
        expect(newState.error?.code).toBe('PAGE_CLOSED');
        expect(newState.error?.previousPhase).toBe('capturing');
        expect(newState.error?.recoverable).toBe(false); // PAGE_CLOSED is not recoverable
      });

      it('should mark recoverable errors correctly', () => {
        const state: RecordingPipelineState = {
          phase: 'capturing',
          totalGenerations: 1,
        };
        const transition: RecordingTransition = {
          type: 'ERROR',
          code: 'SCRIPT_NOT_READY',
          message: 'Script not ready',
        };

        const newState = recordingReducer(state, transition);

        expect(newState.error?.recoverable).toBe(true);
      });
    });

    describe('RECOVER transition', () => {
      it('should transition from error to ready if recoverable', () => {
        const state: RecordingPipelineState = {
          phase: 'error',
          error: {
            code: 'SCRIPT_NOT_READY',
            message: 'Script not ready',
            recoverable: true,
            occurredAt: new Date().toISOString(),
            previousPhase: 'verifying',
          },
          totalGenerations: 0,
        };
        const transition: RecordingTransition = { type: 'RECOVER' };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('ready');
        expect(newState.error).toBeUndefined();
      });

      it('should not recover if not recoverable', () => {
        const state: RecordingPipelineState = {
          phase: 'error',
          error: {
            code: 'WRONG_CONTEXT',
            message: 'Wrong context',
            recoverable: false,
            occurredAt: new Date().toISOString(),
            previousPhase: 'verifying',
          },
          totalGenerations: 0,
        };
        const transition: RecordingTransition = { type: 'RECOVER' };

        const newState = recordingReducer(state, transition);

        expect(newState).toBe(state); // No change
      });
    });

    describe('RESET transition', () => {
      it('should reset to initial state', () => {
        const state: RecordingPipelineState = {
          phase: 'error',
          error: {
            code: 'WRONG_CONTEXT',
            message: 'Wrong context',
            recoverable: false,
            occurredAt: new Date().toISOString(),
            previousPhase: 'verifying',
          },
          initialization: { startedAt: Date.now(), contextId: 'ctx', injectionAttempts: 3 },
          totalGenerations: 10,
        };
        const transition: RecordingTransition = { type: 'RESET' };

        const newState = recordingReducer(state, transition);

        expect(newState.phase).toBe('uninitialized');
        expect(newState.totalGenerations).toBe(0);
        expect(newState.error).toBeUndefined();
        expect(newState.initialization).toBeUndefined();
      });
    });

    describe('NAVIGATION transition', () => {
      it('should update navigation history', () => {
        const state: RecordingPipelineState = {
          phase: 'capturing',
          recording: {
            sessionId: 's1',
            recordingId: 'r1',
            startedAt: new Date().toISOString(),
            generation: 1,
            actionCount: 5,
            lastUrl: 'https://example.com/page1',
          },
          loopDetection: {
            navigationHistory: [],
            isBreakingLoop: false,
            lastCheckedUrl: null,
          },
          totalGenerations: 1,
        };
        const transition: RecordingTransition = {
          type: 'NAVIGATION',
          url: 'https://example.com/page2',
        };

        const newState = recordingReducer(state, transition);

        expect(newState.recording?.lastUrl).toBe('https://example.com/page2');
        expect(newState.loopDetection?.navigationHistory.length).toBe(1);
        expect(newState.loopDetection?.navigationHistory[0].url).toBe('https://example.com/page2');
        expect(newState.loopDetection?.lastCheckedUrl).toBe('https://example.com/page2');
      });

      it('should limit navigation history to 50 entries', () => {
        const history = Array.from({ length: 60 }, (_, i) => ({
          url: `https://example.com/page${i}`,
          timestamp: Date.now() - i * 1000,
        }));

        const state: RecordingPipelineState = {
          phase: 'capturing',
          recording: {
            sessionId: 's1',
            recordingId: 'r1',
            startedAt: new Date().toISOString(),
            generation: 1,
            actionCount: 5,
            lastUrl: null,
          },
          loopDetection: {
            navigationHistory: history,
            isBreakingLoop: false,
            lastCheckedUrl: null,
          },
          totalGenerations: 1,
        };
        const transition: RecordingTransition = {
          type: 'NAVIGATION',
          url: 'https://example.com/new',
        };

        const newState = recordingReducer(state, transition);

        expect(newState.loopDetection?.navigationHistory.length).toBe(50);
      });
    });
  });

  describe('createRecordingStateMachine', () => {
    let machine: ReturnType<typeof createRecordingStateMachine>;

    beforeEach(() => {
      machine = createRecordingStateMachine();
    });

    it('should start in uninitialized state', () => {
      expect(machine.getState().phase).toBe('uninitialized');
      expect(machine.getPhase()).toBe('uninitialized');
    });

    it('should dispatch transitions and return new state', () => {
      const newState = machine.dispatch({ type: 'INITIALIZE', contextId: 'ctx' });

      expect(newState.phase).toBe('initializing');
      expect(machine.getState().phase).toBe('initializing');
    });

    it('should notify subscribers on state change', () => {
      const listener = jest.fn();
      machine.subscribe(listener);

      machine.dispatch({ type: 'INITIALIZE', contextId: 'ctx' });

      expect(listener).toHaveBeenCalledTimes(1);
      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({ phase: 'initializing' }),
        { type: 'INITIALIZE', contextId: 'ctx' },
        expect.objectContaining({ phase: 'uninitialized' })
      );
    });

    it('should not notify on invalid transition', () => {
      const listener = jest.fn();
      machine.subscribe(listener);

      // Try invalid transition from uninitialized
      machine.dispatch({ type: 'START_RECORDING', sessionId: 's', recordingId: 'r' });

      expect(listener).not.toHaveBeenCalled();
    });

    it('should allow unsubscribe', () => {
      const listener = jest.fn();
      const unsubscribe = machine.subscribe(listener);

      machine.dispatch({ type: 'INITIALIZE', contextId: 'ctx' });
      expect(listener).toHaveBeenCalledTimes(1);

      unsubscribe();

      machine.dispatch({ type: 'INJECTION_COMPLETE', success: true });
      expect(listener).toHaveBeenCalledTimes(1); // Not called again
    });

    it('should provide convenience methods', () => {
      expect(machine.isRecording()).toBe(false);
      expect(machine.isReady()).toBe(false);
      expect(machine.isError()).toBe(false);
      expect(machine.getRecordingId()).toBeUndefined();
      expect(machine.getGeneration()).toBe(0);

      // Go through full lifecycle
      machine.dispatch({ type: 'INITIALIZE', contextId: 'ctx' });
      machine.dispatch({ type: 'INJECTION_COMPLETE', success: true });
      machine.dispatch({
        type: 'VERIFICATION_COMPLETE',
        verification: {
          scriptLoaded: true,
          scriptReady: true,
          inMainContext: true,
          handlersCount: 7,
          eventRouteActive: true,
          verifiedAt: new Date().toISOString(),
          version: '2.2.0',
        },
      });

      expect(machine.isReady()).toBe(true);

      machine.dispatch({
        type: 'START_RECORDING',
        sessionId: 's1',
        recordingId: 'r1',
      });
      machine.dispatch({ type: 'RECORDING_STARTED', startedAt: new Date().toISOString() });

      expect(machine.isRecording()).toBe(true);
      expect(machine.getRecordingId()).toBe('r1');
      expect(machine.getGeneration()).toBe(1);
    });

    it('should handle listener errors gracefully', () => {
      const errorListener = jest.fn(() => {
        throw new Error('Listener error');
      });
      const goodListener = jest.fn();

      machine.subscribe(errorListener);
      machine.subscribe(goodListener);

      // Should not throw and should call second listener
      expect(() => {
        machine.dispatch({ type: 'INITIALIZE', contextId: 'ctx' });
      }).not.toThrow();

      expect(goodListener).toHaveBeenCalled();
    });
  });

  describe('RECOVERABLE_ERRORS', () => {
    it('should include the right error codes', () => {
      expect(RECOVERABLE_ERRORS).toContain('SCRIPT_NOT_READY');
      expect(RECOVERABLE_ERRORS).toContain('EVENT_ROUTE_FAILED');
      expect(RECOVERABLE_ERRORS).toContain('REDIRECT_LOOP');
      expect(RECOVERABLE_ERRORS).toContain('VERIFICATION_TIMEOUT');

      expect(RECOVERABLE_ERRORS).not.toContain('WRONG_CONTEXT');
      expect(RECOVERABLE_ERRORS).not.toContain('PAGE_CLOSED');
    });
  });
});
