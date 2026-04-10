import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { usePreviewUrlOrchestration } from './usePreviewUrlOrchestration';

const { buildPreviewUrlMock } = vi.hoisted(() => ({
  buildPreviewUrlMock: vi.fn(),
}));

vi.mock('@/utils/appPreview', () => ({
  buildPreviewUrl: buildPreviewUrlMock,
}));

describe('usePreviewUrlOrchestration', () => {
  beforeEach(() => {
    buildPreviewUrlMock.mockReset();
  });

  it('applies fallback URL when app is missing and preview is not custom', () => {
    const applyDefaultPreviewUrl = vi.fn();
    const resetPreviewState = vi.fn();
    const setPreviewUrl = vi.fn();
    const initialPreviewUrlRef = { current: null as string | null };

    const { result } = renderHook(() => usePreviewUrlOrchestration({
      hasCustomPreviewUrl: false,
      previewUrl: null,
      applyDefaultPreviewUrl,
      resetPreviewState,
      setPreviewUrl,
      initialPreviewUrlRef,
    }));

    let output: { hasPreviewCandidate: boolean; defaultPreviewUrl: string | null } | null = null;
    act(() => {
      output = result.current({
        appForPreview: null,
        fallbackPreviewUrl: 'http://localhost:4310',
      });
    });

    expect(resetPreviewState).toHaveBeenCalled();
    expect(applyDefaultPreviewUrl).toHaveBeenCalledWith('http://localhost:4310');
    expect(output).toEqual({
      hasPreviewCandidate: false,
      defaultPreviewUrl: 'http://localhost:4310',
    });
  });

  it('applies generated preview URL for app when available', () => {
    buildPreviewUrlMock.mockReturnValue('http://localhost:5001');
    const applyDefaultPreviewUrl = vi.fn();
    const resetPreviewState = vi.fn();
    const setPreviewUrl = vi.fn();
    const initialPreviewUrlRef = { current: null as string | null };

    const { result } = renderHook(() => usePreviewUrlOrchestration({
      hasCustomPreviewUrl: false,
      previewUrl: null,
      applyDefaultPreviewUrl,
      resetPreviewState,
      setPreviewUrl,
      initialPreviewUrlRef,
    }));

    let output: { hasPreviewCandidate: boolean; defaultPreviewUrl: string | null } | null = null;
    act(() => {
      output = result.current({
        appForPreview: { id: 'a' } as never,
      });
    });

    expect(applyDefaultPreviewUrl).toHaveBeenCalledWith('http://localhost:5001');
    expect(output).toEqual({
      hasPreviewCandidate: true,
      defaultPreviewUrl: 'http://localhost:5001',
    });
  });

  it('restores generated URL for custom sessions when URL is currently null', () => {
    buildPreviewUrlMock.mockReturnValue('http://localhost:6600');
    const applyDefaultPreviewUrl = vi.fn();
    const resetPreviewState = vi.fn();
    const setPreviewUrl = vi.fn();
    const initialPreviewUrlRef = { current: null as string | null };

    const { result } = renderHook(() => usePreviewUrlOrchestration({
      hasCustomPreviewUrl: true,
      previewUrl: null,
      applyDefaultPreviewUrl,
      resetPreviewState,
      setPreviewUrl,
      initialPreviewUrlRef,
    }));

    act(() => {
      result.current({
        appForPreview: { id: 'a' } as never,
      });
    });

    expect(initialPreviewUrlRef.current).toBe('http://localhost:6600');
    expect(setPreviewUrl).toHaveBeenCalledWith('http://localhost:6600');
    expect(applyDefaultPreviewUrl).not.toHaveBeenCalled();
  });

  it('does not override in-app proxy navigation when URL changed within same scenario', () => {
    buildPreviewUrlMock.mockReturnValue('/apps/git-control-tower/proxy/');
    const applyDefaultPreviewUrl = vi.fn();
    const resetPreviewState = vi.fn();
    const setPreviewUrl = vi.fn();
    const initialPreviewUrlRef = { current: 'http://localhost/apps/git-control-tower/proxy/' as string | null };

    const { result } = renderHook(() => usePreviewUrlOrchestration({
      hasCustomPreviewUrl: false,
      previewUrl: '/apps/git-control-tower/proxy/?path=README.md',
      applyDefaultPreviewUrl,
      resetPreviewState,
      setPreviewUrl,
      initialPreviewUrlRef,
    }));

    act(() => {
      result.current({
        appForPreview: { id: 'git-control-tower' } as never,
      });
    });

    expect(applyDefaultPreviewUrl).not.toHaveBeenCalled();
    expect(resetPreviewState).not.toHaveBeenCalled();
  });

  it('still reapplies default URL when preview URL points outside the scenario proxy', () => {
    buildPreviewUrlMock.mockReturnValue('/apps/git-control-tower/proxy/');
    const applyDefaultPreviewUrl = vi.fn();
    const resetPreviewState = vi.fn();
    const setPreviewUrl = vi.fn();
    const initialPreviewUrlRef = { current: null as string | null };

    const { result } = renderHook(() => usePreviewUrlOrchestration({
      hasCustomPreviewUrl: false,
      previewUrl: 'https://example.com/dashboard',
      applyDefaultPreviewUrl,
      resetPreviewState,
      setPreviewUrl,
      initialPreviewUrlRef,
    }));

    act(() => {
      result.current({
        appForPreview: { id: 'git-control-tower' } as never,
      });
    });

    expect(applyDefaultPreviewUrl).toHaveBeenCalledWith('/apps/git-control-tower/proxy/');
  });
});
