const mockLogger = {
  debug: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
};

jest.mock('../../../src/utils', () => ({
  logger: mockLogger,
  scopedLog: (_context: unknown, message: string): string => message,
  LogContext: {
    SESSION: 'session',
  },
}));

import type { SessionPhase } from '../../../src/types';
import {
  canAcceptInstructions,
  canClose,
  canTransition,
  getValidTransitions,
  isBusy,
  isOperational,
  isTerminal,
  transition,
  transitionStrict,
} from '../../../src/session/state-machine';

describe('session state machine', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('defines the valid transition contract for every phase', () => {
    const expectedTransitions: Record<SessionPhase, readonly SessionPhase[]> = {
      initializing: ['ready', 'closing'],
      ready: ['executing', 'recording', 'resetting', 'closing'],
      executing: ['ready', 'recording', 'resetting', 'closing'],
      recording: ['ready', 'executing', 'resetting', 'closing'],
      resetting: ['ready', 'closing'],
      closing: [],
    };

    for (const [phase, targets] of Object.entries(expectedTransitions) as Array<[SessionPhase, readonly SessionPhase[]]>) {
      expect(getValidTransitions(phase)).toEqual(targets);
      for (const target of targets) {
        expect(canTransition(phase, target)).toBe(true);
      }
    }
  });

  it('allows closing from every non-terminal phase only', () => {
    const phases: SessionPhase[] = ['initializing', 'ready', 'executing', 'recording', 'resetting', 'closing'];

    expect(phases.filter(canClose)).toEqual(['initializing', 'ready', 'executing', 'recording', 'resetting']);
    expect(canTransition('closing', 'ready')).toBe(false);
  });

  it('returns the target phase and logs debug for valid transitions', () => {
    expect(transition('ready', 'executing', 'session-1')).toBe('executing');

    expect(mockLogger.debug).toHaveBeenCalledWith('phase transition', {
      sessionId: 'session-1',
      from: 'ready',
      to: 'executing',
    });
    expect(mockLogger.warn).not.toHaveBeenCalled();
  });

  it('keeps the current phase and logs warning for fail-safe invalid transitions', () => {
    expect(transition('closing', 'ready', 'session-1')).toBe('closing');

    expect(mockLogger.warn).toHaveBeenCalledWith('invalid phase transition attempted', {
      sessionId: 'session-1',
      from: 'closing',
      to: 'ready',
      validTargets: [],
      hint: "Transition from 'closing' to 'ready' is not allowed. Check state machine rules.",
    });
  });

  it('throws and logs error for strict invalid transitions', () => {
    expect(() => transitionStrict('initializing', 'executing', 'session-1')).toThrow(
      "Invalid session phase transition: initializing \u2192 executing (session: session-1). Valid transitions from 'initializing': [ready, closing]"
    );

    expect(mockLogger.error).toHaveBeenCalledWith('invalid phase transition', {
      sessionId: 'session-1',
      from: 'initializing',
      to: 'executing',
      validTargets: ['ready', 'closing'],
      error: "Invalid session phase transition: initializing \u2192 executing (session: session-1). Valid transitions from 'initializing': [ready, closing]",
    });
  });

  it('classifies operational, busy, terminal, and instruction-accepting phases', () => {
    const phases: SessionPhase[] = ['initializing', 'ready', 'executing', 'recording', 'resetting', 'closing'];

    expect(phases.filter(isOperational)).toEqual(['ready', 'executing', 'recording']);
    expect(phases.filter(isBusy)).toEqual(['initializing', 'executing', 'resetting']);
    expect(phases.filter(isTerminal)).toEqual(['closing']);
    expect(phases.filter(canAcceptInstructions)).toEqual(['ready', 'recording']);
  });
});
