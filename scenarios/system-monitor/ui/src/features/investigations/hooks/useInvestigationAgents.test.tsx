import { renderHook, act, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { InvestigationStatus } from '../../../types';
import { useInvestigationAgents } from './useInvestigationAgents';

const mocks = vi.hoisted(() => ({
  protoFetch: vi.fn(),
  usePolling: vi.fn(),
}));

vi.mock('../../../shared/api/apiFetch', () => ({
  protoFetch: mocks.protoFetch,
  extractErrorMessage: (error: unknown, fallback: string) => error instanceof Error ? error.message : fallback,
  isApiError: () => false,
}));
vi.mock('../../../shared/hooks/usePolling', () => ({ usePolling: mocks.usePolling }));

const ts = timestampFromDate(new Date('2026-02-02T00:00:00Z'));

describe('useInvestigationAgents', () => {
  beforeEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
    mocks.protoFetch.mockResolvedValue({ id: '' });
  });

  it('loads no active agent, spawns one, and removes it on stop', async () => {
    const { result } = renderHook(() => useInvestigationAgents());
    await waitFor(() => { expect(mocks.protoFetch).toHaveBeenCalledWith('/investigations/agent/current', expect.anything()); }, { timeout: 3000 });
    await act(async () => { await new Promise(resolve => window.setTimeout(resolve, 25)); });
    expect(result.current.agents).toHaveLength(0);

    mocks.protoFetch.mockResolvedValueOnce({ investigationId: 'agent-1', autoFix: true, note: 'inspect' });
    await act(async () => {
      await result.current.spawnAgent({ autoFix: true, note: 'inspect' });
    });
    await waitFor(() => { expect(result.current.agents[0]?.id).toBe('agent-1'); });

    mocks.protoFetch.mockResolvedValueOnce({ status: 'stopped', id: 'agent-1' });
    await act(async () => { await result.current.stopAgent('agent-1'); });
    expect(result.current.agents).toHaveLength(0);
  });

  it('maps status updates and drops terminal agents during polling', async () => {
    const { result } = renderHook(() => useInvestigationAgents());
    await waitFor(() => { expect(mocks.protoFetch).toHaveBeenCalledWith('/investigations/agent/current', expect.anything()); }, { timeout: 3000 });
    await act(async () => { await new Promise(resolve => window.setTimeout(resolve, 25)); });
    mocks.protoFetch.mockReset();
    mocks.protoFetch
      .mockResolvedValueOnce({ id: 'agent-1', status: InvestigationStatus.IN_PROGRESS, startTime: ts, progress: 20 })
      .mockResolvedValueOnce({ id: 'agent-1', status: InvestigationStatus.IN_PROGRESS, startTime: ts, progress: 20 })
      .mockResolvedValueOnce({ id: 'agent-1', status: InvestigationStatus.COMPLETED, startTime: ts, endTime: ts });
    await act(async () => { await result.current.refreshAgents(); });
    await waitFor(() => { expect(result.current.agents[0]?.id).toBe('agent-1'); });
    const poll = mocks.usePolling.mock.calls[0]?.[0] as (() => Promise<void>) | undefined;
    expect(poll).toBeDefined();
    await act(async () => { await poll?.(); });
    await waitFor(() => { expect(result.current.agents).toHaveLength(0); });
    expect(mocks.protoFetch).toHaveBeenCalledWith('/investigations/agent/current', expect.anything());
  });

  it('surfaces spawn and stop failures while clearing in-flight state', async () => {
    const { result } = renderHook(() => useInvestigationAgents());
    await waitFor(() => { expect(mocks.protoFetch).toHaveBeenCalledWith('/investigations/agent/current', expect.anything()); }, { timeout: 3000 });
    mocks.protoFetch.mockReset();
    mocks.protoFetch.mockRejectedValueOnce(new Error('spawn failed'));
    await act(async () => {
      await expect(result.current.spawnAgent({ autoFix: false })).rejects.toThrow('spawn failed');
    });
    await waitFor(() => { expect(result.current.spawnAgentError).toBe('spawn failed'); });

    mocks.protoFetch
      .mockResolvedValueOnce({ investigationId: 'agent-2', autoFix: false })
      .mockResolvedValueOnce({ id: 'agent-2', status: InvestigationStatus.IN_PROGRESS, startTime: ts });
    await act(async () => { await result.current.spawnAgent({ autoFix: false }); });
    await waitFor(() => { expect(result.current.agents[0]?.id).toBe('agent-2'); });
    await act(async () => { await new Promise(resolve => window.setTimeout(resolve, 25)); });
    mocks.protoFetch.mockReset();
    mocks.protoFetch.mockRejectedValueOnce(new Error('stop failed'));
    let stopError: unknown;
    await act(async () => {
      try {
        await result.current.stopAgent('agent-2');
      } catch (error) {
        stopError = error;
      }
    });
    expect(stopError).toEqual(new Error('stop failed'));
    await waitFor(() => { expect(result.current.agentErrors['agent-2']).toBe('stop failed'); });
  });

  it('treats parse failures as no agent and ignores polling failures', async () => {
    mocks.protoFetch.mockRejectedValueOnce(new SyntaxError('empty response'));
    const { result } = renderHook(() => useInvestigationAgents());
    await act(async () => { await result.current.refreshAgents(); });
    expect(result.current.agents).toHaveLength(0);
    mocks.protoFetch.mockResolvedValueOnce({ investigationId: 'agent-3', autoFix: false });
    await act(async () => { await result.current.spawnAgent({ autoFix: false }); });
    await waitFor(() => { expect(result.current.agents[0]?.id).toBe('agent-3'); });
    mocks.protoFetch.mockRejectedValueOnce(new Error('poll unavailable'));
    const poll = mocks.usePolling.mock.calls[0]?.[0] as (() => Promise<void>) | undefined;
    await act(async () => { await poll?.(); });
    expect(result.current.agents[0]?.id).toBe('agent-3');
  });

  it('coalesces concurrent spawns and handles duplicate agent updates', async () => {
    let resolveSpawn: ((value: unknown) => void) | undefined;
    mocks.protoFetch.mockImplementationOnce(() => new Promise(resolve => { resolveSpawn = resolve; }));
    const { result } = renderHook(() => useInvestigationAgents());
    let first: Promise<unknown> | undefined;
    await act(async () => { first = result.current.spawnAgent({ autoFix: true, note: 'one' }); await Promise.resolve(); });
    let second: unknown;
    await act(async () => { second = await result.current.spawnAgent({ autoFix: false, note: 'two' }); });
    expect(second).toBeUndefined();
    await act(async () => { resolveSpawn?.({ investigationId: 'agent-4', autoFix: true, note: 'one' }); await first; });
    mocks.protoFetch.mockResolvedValueOnce({ investigationId: 'agent-4', autoFix: false, note: 'updated' });
    await act(async () => { await result.current.spawnAgent({ autoFix: false, note: 'updated' }); });
    expect(result.current.agents).toHaveLength(1);
    expect(result.current.agents[0]?.note).toBe('updated');
  });
});
