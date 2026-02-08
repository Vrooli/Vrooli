import { fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import AppPreviewToolbar from './AppPreviewToolbar';

vi.mock('@/hooks/useOverlayRouter', () => ({
  useOverlayRouter: () => ({
    overlay: null,
    openOverlay: vi.fn(),
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
        urlSuggestions={[suggestion]}
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
        urlSuggestions={[suggestion]}
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
          urlSuggestions={[suggestion]}
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
});
