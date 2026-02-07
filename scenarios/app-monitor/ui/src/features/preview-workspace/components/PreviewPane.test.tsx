import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import type { App } from '@/types';
import PreviewPane from './PreviewPane';

vi.mock('@/hooks/useIframeBridge', () => ({
  useIframeBridge: () => ({
    state: {
      isSupported: false,
      href: null,
      canGoBack: false,
      canGoForward: false,
      isReady: false,
      caps: [],
    },
    childOrigin: null,
    sendNav: vi.fn(),
    runComplianceCheck: vi.fn().mockResolvedValue({ ok: true, failures: [], checkedAt: Date.now() }),
    resetState: vi.fn(),
    requestScreenshot: vi.fn(),
    logState: null,
    networkState: null,
    subscribeLogs: vi.fn(() => () => {}),
    getRecentLogs: vi.fn(() => []),
    requestLogBatch: vi.fn().mockResolvedValue([]),
    configureLogs: vi.fn(() => false),
    subscribeNetwork: vi.fn(() => () => {}),
    getRecentNetworkEvents: vi.fn(() => []),
    requestNetworkBatch: vi.fn().mockResolvedValue([]),
    configureNetwork: vi.fn(() => false),
    inspectState: {
      supported: false,
      active: false,
      lastReason: null,
      hover: null,
      result: null,
      error: null,
    },
    startInspect: vi.fn(() => false),
    stopInspect: vi.fn(() => false),
    setInspectTargetIndex: vi.fn(() => false),
    shiftInspectTarget: vi.fn(() => false),
  }),
}));

const createApp = (): App => ({
  id: 'scenario-1',
  name: 'Scenario One',
  scenario_name: 'scenario-one',
  path: '/tmp/scenario-one',
  created_at: '2026-02-07T00:00:00Z',
  updated_at: '2026-02-07T00:00:00Z',
  status: 'running',
  port_mappings: { UI_PORT: 4310 },
  environment: {},
  config: {},
});

describe('PreviewPane', () => {
  it('renders the shared preview toolbar and supports removing pane', async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    const app = createApp();

    const { container } = render(
      <MemoryRouter>
        <PreviewPane
          paneId="pane-1"
          appId={app.id}
          apps={[app]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={onRemove}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(container.querySelector('.preview-toolbar')).not.toBeNull();
    expect(screen.queryByRole('button', { name: /remove pane/i })).not.toBeNull();

    await user.click(screen.getByRole('button', { name: /remove pane/i }));
    expect(onRemove).toHaveBeenCalledWith('pane-1');
  });
});
