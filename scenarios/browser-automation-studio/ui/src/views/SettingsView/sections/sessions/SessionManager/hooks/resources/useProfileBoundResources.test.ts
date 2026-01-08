import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest';
import { useProfileBoundResources } from './useProfileBoundResources';
import { useStorageState, useServiceWorkers, useHistory } from '@/domains/recording';
import { useTabs } from '@/domains/recording/hooks/useTabs';

// Mock all underlying hooks
vi.mock('@/domains/recording', () => ({
  useStorageState: vi.fn(),
  useServiceWorkers: vi.fn(),
  useHistory: vi.fn(),
}));

vi.mock('@/domains/recording/hooks/useTabs', () => ({
  useTabs: vi.fn(),
}));

describe('useProfileBoundResources', () => {
  const mockProfileId = 'test-profile-123';

  // Default mock return values
  const defaultStorageMock = {
    storageState: null,
    loading: false,
    error: null,
    deleting: false,
    fetchStorageState: vi.fn(),
    clear: vi.fn(),
    clearAllCookies: vi.fn().mockResolvedValue(true),
    deleteCookiesByDomain: vi.fn().mockResolvedValue(true),
    deleteCookie: vi.fn().mockResolvedValue(true),
    clearAllLocalStorage: vi.fn().mockResolvedValue(true),
    deleteLocalStorageByOrigin: vi.fn().mockResolvedValue(true),
    deleteLocalStorageItem: vi.fn().mockResolvedValue(true),
    clearAllStorage: vi.fn().mockResolvedValue(true),
  };

  const defaultServiceWorkersMock = {
    serviceWorkers: null,
    loading: false,
    error: null,
    deleting: false,
    fetchServiceWorkers: vi.fn(),
    clear: vi.fn(),
    unregisterAll: vi.fn().mockResolvedValue(true),
    unregisterWorker: vi.fn().mockResolvedValue(true),
  };

  const defaultHistoryMock = {
    history: null,
    loading: false,
    error: null,
    deleting: false,
    navigating: false,
    fetchHistory: vi.fn(),
    clear: vi.fn(),
    clearAllHistory: vi.fn().mockResolvedValue(true),
    deleteHistoryEntry: vi.fn().mockResolvedValue(true),
    updateSettings: vi.fn().mockResolvedValue(true),
    navigateToUrl: vi.fn().mockResolvedValue(true),
  };

  const defaultTabsMock = {
    tabs: [],
    loading: false,
    error: null,
    deleting: false,
    fetchTabs: vi.fn(),
    clear: vi.fn(),
    clearAllTabs: vi.fn().mockResolvedValue(true),
    deleteTab: vi.fn().mockResolvedValue(true),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    (useStorageState as Mock).mockReturnValue(defaultStorageMock);
    (useServiceWorkers as Mock).mockReturnValue(defaultServiceWorkersMock);
    (useHistory as Mock).mockReturnValue(defaultHistoryMock);
    (useTabs as Mock).mockReturnValue(defaultTabsMock);
  });

  describe('storage operations', () => {
    it('binds profileId to fetch', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.storage.fetch();
      });

      expect(defaultStorageMock.fetchStorageState).toHaveBeenCalledWith(mockProfileId);
    });

    it('binds profileId to clearAllCookies', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.storage.clearAllCookies();
      });

      expect(defaultStorageMock.clearAllCookies).toHaveBeenCalledWith(mockProfileId);
    });

    it('binds profileId to deleteCookiesByDomain', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.storage.deleteCookiesByDomain('example.com');
      });

      expect(defaultStorageMock.deleteCookiesByDomain).toHaveBeenCalledWith(
        mockProfileId,
        'example.com'
      );
    });

    it('binds profileId to deleteCookie', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.storage.deleteCookie('example.com', 'sessionId');
      });

      expect(defaultStorageMock.deleteCookie).toHaveBeenCalledWith(
        mockProfileId,
        'example.com',
        'sessionId'
      );
    });

    it('exposes storage state correctly', () => {
      const mockStorageState = {
        cookies: [{ name: 'test', value: 'value', domain: 'example.com' }],
        origins: [],
        stats: { cookieCount: 1, localStorageCount: 0, originCount: 0 },
      };

      (useStorageState as Mock).mockReturnValue({
        ...defaultStorageMock,
        storageState: mockStorageState,
        loading: true,
        error: 'test error',
      });

      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      expect(result.current.storage.state).toBe(mockStorageState);
      expect(result.current.storage.loading).toBe(true);
      expect(result.current.storage.error).toBe('test error');
    });
  });

  describe('service workers operations', () => {
    it('binds profileId to fetch', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.serviceWorkers.fetch();
      });

      expect(defaultServiceWorkersMock.fetchServiceWorkers).toHaveBeenCalledWith(mockProfileId);
    });

    it('binds profileId to unregisterAll', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.serviceWorkers.unregisterAll();
      });

      expect(defaultServiceWorkersMock.unregisterAll).toHaveBeenCalledWith(mockProfileId);
    });

    it('binds profileId to unregister', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.serviceWorkers.unregister('https://example.com/sw-scope');
      });

      expect(defaultServiceWorkersMock.unregisterWorker).toHaveBeenCalledWith(
        mockProfileId,
        'https://example.com/sw-scope'
      );
    });

    it('provides workers array shorthand', () => {
      const mockWorkers = [
        { registrationId: '1', scopeURL: 'https://example.com/', scriptURL: 'sw.js', status: 'running' as const },
      ];

      (useServiceWorkers as Mock).mockReturnValue({
        ...defaultServiceWorkersMock,
        serviceWorkers: { session_id: 'sess-123', workers: mockWorkers, control: { mode: 'allow' } },
      });

      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      expect(result.current.serviceWorkers.workers).toEqual(mockWorkers);
    });

    it('returns empty workers array when no data', () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      expect(result.current.serviceWorkers.workers).toEqual([]);
    });
  });

  describe('history operations', () => {
    it('binds profileId to fetch', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.history.fetch();
      });

      expect(defaultHistoryMock.fetchHistory).toHaveBeenCalledWith(mockProfileId);
    });

    it('binds profileId to clearAll', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.history.clearAll();
      });

      expect(defaultHistoryMock.clearAllHistory).toHaveBeenCalledWith(mockProfileId);
    });

    it('binds profileId to deleteEntry', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.history.deleteEntry('entry-456');
      });

      expect(defaultHistoryMock.deleteHistoryEntry).toHaveBeenCalledWith(
        mockProfileId,
        'entry-456'
      );
    });

    it('binds profileId to updateSettings', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      const newSettings = { maxEntries: 200, retentionDays: 7 };
      await act(async () => {
        await result.current.history.updateSettings(newSettings);
      });

      expect(defaultHistoryMock.updateSettings).toHaveBeenCalledWith(mockProfileId, newSettings);
    });

    it('binds profileId to navigateTo', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.history.navigateTo('https://example.com/page');
      });

      expect(defaultHistoryMock.navigateToUrl).toHaveBeenCalledWith(
        mockProfileId,
        'https://example.com/page'
      );
    });

    it('exposes navigating state', () => {
      (useHistory as Mock).mockReturnValue({
        ...defaultHistoryMock,
        navigating: true,
      });

      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      expect(result.current.history.navigating).toBe(true);
    });
  });

  describe('tabs operations', () => {
    it('binds profileId to fetch', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.tabs.fetch();
      });

      expect(defaultTabsMock.fetchTabs).toHaveBeenCalledWith(mockProfileId);
    });

    it('binds profileId to clearAll', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.tabs.clearAll();
      });

      expect(defaultTabsMock.clearAllTabs).toHaveBeenCalledWith(mockProfileId);
    });

    it('binds profileId to delete', async () => {
      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      await act(async () => {
        await result.current.tabs.delete(2);
      });

      expect(defaultTabsMock.deleteTab).toHaveBeenCalledWith(mockProfileId, 2);
    });

    it('exposes tabs data correctly', () => {
      const mockTabs = [
        { url: 'https://example.com', title: 'Example', isActive: true, order: 0 },
        { url: 'https://test.com', title: 'Test', isActive: false, order: 1 },
      ];

      (useTabs as Mock).mockReturnValue({
        ...defaultTabsMock,
        tabs: mockTabs,
      });

      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      expect(result.current.tabs.data).toEqual(mockTabs);
    });
  });

  describe('hasActiveSession derivation', () => {
    it('returns false when service workers has no session_id', () => {
      (useServiceWorkers as Mock).mockReturnValue({
        ...defaultServiceWorkersMock,
        serviceWorkers: null,
      });

      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      expect(result.current.hasActiveSession).toBe(false);
    });

    it('returns false when service workers session_id is empty', () => {
      (useServiceWorkers as Mock).mockReturnValue({
        ...defaultServiceWorkersMock,
        serviceWorkers: { session_id: '', workers: [], control: { mode: 'allow' } },
      });

      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      expect(result.current.hasActiveSession).toBe(false);
    });

    it('returns true when service workers has session_id', () => {
      (useServiceWorkers as Mock).mockReturnValue({
        ...defaultServiceWorkersMock,
        serviceWorkers: { session_id: 'active-session-123', workers: [], control: { mode: 'allow' } },
      });

      const { result } = renderHook(() =>
        useProfileBoundResources({ profileId: mockProfileId })
      );

      expect(result.current.hasActiveSession).toBe(true);
    });
  });

  describe('profileId changes', () => {
    it('creates new bound callbacks when profileId changes', () => {
      const { result, rerender } = renderHook(
        ({ profileId }) => useProfileBoundResources({ profileId }),
        { initialProps: { profileId: 'profile-1' } }
      );

      const firstFetch = result.current.storage.fetch;

      rerender({ profileId: 'profile-2' });

      const secondFetch = result.current.storage.fetch;

      // The fetch function should be different because profileId changed
      expect(firstFetch).not.toBe(secondFetch);
    });
  });
});
