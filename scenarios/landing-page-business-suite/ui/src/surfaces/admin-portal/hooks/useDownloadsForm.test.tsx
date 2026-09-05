import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import type {
  DownloadApp,
  listDownloadAppsAdmin,
  createDownloadAppAdmin,
  saveDownloadAppAdmin,
  deleteDownloadAppAdmin,
} from '../../../shared/api';
import { useDownloadsForm } from './useDownloadsForm';
import { assertDefined } from '../../../shared/test-utils/api-mocks';

// Mock API functions
type ListDownloadAppsAdminFn = typeof listDownloadAppsAdmin;
type CreateDownloadAppAdminFn = typeof createDownloadAppAdmin;
type SaveDownloadAppAdminFn = typeof saveDownloadAppAdmin;
type DeleteDownloadAppAdminFn = typeof deleteDownloadAppAdmin;

const listDownloadAppsAdminMock = vi.fn<ListDownloadAppsAdminFn>();
const createDownloadAppAdminMock = vi.fn<CreateDownloadAppAdminFn>();
const saveDownloadAppAdminMock = vi.fn<SaveDownloadAppAdminFn>();
const deleteDownloadAppAdminMock = vi.fn<DeleteDownloadAppAdminFn>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    listDownloadAppsAdmin: (...args: Parameters<ListDownloadAppsAdminFn>) => listDownloadAppsAdminMock(...args),
    createDownloadAppAdmin: (...args: Parameters<CreateDownloadAppAdminFn>) => createDownloadAppAdminMock(...args),
    saveDownloadAppAdmin: (...args: Parameters<SaveDownloadAppAdminFn>) => saveDownloadAppAdminMock(...args),
    deleteDownloadAppAdmin: (...args: Parameters<DeleteDownloadAppAdminFn>) => deleteDownloadAppAdminMock(...args),
  };
});

// Mock window.confirm
const confirmSpy = vi.spyOn(window, 'confirm');

const mockApp: DownloadApp = {
  bundle_key: 'test-bundle',
  app_key: 'test-app',
  name: 'Test App',
  tagline: 'A great app',
  description: 'Full description',
  display_order: 0,
  platforms: [
    {
      bundle_key: 'test-bundle',
      app_key: 'test-app',
      platform: 'windows',
      artifact_url: 'https://example.com/app.exe',
      release_version: '1.0.0',
      requires_entitlement: false,
    },
  ],
  storefronts: [
    {
      store: 'app_store',
      label: 'App Store',
      url: 'https://apps.apple.com/123',
    },
  ],
};

const mockApp2: DownloadApp = {
  bundle_key: 'test-bundle',
  app_key: 'test-app-2',
  name: 'Test App 2',
  display_order: 1,
  platforms: [],
};

type DownloadAppInputPayload = Parameters<CreateDownloadAppAdminFn>[0];

const buildDownloadApp = (appKey: string, payload: DownloadAppInputPayload): DownloadApp => ({
  bundle_key: mockApp.bundle_key,
  app_key: appKey,
  name: payload.name,
  tagline: payload.tagline,
  description: payload.description,
  icon_url: payload.icon_url,
  screenshot_url: payload.screenshot_url,
  install_overview: payload.install_overview,
  install_steps: payload.install_steps,
  storefronts: payload.storefronts ?? [],
  metadata: payload.metadata,
  display_order: payload.display_order ?? 0,
  platforms: payload.platforms.map((platform) => ({
    bundle_key: mockApp.bundle_key,
    app_key: appKey,
    platform: platform.platform,
    artifact_url: platform.artifact_url,
    artifact_source: platform.artifact_source,
    artifact_id: platform.artifact_id,
    release_version: platform.release_version,
    release_notes: platform.release_notes,
    checksum: platform.checksum,
    requires_entitlement: platform.requires_entitlement ?? false,
    metadata: platform.metadata,
  })),
});

describe('useDownloadsForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listDownloadAppsAdminMock.mockResolvedValue({ apps: [mockApp] });
    createDownloadAppAdminMock.mockImplementation((payload) => {
      const appKey = payload.app_key ?? 'new-app';
      return Promise.resolve(buildDownloadApp(appKey, payload));
    });
    saveDownloadAppAdminMock.mockImplementation((key, payload) =>
      Promise.resolve(buildDownloadApp(key, payload))
    );
    deleteDownloadAppAdminMock.mockResolvedValue({});
    confirmSpy.mockReturnValue(true);
  });

  describe('initial state', () => {
    it('loads apps on mount', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      expect(result.current.loading).toBe(true);

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(listDownloadAppsAdminMock).toHaveBeenCalled();
      expect(result.current.forms).toHaveLength(1);
      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.appKey).toBe('test-app');
    });

    it('handles load error', async () => {
      listDownloadAppsAdminMock.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Network error');
      expect(result.current.forms).toHaveLength(0);
    });

    it('uses a stable fallback when the app catalog rejects with a non-Error value', async () => {
      listDownloadAppsAdminMock.mockRejectedValue('offline');
      const { result } = renderHook(() => useDownloadsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.error).toBe('Failed to load download apps');
    });

    it('sorts apps by display_order', async () => {
      listDownloadAppsAdminMock.mockResolvedValue({
        apps: [mockApp2, mockApp], // Out of order
      });

      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.appKey).toBe('test-app'); // display_order: 0
      const form1 = result.current.forms[1];
      assertDefined(form1, 'forms[1]');
      expect(form1.values.appKey).toBe('test-app-2'); // display_order: 1
    });
  });

  describe('handleFieldChange', () => {
    it('updates field value', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('test-app', 'name', 'Updated Name');
      });

      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.name).toBe('Updated Name');
    });

    it('converts displayOrder to number', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('test-app', 'displayOrder', '5');
      });

      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.displayOrder).toBe(5);
    });
  });

  describe('handlePlatformChange', () => {
    it('updates platform field value', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handlePlatformChange('test-app', 'windows', 'releaseVersion', '2.0.0');
      });

      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.platforms.windows.releaseVersion).toBe('2.0.0');
    });

    it('updates platform enabled state', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handlePlatformChange('test-app', 'mac', 'enabled', true);
      });

      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.platforms.mac.enabled).toBe(true);
    });
  });

  describe('handleAddApp', () => {
    it('adds a new app form', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleAddApp();
      });

      expect(result.current.forms).toHaveLength(2);
      const newForm = result.current.forms[1];
      assertDefined(newForm, 'forms[1]');
      expect(newForm.isNew).toBe(true);
      expect(newForm.values.name).toBe('New Bundle App');
      expect(newForm.key).toMatch(/^app-\d+$/);
    });
  });

  describe('handleReset', () => {
    it('resets form to original values', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Make changes
      act(() => {
        result.current.handleFieldChange('test-app', 'name', 'Changed Name');
        result.current.handleFieldChange('test-app', 'tagline', 'Changed Tagline');
      });

      let form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.name).toBe('Changed Name');

      // Reset
      act(() => {
        result.current.handleReset('test-app');
      });

      form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.name).toBe('Test App');
      expect(form0.values.tagline).toBe('A great app');
    });

    it('clears error on reset', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Simulate error
      saveDownloadAppAdminMock.mockRejectedValueOnce(new Error('Save failed'));

      await act(async () => {
        await result.current.handleSave('test-app');
      });

      let form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.error).toBe('Save failed');

      act(() => {
        result.current.handleReset('test-app');
      });

      form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.error).toBeUndefined();
    });
  });

  describe('handleDelete', () => {
    it('removes new unsaved app without API call', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleAddApp();
      });

      const form1 = result.current.forms[1];
      assertDefined(form1, 'forms[1]');
      const newKey = form1.key;

      await act(async () => {
        await result.current.handleDelete(newKey);
      });

      expect(deleteDownloadAppAdminMock).not.toHaveBeenCalled();
      expect(result.current.forms).toHaveLength(1);
    });

    it('calls API to delete existing app', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleDelete('test-app');
      });

      expect(deleteDownloadAppAdminMock).toHaveBeenCalledWith('test-app');
      expect(result.current.forms).toHaveLength(0);
    });

    it('respects confirmation dialog', async () => {
      confirmSpy.mockReturnValue(false);

      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleDelete('test-app');
      });

      expect(deleteDownloadAppAdminMock).not.toHaveBeenCalled();
      expect(result.current.forms).toHaveLength(1);
    });

    it('handles delete error', async () => {
      deleteDownloadAppAdminMock.mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleDelete('test-app');
      });

      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.error).toBe('Delete failed');
      expect(result.current.forms).toHaveLength(1);
    });

    it('keeps an existing app editable after a non-Error delete failure', async () => {
      deleteDownloadAppAdminMock.mockRejectedValue('network lost');
      const { result } = renderHook(() => useDownloadsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      await act(async () => { await result.current.handleDelete('test-app'); });
      expect(result.current.forms[0]).toMatchObject({ key: 'test-app', saving: false, error: 'Failed to delete app' });
    });
  });

  describe('handleSave', () => {
    it('ignores save requests for forms that no longer exist', async () => {
      const { result } = renderHook(() => useDownloadsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      await act(async () => { await result.current.handleSave('missing-app'); });
      expect(saveDownloadAppAdminMock).not.toHaveBeenCalled();
      expect(createDownloadAppAdminMock).not.toHaveBeenCalled();
    });
    it('saves existing app', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('test-app', 'name', 'Updated Name');
      });

      await act(async () => {
        await result.current.handleSave('test-app');
      });

      expect(saveDownloadAppAdminMock).toHaveBeenCalled();
      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.lastSavedAt).toBeDefined();
    });

    it('creates new app', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleAddApp();
      });

      let form1 = result.current.forms[1];
      assertDefined(form1, 'forms[1]');
      const newKey = form1.key;

      act(() => {
        result.current.handleFieldChange(newKey, 'appKey', 'new-app');
      });

      await act(async () => {
        await result.current.handleSave(newKey);
      });

      expect(createDownloadAppAdminMock).toHaveBeenCalled();
      form1 = result.current.forms[1];
      assertDefined(form1, 'forms[1]');
      expect(form1.isNew).toBe(false);
    });

    it('validates required appKey', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleAddApp();
      });

      let form1 = result.current.forms[1];
      assertDefined(form1, 'forms[1]');
      const newKey = form1.key;

      // Don't set appKey, leave it as temp key
      act(() => {
        result.current.handleFieldChange(newKey, 'appKey', '  ');
      });

      await act(async () => {
        await result.current.handleSave(newKey);
      });

      expect(createDownloadAppAdminMock).not.toHaveBeenCalled();
      form1 = result.current.forms[1];
      assertDefined(form1, 'forms[1]');
      expect(form1.error).toBe('App key is required before saving.');
    });

    it('handles save error', async () => {
      saveDownloadAppAdminMock.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleSave('test-app');
      });

      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.error).toBe('Save failed');
      expect(form0.saving).toBe(false);
    });

    it('uses a safe error message when an API rejection is not an Error object', async () => {
      saveDownloadAppAdminMock.mockRejectedValueOnce('offline');
      const { result } = renderHook(() => useDownloadsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      await act(async () => { await result.current.handleSave('test-app'); });
      expect(result.current.forms[0]?.error).toBe('Failed to save app');
    });
  });

  describe('handleSaveAll', () => {
    it('saves all dirty forms', async () => {
      listDownloadAppsAdminMock.mockResolvedValue({ apps: [mockApp, mockApp2] });

      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Make changes to both apps
      act(() => {
        result.current.handleFieldChange('test-app', 'name', 'Updated 1');
        result.current.handleFieldChange('test-app-2', 'name', 'Updated 2');
      });

      expect(result.current.dirtyCount).toBe(2);

      await act(async () => {
        await result.current.handleSaveAll();
      });

      expect(saveDownloadAppAdminMock).toHaveBeenCalledTimes(2);
      expect(result.current.dirtyCount).toBe(0);
    });

    it('does nothing when no dirty forms', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleSaveAll();
      });

      expect(saveDownloadAppAdminMock).not.toHaveBeenCalled();
    });

    it('handles partial save failures', async () => {
      listDownloadAppsAdminMock.mockResolvedValue({ apps: [mockApp, mockApp2] });
      saveDownloadAppAdminMock
        .mockResolvedValueOnce(mockApp) // First save succeeds
        .mockRejectedValueOnce(new Error('Second save failed')); // Second save fails

      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('test-app', 'name', 'Updated 1');
        result.current.handleFieldChange('test-app-2', 'name', 'Updated 2');
      });

      await act(async () => {
        await result.current.handleSaveAll();
      });

      // First form saved successfully
      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.lastSavedAt).toBeDefined();
      // Second form has error
      const form1 = result.current.forms[1];
      assertDefined(form1, 'forms[1]');
      expect(form1.error).toBe('Second save failed');
    });

    it('keeps a non-Error bulk-save rejection actionable without failing the other form', async () => {
      listDownloadAppsAdminMock.mockResolvedValue({ apps: [mockApp, mockApp2] });
      saveDownloadAppAdminMock.mockResolvedValueOnce(mockApp).mockRejectedValueOnce('connection lost');
      const { result } = renderHook(() => useDownloadsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      act(() => {
        result.current.handleFieldChange('test-app', 'name', 'Saved');
        result.current.handleFieldChange('test-app-2', 'name', 'Retry later');
      });
      await act(async () => { await result.current.handleSaveAll(); });
      expect(result.current.forms[0]?.error).toBeUndefined();
      expect(result.current.forms[1]).toMatchObject({ saving: false, error: 'Failed to save app' });
      expect(result.current.savingAll).toBe(false);
    });
  });

  describe('dirtyCount', () => {
    it('counts dirty forms correctly', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.dirtyCount).toBe(0);

      act(() => {
        result.current.handleFieldChange('test-app', 'name', 'Changed');
      });

      expect(result.current.dirtyCount).toBe(1);

      act(() => {
        result.current.handleReset('test-app');
      });

      expect(result.current.dirtyCount).toBe(0);
    });
  });

  describe('downloadHealth', () => {
    it('computes health metrics', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.downloadHealth).toMatchObject({
        appCount: 1,
        hasApps: true,
        platformsConfigured: 1, // Windows is configured
        platformsMissing: 2, // Mac and Linux are not configured
        storefrontsConfigured: 1, // App Store is configured
      });
    });
  });

  describe('drag and drop', () => {
    it('clears drag state without changing order for same or unknown drag sources', async () => {
      const { result } = renderHook(() => useDownloadsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      const event = { dataTransfer: { getData: vi.fn(() => 'test-app'), effectAllowed: '', dropEffect: '', setData: vi.fn() }, preventDefault: vi.fn() } as unknown as React.DragEvent;
      act(() => { result.current.handleDragStart('test-app')(event); });
      act(() => { result.current.handleDrop('test-app')(event); });
      expect(result.current.draggingKey).toBeNull();
      expect(result.current.forms[0]?.values.appKey).toBe('test-app');
      const unknownEvent = {
        dataTransfer: { getData: vi.fn(() => 'unknown'), effectAllowed: '', dropEffect: '', setData: vi.fn() },
        preventDefault: vi.fn(),
      } as unknown as React.DragEvent;
      act(() => { result.current.handleDrop('test-app')(unknownEvent); });
      expect(result.current.forms).toHaveLength(1);
    });
    it('reorders forms on drop', async () => {
      listDownloadAppsAdminMock.mockResolvedValue({ apps: [mockApp, mockApp2] });

      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      let form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.appKey).toBe('test-app');
      let form1 = result.current.forms[1];
      assertDefined(form1, 'forms[1]');
      expect(form1.values.appKey).toBe('test-app-2');

      // Simulate drag and drop
      const mockDragEvent = {
        dataTransfer: {
          effectAllowed: '',
          dropEffect: '',
          getData: vi.fn().mockReturnValue('test-app-2'),
          setData: vi.fn(),
        },
        preventDefault: vi.fn(),
      } as unknown as React.DragEvent;

      act(() => {
        result.current.handleDragStart('test-app-2')(mockDragEvent);
      });

      expect(result.current.draggingKey).toBe('test-app-2');

      act(() => {
        result.current.handleDrop('test-app')(mockDragEvent);
      });

      // Order should be swapped
      form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      form1 = result.current.forms[1];
      assertDefined(form1, 'forms[1]');
      expect(form0.values.appKey).toBe('test-app-2');
      expect(form1.values.appKey).toBe('test-app');
      expect(form0.values.displayOrder).toBe(0);
      expect(form1.values.displayOrder).toBe(1);
    });

    it('clears drag state on drag end', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const mockDragEvent = {
        dataTransfer: {
          effectAllowed: '',
          setData: vi.fn(),
        },
        preventDefault: vi.fn(),
      } as unknown as React.DragEvent;

      act(() => {
        result.current.handleDragStart('test-app')(mockDragEvent);
      });

      expect(result.current.draggingKey).toBe('test-app');

      act(() => {
        result.current.handleDragEnd();
      });

      expect(result.current.draggingKey).toBeNull();
      expect(result.current.dragOverKey).toBeNull();
    });

    it('exposes a distinct drag-over target and clears it when the pointer leaves', async () => {
      listDownloadAppsAdminMock.mockResolvedValue({ apps: [mockApp, mockApp2] });
      const { result } = renderHook(() => useDownloadsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      const preventDefault = vi.fn();
      const event = {
        dataTransfer: { effectAllowed: '', dropEffect: '', getData: vi.fn(), setData: vi.fn() },
        preventDefault,
      } as unknown as React.DragEvent;
      act(() => { result.current.handleDragStart('test-app')(event); });
      act(() => { result.current.handleDragOver('test-app-2')(event); });
      expect(result.current.dragOverKey).toBe('test-app-2');
      expect(preventDefault).toHaveBeenCalled();
      act(() => { result.current.handleDragLeave(); });
      expect(result.current.dragOverKey).toBeNull();
    });
  });

  describe('loadApps', () => {
    it('reloads apps from API', async () => {
      const { result } = renderHook(() => useDownloadsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      listDownloadAppsAdminMock.mockResolvedValue({ apps: [mockApp2] });

      await act(async () => {
        await result.current.loadApps();
      });

      expect(listDownloadAppsAdminMock).toHaveBeenCalledTimes(2);
      expect(result.current.forms).toHaveLength(1);
      const form0 = result.current.forms[0];
      assertDefined(form0, 'forms[0]');
      expect(form0.values.appKey).toBe('test-app-2');
    });
  });
});
