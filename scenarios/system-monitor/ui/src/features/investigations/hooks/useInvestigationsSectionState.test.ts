import { renderHook, act } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { InvestigationStatus } from '../../../types';
import { useInvestigationsSectionState } from './useInvestigationsSectionState';

describe('useInvestigationsSectionState', () => {
  it('filters investigations and summarizes empty, single, and multiple agents', () => {
    const spawn = vi.fn().mockResolvedValue(undefined);
    const { result, rerender } = renderHook((props: Parameters<typeof useInvestigationsSectionState>[0]) => useInvestigationsSectionState(props), {
      initialProps: { investigations: [
        { id: 'one', status: InvestigationStatus.COMPLETED, findings: 'disk healthy' },
        { id: 'two', status: InvestigationStatus.FAILED, findings: 'network issue' },
      ], agents: [], onSpawnAgent: spawn },
    });
    expect(result.current.activeAgentSummary).toEqual({ text: 'No active agents', tone: 'idle' });
    act(() => { result.current.setReportsSearch('network'); });
    expect(result.current.filteredInvestigations).toHaveLength(1);
    rerender({ investigations: [], agents: [{ id: 'a', status: 'error', startTime: '', autoFix: false }], onSpawnAgent: spawn });
    expect(result.current.activeAgentSummary).toEqual({ text: 'error', tone: 'error' });
    rerender({ investigations: [], agents: [
      { id: 'a', status: 'completed', startTime: '', autoFix: false },
      { id: 'b', status: 'running', startTime: '', autoFix: true },
    ], onSpawnAgent: spawn });
    expect(result.current.activeAgentSummary.text).toBe('2 agents in flight (1 running)');
  });

  it('spawns with trimmed notes and exposes failures', async () => {
    const spawn = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useInvestigationsSectionState({ investigations: [], agents: [], onSpawnAgent: spawn }));
    act(() => { result.current.setAutoFixEnabled(true); result.current.setAgentNote('  inspect disk  '); });
    await act(async () => { await result.current.handleSpawnAgent(); });
    expect(spawn).toHaveBeenCalledWith({ autoFix: true, note: 'inspect disk' });
    expect(result.current.agentNote).toBe('');
    expect(result.current.showNoteField).toBe(false);

    spawn.mockRejectedValueOnce(new Error('blocked'));
    await act(async () => { await result.current.handleSpawnAgent(); });
    expect(result.current.combinedSpawnError).toBe('blocked');
  });
});
