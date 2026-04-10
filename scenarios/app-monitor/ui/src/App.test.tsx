import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, waitFor } from '@testing-library/react';
import App from './App';

const loadAppsMock = vi.fn(async () => undefined);
const updateAppMock = vi.fn();
const loadResourcesMock = vi.fn(async () => undefined);

vi.mock('@/hooks/useWebSocket', () => ({
  useAppWebSocket: () => ({
    connectionState: 'connected',
  }),
}));

vi.mock('@/state/appsStore', () => ({
  useAppsStore: (selector: (state: {
    loadApps: typeof loadAppsMock;
    updateApp: typeof updateAppMock;
  }) => unknown) => selector({
    loadApps: loadAppsMock,
    updateApp: updateAppMock,
  }),
}));

vi.mock('@/state/resourcesStore', () => ({
  useResourcesStore: (selector: (state: {
    loadResources: typeof loadResourcesMock;
  }) => unknown) => selector({
    loadResources: loadResourcesMock,
  }),
}));

vi.mock('@/components/Shell', async () => {
  const { Outlet } = await import('react-router-dom');
  return {
    default: () => <Outlet />,
  };
});

vi.mock('@/features/preview-workspace/components/PreviewWorkspaceView', () => ({
  default: () => <div data-testid="workspace-view">Workspace</div>,
}));

vi.mock('@/components/views/ResourceDetailView', () => ({
  default: () => <div data-testid="resource-view">Resource</div>,
}));

describe('App routing', () => {
  beforeEach(() => {
    loadAppsMock.mockClear();
    updateAppMock.mockClear();
    loadResourcesMock.mockClear();
    window.history.replaceState({}, '', '/');
  });

  it('routes root to workspace by default', async () => {
    render(<App />);

    await waitFor(() => {
      expect(window.location.pathname).toBe('/apps/workspace');
    });
  });

  it('redirects legacy preview route to workspace intent', async () => {
    window.history.replaceState({}, '', '/apps/scenario-1/preview?from=legacy');
    render(<App />);

    await waitFor(() => {
      expect(window.location.pathname).toBe('/apps/workspace');
    });

    const params = new URLSearchParams(window.location.search);
    expect(params.get('workspaceAppId')).toBe('scenario-1');
    expect(params.get('workspaceMode')).toBe('replace-focused');
    expect(params.get('from')).toBe('legacy');
  });

  it('redirects legacy logs route to workspace logs intent', async () => {
    window.history.replaceState({}, '', '/logs/scenario-2?source=legacy');
    render(<App />);

    await waitFor(() => {
      expect(window.location.pathname).toBe('/apps/workspace');
    });

    const params = new URLSearchParams(window.location.search);
    expect(params.get('workspaceAppId')).toBe('scenario-2');
    expect(params.get('workspaceMode')).toBe('replace-focused');
    expect(params.get('paneLogs')).toBe('1');
    expect(params.get('source')).toBe('legacy');
  });
});
