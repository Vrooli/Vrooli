import type { SessionPhase, SessionState } from '../../../src/types';
import {
  findByExecutionId,
  findByLabels,
  findIdleSessions,
  isSessionActive,
  isSafeForLabelReuse,
  makeReuseDecision,
  shouldCleanupSession,
} from '../../../src/session/session-decisions';

function makeSession(overrides: {
  executionId?: string;
  labels?: Record<string, string>;
  phase?: SessionPhase;
  leaseReleasedAt?: Date;
  lastUsedAt?: Date;
}): SessionState {
  return {
    id: `session-${overrides.executionId ?? 'x'}`,
    spec: {
      execution_id: overrides.executionId ?? 'exec-1',
      labels: overrides.labels ?? { mode: 'execution' },
    },
    phase: overrides.phase ?? 'ready',
    leaseReleasedAt: overrides.leaseReleasedAt,
    lastUsedAt: overrides.lastUsedAt ?? new Date(),
  } as unknown as SessionState;
}

describe('session decisions', () => {
  describe('isSafeForLabelReuse', () => {
    it('allows pooling of ready sessions whose lease was released', () => {
      expect(
        isSafeForLabelReuse(makeSession({ phase: 'ready', leaseReleasedAt: new Date() }))
      ).toBe(true);
    });

    it('rejects a ready session whose owner never released its lease', () => {
      expect(isSafeForLabelReuse(makeSession({ phase: 'ready' }))).toBe(false);
    });

    it('rejects busy phases even with a released lease', () => {
      const busyPhases: SessionPhase[] = [
        'initializing',
        'executing',
        'recording',
        'resetting',
        'closing',
      ];
      for (const phase of busyPhases) {
        expect(isSafeForLabelReuse(makeSession({ phase, leaseReleasedAt: new Date() }))).toBe(
          false
        );
      }
    });
  });

  describe('findByLabels', () => {
    it('skips sessions that are executing for another execution', () => {
      // Regression: two adhoc executions launched concurrently used to match
      // each other's session by labels ({mode: execution}); the second caller
      // hijacked the first's session mid-instruction, aborting its navigation
      // (net::ERR_ABORTED) and racing into SESSION_BUSY.
      const busy = makeSession({ executionId: 'exec-1', phase: 'executing' });
      const found = findByLabels([busy], { mode: 'execution' });
      expect(found).toBeNull();
    });

    it('returns a released idle session with matching labels', () => {
      const busy = makeSession({ executionId: 'exec-1', phase: 'executing' });
      const idle = makeSession({
        executionId: 'exec-2',
        phase: 'ready',
        leaseReleasedAt: new Date(),
      });
      const found = findByLabels([busy, idle], { mode: 'execution' });
      expect(found).toBe(idle);
    });

    it('returns null when labels do not match', () => {
      const idle = makeSession({
        executionId: 'exec-1',
        labels: { mode: 'record' },
        leaseReleasedAt: new Date(),
      });
      expect(findByLabels([idle], { mode: 'execution' })).toBeNull();
    });
  });

  describe('findByExecutionId', () => {
    it('still returns busy sessions for the same execution (idempotent retry)', () => {
      const busy = makeSession({ executionId: 'exec-1', phase: 'executing' });
      expect(findByExecutionId([busy], 'exec-1')).toBe(busy);
    });
  });

  describe('makeReuseDecision', () => {
    it('recovers stuck executing phase only for execution_id retries', () => {
      const stuck = makeSession({ executionId: 'exec-1', phase: 'executing' });
      const spec = stuck.spec;
      expect(makeReuseDecision(stuck, spec, 'execution_id_match').shouldRecoverPhase).toBe(true);
      expect(makeReuseDecision(stuck, spec, 'label_match').shouldRecoverPhase).toBe(false);
    });
  });

  describe('idle cleanup', () => {
    it('keeps an executing session active beyond the idle timeout', () => {
      const old = new Date(0);
      const executing = makeSession({ phase: 'executing', lastUsedAt: old });

      expect(isSessionActive(executing, 100, 1000)).toBe(true);
      expect(shouldCleanupSession(executing, 100, 1000)).toBe(false);
      expect(findIdleSessions(new Map([[executing.id, executing]]), 100, 1000)).toEqual([]);
    });

    it('still cleans an old ready session', () => {
      const ready = makeSession({ phase: 'ready', lastUsedAt: new Date(0) });

      expect(isSessionActive(ready, 100, 1000)).toBe(false);
      expect(shouldCleanupSession(ready, 100, 1000)).toBe(true);
      expect(findIdleSessions(new Map([[ready.id, ready]]), 100, 1000)).toEqual([ready.id]);
    });
  });
});
