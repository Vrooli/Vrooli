import { renderHook, act, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { ScriptExecutionStatus } from '../../../types';
import { useScriptExecution } from './useScriptExecution';
import type { InvestigationScript } from '../../../types';

const mocks = vi.hoisted(() => ({ protoFetch: vi.fn() }));
vi.mock('../../../shared/api/apiFetch', () => ({ protoFetch: mocks.protoFetch }));

const script = {
  id: 'disk-check',
  name: 'Disk check',
  description: 'Inspect disk pressure',
  category: 'storage',
  author: 'system',
  enabled: true,
} as unknown as InvestigationScript;

describe('useScriptExecution', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('opens and closes the editor and results modals with the requested context', () => {
    const { result } = renderHook(() => useScriptExecution());
    expect(result.current.modalState.scriptEditor.isOpen).toBe(false);
    act(() => { result.current.openScriptEditor(script, 'echo ok', 'edit'); });
    expect(result.current.modalState.scriptEditor).toMatchObject({ isOpen: true, mode: 'edit', scriptId: 'disk-check', scriptContent: 'echo ok' });
    act(() => { result.current.closeScriptEditor(); });
    expect(result.current.modalState.scriptEditor.isOpen).toBe(false);
    act(() => { result.current.closeScriptResults(); });
    expect(result.current.modalState.scriptResults.isOpen).toBe(false);
  });

  it('shows a running result and replaces it with the server execution', async () => {
    const completed = {
      scriptId: 'disk-check',
      executionId: 'exec-1',
      status: ScriptExecutionStatus.COMPLETED,
      startedAt: timestampFromDate(new Date('2026-01-01T00:00:00Z')),
      completedAt: timestampFromDate(new Date('2026-01-01T00:00:01Z')),
      exitCode: 0,
      output: 'healthy',
    };
    mocks.protoFetch.mockResolvedValueOnce({ execution: completed });
    const { result } = renderHook(() => useScriptExecution());
    act(() => { result.current.openScriptEditor(script, 'echo ok', 'view'); });
    await act(async () => { await result.current.executeScript('disk-check', 'echo ok'); });
    await waitFor(() => { expect(result.current.modalState.scriptResults.execution?.status).toBe(ScriptExecutionStatus.COMPLETED); });
    expect(result.current.modalState.scriptResults).toMatchObject({ isOpen: true, scriptId: 'disk-check' });
    expect(result.current.modalState.scriptResults.execution?.executionId).toBe('exec-1');
    expect(result.current.modalState.scriptEditor.isOpen).toBe(false);
    expect(mocks.protoFetch).toHaveBeenCalledWith('/investigations/scripts/disk-check/execute', expect.anything(), expect.objectContaining({ method: 'POST', body: JSON.stringify({ content: 'echo ok' }) }));
  });

  it('records failed execution details and closes the editor even when the API fails', async () => {
    mocks.protoFetch.mockRejectedValueOnce(new Error('script unavailable'));
    const { result } = renderHook(() => useScriptExecution());
    act(() => { result.current.openScriptEditor(script, '', 'edit'); });
    await act(async () => { await result.current.executeScript('disk-check', ''); });
    await waitFor(() => { expect(result.current.modalState.scriptResults.execution?.status).toBe(ScriptExecutionStatus.FAILED); });
    expect(result.current.modalState.scriptResults.execution?.error).toBe('script unavailable');
    expect(result.current.modalState.scriptEditor.isOpen).toBe(false);
  });

  it('uses a generic failure message for non-Error execution failures', async () => {
    mocks.protoFetch.mockRejectedValueOnce('offline');
    const { result } = renderHook(() => useScriptExecution());
    await act(async () => { await result.current.executeScript('disk-check', 'echo ok'); });
    await waitFor(() => { expect(result.current.modalState.scriptResults.execution?.error).toBe('Script execution failed'); });
  });

  it('keeps the pending result when the server returns no execution payload', async () => {
    mocks.protoFetch.mockResolvedValueOnce({ execution: undefined });
    const { result } = renderHook(() => useScriptExecution());
    await act(async () => { await result.current.executeScript('disk-check', 'echo ok'); });
    expect(result.current.modalState.scriptResults.execution?.status).toBe(ScriptExecutionStatus.RUNNING);
  });

  it('persists a script before closing the editor', async () => {
    const { result } = renderHook(() => useScriptExecution());
    act(() => { result.current.openScriptEditor(script, 'echo ok', 'edit'); });
    await act(async () => { await result.current.saveScript(script, 'echo changed'); });
    expect(result.current.modalState.scriptEditor.isOpen).toBe(false);
    expect(mocks.protoFetch).toHaveBeenCalledWith(
      '/investigations/scripts/disk-check',
      expect.any(Function),
      expect.objectContaining({ method: 'PUT', body: JSON.stringify({ id: 'disk-check', content: 'echo changed' }) }),
    );
  });
});
