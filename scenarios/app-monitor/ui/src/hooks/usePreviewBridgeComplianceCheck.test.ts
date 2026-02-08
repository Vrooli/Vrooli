import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { usePreviewBridgeComplianceCheck } from './usePreviewBridgeComplianceCheck';

describe('usePreviewBridgeComplianceCheck', () => {
  it('runs compliance check and publishes successful result', async () => {
    const runComplianceCheck = vi.fn().mockResolvedValue({ ok: true, failures: [], checkedAt: 1 });
    const onSuccess = vi.fn();
    const onError = vi.fn();

    renderHook(() => usePreviewBridgeComplianceCheck({
      enabled: true,
      runComplianceCheck,
      onSuccess,
      onError,
    }));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledWith({ ok: true, failures: [], checkedAt: 1 });
    });
    expect(onError).not.toHaveBeenCalled();
  });

  it('runs once while enabled when runOnceWhileEnabled is true', async () => {
    const runComplianceCheck = vi.fn().mockResolvedValue({ ok: true, failures: [], checkedAt: 2 });
    const onSuccess = vi.fn();

    const { rerender } = renderHook((enabled: boolean) => usePreviewBridgeComplianceCheck({
      enabled,
      runComplianceCheck,
      onSuccess,
      onError: vi.fn(),
      runOnceWhileEnabled: true,
    }), { initialProps: true });

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledTimes(1);
    });

    rerender(true);
    await waitFor(() => {
      expect(runComplianceCheck).toHaveBeenCalledTimes(1);
    });
  });
});
