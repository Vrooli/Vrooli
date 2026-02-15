import { fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AppPreviewToolbar from './AppPreviewToolbar';

const { openOverlayMock } = vi.hoisted(() => ({
  openOverlayMock: vi.fn(),
}));

vi.mock('@/hooks/useOverlayRouter', () => ({
  useOverlayRouter: () => ({
    overlay: null,
    openOverlay: openOverlayMock,
    closeOverlay: vi.fn(),
  }),
}));

vi.mock('@/hooks/useDraggablePosition', () => ({
  useDraggablePosition: () => ({
    elementRef: { current: null },
    floatingStyle: undefined,
    isDragging: false,
    pointerHandlers: {
      onPointerDown: vi.fn(),
      onPointerMove: vi.fn(),
      onPointerUp: vi.fn(),
      onPointerCancel: vi.fn(),
    },
    handleClickCapture: vi.fn(),
  }),
}));

describe('AppPreviewToolbar', () => {
  const setMatchMedia = (isSmallScreen: boolean) => {
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: query.includes('max-width: 640px') ? isSmallScreen : false,
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });
  };

  beforeEach(() => {
    openOverlayMock.mockReset();
    setMatchMedia(false);
  });

  const buildSingleSuggestionSection = (suggestion: string) => () => ([
    {
      id: 'recent-urls',
      label: 'Recent URLs',
      items: [{
        id: `item-${suggestion}`,
        label: suggestion,
        value: suggestion,
        kind: 'recent-url' as const,
      }],
    },
  ]);

  it('does not blur-commit stale input when selecting a URL suggestion', () => {
    const onPreviewUrlInputBlur = vi.fn();
    const onSelectUrlSuggestion = vi.fn();
    const suggestion = 'http://localhost:4310/settings';

    render(
      <AppPreviewToolbar
        canGoBack={false}
        canGoForward={false}
        onGoBack={vi.fn()}
        onGoForward={vi.fn()}
        onRefresh={vi.fn()}
        isRefreshing={false}
        onOpenDetails={vi.fn()}
        previewUrlInput=""
        onPreviewUrlInputChange={vi.fn()}
        onPreviewUrlInputBlur={onPreviewUrlInputBlur}
        onPreviewUrlInputKeyDown={vi.fn()}
        onOpenInNewTab={vi.fn()}
        openPreviewTarget={null}
        urlStatusClass="running"
        urlStatusTitle="running"
        hasDetailsWarning={false}
        hasCurrentApp={true}
        isAppRunning={true}
        pendingAction={null}
        actionInProgress={false}
        toggleActionLabel="Stop"
        onToggleApp={vi.fn()}
        restartActionLabel="Restart"
        onRestartApp={vi.fn()}
        onToggleLogs={vi.fn()}
        areLogsVisible={false}
        onReportIssue={vi.fn()}
        appStatusLabel="Running"
        isFullView={false}
        onToggleFullView={vi.fn()}
        isDeviceEmulationActive={false}
        onToggleDeviceEmulation={vi.fn()}
        canInspect={false}
        isInspecting={false}
        onToggleInspect={vi.fn()}
        menuPortalContainer={null}
        canOpenTabsOverlay={true}
        previewInteractionSignal={0}
        issueCaptureCount={0}
        showDetailsButton={false}
        showLifecycleMenu={false}
        showDevMenu={false}
        buildUrlSuggestionSections={buildSingleSuggestionSection(suggestion)}
        onSelectUrlSuggestion={onSelectUrlSuggestion}
      />,
    );

    const input = screen.getByLabelText('Preview URL');
    fireEvent.focus(input);

    const suggestionButton = screen.getByRole('button', { name: suggestion });
    fireEvent.pointerDown(suggestionButton);
    fireEvent.blur(input);
    fireEvent.click(suggestionButton);

    expect(onSelectUrlSuggestion).toHaveBeenCalledWith(suggestion);
    expect(onSelectUrlSuggestion).toHaveBeenCalledTimes(1);
    expect(onPreviewUrlInputBlur).not.toHaveBeenCalled();
  });

  it('still suppresses blur commit when blur happens after suggestion click', () => {
    const onPreviewUrlInputBlur = vi.fn();
    const onSelectUrlSuggestion = vi.fn();
    const suggestion = 'http://localhost:4310/settings';

    render(
      <AppPreviewToolbar
        canGoBack={false}
        canGoForward={false}
        onGoBack={vi.fn()}
        onGoForward={vi.fn()}
        onRefresh={vi.fn()}
        isRefreshing={false}
        onOpenDetails={vi.fn()}
        previewUrlInput=""
        onPreviewUrlInputChange={vi.fn()}
        onPreviewUrlInputBlur={onPreviewUrlInputBlur}
        onPreviewUrlInputKeyDown={vi.fn()}
        onOpenInNewTab={vi.fn()}
        openPreviewTarget={null}
        urlStatusClass="running"
        urlStatusTitle="running"
        hasDetailsWarning={false}
        hasCurrentApp={true}
        isAppRunning={true}
        pendingAction={null}
        actionInProgress={false}
        toggleActionLabel="Stop"
        onToggleApp={vi.fn()}
        restartActionLabel="Restart"
        onRestartApp={vi.fn()}
        onToggleLogs={vi.fn()}
        areLogsVisible={false}
        onReportIssue={vi.fn()}
        appStatusLabel="Running"
        isFullView={false}
        onToggleFullView={vi.fn()}
        isDeviceEmulationActive={false}
        onToggleDeviceEmulation={vi.fn()}
        canInspect={false}
        isInspecting={false}
        onToggleInspect={vi.fn()}
        menuPortalContainer={null}
        canOpenTabsOverlay={true}
        previewInteractionSignal={0}
        issueCaptureCount={0}
        showDetailsButton={false}
        showLifecycleMenu={false}
        showDevMenu={false}
        buildUrlSuggestionSections={buildSingleSuggestionSection(suggestion)}
        onSelectUrlSuggestion={onSelectUrlSuggestion}
      />,
    );

    const input = screen.getByLabelText('Preview URL');
    fireEvent.focus(input);

    const suggestionButton = screen.getByRole('button', { name: suggestion });
    fireEvent.pointerDown(suggestionButton);
    fireEvent.click(suggestionButton);
    fireEvent.blur(input);

    expect(onSelectUrlSuggestion).toHaveBeenCalledWith(suggestion);
    expect(onSelectUrlSuggestion).toHaveBeenCalledTimes(1);
    expect(onPreviewUrlInputBlur).not.toHaveBeenCalled();
  });

  it('submits typed URL on Enter when no suggestion is explicitly selected', () => {
    const onSelectUrlSuggestion = vi.fn();
    const onPreviewUrlInputKeyDown = vi.fn();
    const suggestion = 'http://localhost:4310/settings';

    function Harness() {
      const [value, setValue] = useState('');
      return (
        <AppPreviewToolbar
          canGoBack={false}
          canGoForward={false}
          onGoBack={vi.fn()}
          onGoForward={vi.fn()}
          onRefresh={vi.fn()}
          isRefreshing={false}
          onOpenDetails={vi.fn()}
          previewUrlInput={value}
          onPreviewUrlInputChange={(event) => setValue(event.target.value)}
          onPreviewUrlInputBlur={vi.fn()}
          onPreviewUrlInputKeyDown={onPreviewUrlInputKeyDown}
          onOpenInNewTab={vi.fn()}
          openPreviewTarget={null}
          urlStatusClass="running"
          urlStatusTitle="running"
          hasDetailsWarning={false}
          hasCurrentApp={true}
          isAppRunning={true}
          pendingAction={null}
          actionInProgress={false}
          toggleActionLabel="Stop"
          onToggleApp={vi.fn()}
          restartActionLabel="Restart"
          onRestartApp={vi.fn()}
          onToggleLogs={vi.fn()}
          areLogsVisible={false}
          onReportIssue={vi.fn()}
          appStatusLabel="Running"
          isFullView={false}
          onToggleFullView={vi.fn()}
          isDeviceEmulationActive={false}
          onToggleDeviceEmulation={vi.fn()}
          canInspect={false}
          isInspecting={false}
          onToggleInspect={vi.fn()}
          menuPortalContainer={null}
          canOpenTabsOverlay={true}
          previewInteractionSignal={0}
          issueCaptureCount={0}
          showDetailsButton={false}
          showLifecycleMenu={false}
          showDevMenu={false}
          buildUrlSuggestionSections={buildSingleSuggestionSection(suggestion)}
          onSelectUrlSuggestion={onSelectUrlSuggestion}
        />
      );
    }

    render(<Harness />);

    const input = screen.getByLabelText('Preview URL');
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: 'vrooli.com' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onSelectUrlSuggestion).not.toHaveBeenCalled();
    expect(onPreviewUrlInputKeyDown).toHaveBeenCalledTimes(1);
  });

  it('auto-selects the first suggestion on Enter for non-URL text input', () => {
    const onSelectUrlSuggestion = vi.fn();
    const suggestion = 'http://localhost:3000/apps/deploy-manager/proxy/';

    render(
      <AppPreviewToolbar
        canGoBack={false}
        canGoForward={false}
        onGoBack={vi.fn()}
        onGoForward={vi.fn()}
        onRefresh={vi.fn()}
        isRefreshing={false}
        onOpenDetails={vi.fn()}
        previewUrlInput="depl"
        onPreviewUrlInputChange={vi.fn()}
        onPreviewUrlInputBlur={vi.fn()}
        onPreviewUrlInputKeyDown={vi.fn()}
        onOpenInNewTab={vi.fn()}
        openPreviewTarget={null}
        urlStatusClass="running"
        urlStatusTitle="running"
        hasDetailsWarning={false}
        hasCurrentApp={true}
        isAppRunning={true}
        pendingAction={null}
        actionInProgress={false}
        toggleActionLabel="Stop"
        onToggleApp={vi.fn()}
        restartActionLabel="Restart"
        onRestartApp={vi.fn()}
        onToggleLogs={vi.fn()}
        areLogsVisible={false}
        onReportIssue={vi.fn()}
        appStatusLabel="Running"
        isFullView={false}
        onToggleFullView={vi.fn()}
        isDeviceEmulationActive={false}
        onToggleDeviceEmulation={vi.fn()}
        canInspect={false}
        isInspecting={false}
        onToggleInspect={vi.fn()}
        menuPortalContainer={null}
        canOpenTabsOverlay={true}
        previewInteractionSignal={0}
        issueCaptureCount={0}
        showDetailsButton={false}
        showLifecycleMenu={false}
        showDevMenu={false}
        onSelectUrlSuggestion={onSelectUrlSuggestion}
        buildUrlSuggestionSections={() => ([
          {
            id: 'scenario-matches',
            label: 'Scenario matches',
            items: [{
              id: 'match-1',
              label: 'deploy-manager',
              value: suggestion,
              kind: 'scenario',
            }],
          },
        ])}
      />,
    );

    const input = screen.getByLabelText('Preview URL');
    fireEvent.focus(input);
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onSelectUrlSuggestion).toHaveBeenCalledWith(suggestion);
  });

  it('opens scenario selector on Enter when it is the only available option', () => {
    const onOpenScenarioSelector = vi.fn();
    render(
      <AppPreviewToolbar
        canGoBack={false}
        canGoForward={false}
        onGoBack={vi.fn()}
        onGoForward={vi.fn()}
        onRefresh={vi.fn()}
        isRefreshing={false}
        onOpenDetails={vi.fn()}
        previewUrlInput="scenario-com"
        onPreviewUrlInputChange={vi.fn()}
        onPreviewUrlInputBlur={vi.fn()}
        onPreviewUrlInputKeyDown={vi.fn()}
        onOpenInNewTab={vi.fn()}
        openPreviewTarget={null}
        urlStatusClass="running"
        urlStatusTitle="running"
        hasDetailsWarning={false}
        hasCurrentApp={true}
        isAppRunning={true}
        pendingAction={null}
        actionInProgress={false}
        toggleActionLabel="Stop"
        onToggleApp={vi.fn()}
        restartActionLabel="Restart"
        onRestartApp={vi.fn()}
        onToggleLogs={vi.fn()}
        areLogsVisible={false}
        onReportIssue={vi.fn()}
        appStatusLabel="Running"
        isFullView={false}
        onToggleFullView={vi.fn()}
        isDeviceEmulationActive={false}
        onToggleDeviceEmulation={vi.fn()}
        canInspect={false}
        isInspecting={false}
        onToggleInspect={vi.fn()}
        menuPortalContainer={null}
        canOpenTabsOverlay={true}
        previewInteractionSignal={0}
        issueCaptureCount={0}
        showDetailsButton={false}
        showLifecycleMenu={false}
        showDevMenu={false}
        onSelectUrlSuggestion={vi.fn()}
        buildUrlSuggestionSections={() => []}
        onOpenScenarioSelector={onOpenScenarioSelector}
      />,
    );

    const input = screen.getByLabelText('Preview URL');
    fireEvent.focus(input);
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onOpenScenarioSelector).toHaveBeenCalledTimes(1);
  });

  it('renders grouped suggestion sections from shared discovery data', () => {
    const onSelectUrlSuggestion = vi.fn();

    render(
      <AppPreviewToolbar
        canGoBack={false}
        canGoForward={false}
        onGoBack={vi.fn()}
        onGoForward={vi.fn()}
        onRefresh={vi.fn()}
        isRefreshing={false}
        onOpenDetails={vi.fn()}
        previewUrlInput="workspace"
        onPreviewUrlInputChange={vi.fn()}
        onPreviewUrlInputBlur={vi.fn()}
        onPreviewUrlInputKeyDown={vi.fn()}
        onOpenInNewTab={vi.fn()}
        openPreviewTarget={null}
        urlStatusClass="running"
        urlStatusTitle="running"
        hasDetailsWarning={false}
        hasCurrentApp={true}
        isAppRunning={true}
        pendingAction={null}
        actionInProgress={false}
        toggleActionLabel="Stop"
        onToggleApp={vi.fn()}
        restartActionLabel="Restart"
        onRestartApp={vi.fn()}
        onToggleLogs={vi.fn()}
        areLogsVisible={false}
        onReportIssue={vi.fn()}
        appStatusLabel="Running"
        isFullView={false}
        onToggleFullView={vi.fn()}
        isDeviceEmulationActive={false}
        onToggleDeviceEmulation={vi.fn()}
        canInspect={false}
        isInspecting={false}
        onToggleInspect={vi.fn()}
        menuPortalContainer={null}
        canOpenTabsOverlay={true}
        previewInteractionSignal={0}
        issueCaptureCount={0}
        showDetailsButton={false}
        showLifecycleMenu={false}
        showDevMenu={false}
        onSelectUrlSuggestion={onSelectUrlSuggestion}
        buildUrlSuggestionSections={() => ([
          {
            id: 'running-scenarios',
            label: 'Running scenarios',
            items: [
              {
                id: 'running-1',
                label: 'workspace-manager',
                value: 'http://localhost:3000/apps/workspace-manager/proxy/',
                detail: 'Running',
                kind: 'running-scenario',
              },
            ],
          },
        ])}
      />,
    );

    const input = screen.getByLabelText('Preview URL');
    fireEvent.focus(input);
    expect(screen.getByText('Running scenarios')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /workspace-manager/i }));
    expect(onSelectUrlSuggestion).toHaveBeenCalledWith('http://localhost:3000/apps/workspace-manager/proxy/');
  });

  it('keeps URL input visible and uses compact navigation on small screens', () => {
    setMatchMedia(true);

    render(
      <AppPreviewToolbar
        canGoBack={false}
        canGoForward={false}
        onGoBack={vi.fn()}
        onGoForward={vi.fn()}
        onRefresh={vi.fn()}
        isRefreshing={false}
        onOpenDetails={vi.fn()}
        previewUrlInput="http://localhost:4310"
        onPreviewUrlInputChange={vi.fn()}
        onPreviewUrlInputBlur={vi.fn()}
        onPreviewUrlInputKeyDown={vi.fn()}
        onOpenInNewTab={vi.fn()}
        openPreviewTarget="http://localhost:4310"
        urlStatusClass="running"
        urlStatusTitle="running"
        hasDetailsWarning={false}
        hasCurrentApp={true}
        isAppRunning={true}
        pendingAction={null}
        actionInProgress={false}
        toggleActionLabel="Stop"
        onToggleApp={vi.fn()}
        restartActionLabel="Restart"
        onRestartApp={vi.fn()}
        onToggleLogs={vi.fn()}
        areLogsVisible={false}
        onReportIssue={vi.fn()}
        appStatusLabel="Running"
        isFullView={false}
        onToggleFullView={vi.fn()}
        isDeviceEmulationActive={false}
        onToggleDeviceEmulation={vi.fn()}
        canInspect={false}
        isInspecting={false}
        onToggleInspect={vi.fn()}
        menuPortalContainer={null}
        canOpenTabsOverlay={true}
        previewInteractionSignal={0}
        issueCaptureCount={0}
        showDetailsButton={true}
        showLifecycleMenu={true}
        showDevMenu={false}
      />,
    );

    expect(screen.getByLabelText('Preview URL')).toBeInTheDocument();
    expect(screen.getByLabelText('Navigation actions')).toBeInTheDocument();
    expect(screen.queryByLabelText(/^Go back$/)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/Lifecycle actions/i)).not.toBeInTheDocument();
  });

  it('opens tab switcher from fullscreen toolbar button when enabled', () => {
    render(
      <AppPreviewToolbar
        canGoBack={false}
        canGoForward={false}
        onGoBack={vi.fn()}
        onGoForward={vi.fn()}
        onRefresh={vi.fn()}
        isRefreshing={false}
        onOpenDetails={vi.fn()}
        previewUrlInput=""
        onPreviewUrlInputChange={vi.fn()}
        onPreviewUrlInputBlur={vi.fn()}
        onPreviewUrlInputKeyDown={vi.fn()}
        onOpenInNewTab={vi.fn()}
        openPreviewTarget={null}
        urlStatusClass="running"
        urlStatusTitle="running"
        hasDetailsWarning={false}
        hasCurrentApp={true}
        isAppRunning={true}
        pendingAction={null}
        actionInProgress={false}
        toggleActionLabel="Stop"
        onToggleApp={vi.fn()}
        restartActionLabel="Restart"
        onRestartApp={vi.fn()}
        onToggleLogs={vi.fn()}
        areLogsVisible={false}
        onReportIssue={vi.fn()}
        appStatusLabel="Running"
        isFullView={true}
        onToggleFullView={vi.fn()}
        isDeviceEmulationActive={false}
        onToggleDeviceEmulation={vi.fn()}
        canInspect={false}
        isInspecting={false}
        onToggleInspect={vi.fn()}
        menuPortalContainer={null}
        canOpenTabsOverlay={true}
        previewInteractionSignal={0}
        issueCaptureCount={0}
        showDetailsButton={false}
        showLifecycleMenu={false}
        showDevMenu={false}
      />,
    );

    fireEvent.click(screen.getByLabelText('Open tabs switcher'));
    expect(openOverlayMock).toHaveBeenCalledWith('tabs', {
      params: { segment: 'apps' },
    });
  });
});
