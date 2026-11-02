interface RefreshControlProps {
  paused: boolean;
  onTogglePause: () => void;
  onRefresh: () => void;
  isLoading: boolean;
  atTop: boolean;
  globalPaused?: boolean;
}

export const RefreshControl = ({
  paused,
  onTogglePause,
  onRefresh,
  isLoading,
  atTop,
  globalPaused = false,
}: RefreshControlProps) => (
  <div className="flex-row-center gap-sm">
    <button
      type="button"
      className="header-button"
      onClick={onRefresh}
      disabled={isLoading}
      title="Refresh now"
    >
      {isLoading ? 'Refreshing…' : 'Refresh'}
    </button>
    <button
      type="button"
      className="header-button"
      onClick={onTogglePause}
      disabled={globalPaused}
      aria-pressed={paused}
      title={globalPaused ? 'Resume the shared view first' : paused ? 'Resume log updates' : 'Pause log updates'}
    >
      {paused ? 'Resume log updates' : 'Pause log updates'}
    </button>
    {!atTop && !paused && (
      <span className="text-xs text-muted" title="Auto-refresh paused while scrolled">
        Updates paused while reading; this does not stop host collection.
      </span>
    )}
  </div>
);
