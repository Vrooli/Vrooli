interface RefreshControlProps {
  paused: boolean;
  onTogglePause: () => void;
  onRefresh: () => void;
  isLoading: boolean;
  atTop: boolean;
}

export const RefreshControl = ({
  paused,
  onTogglePause,
  onRefresh,
  isLoading,
  atTop,
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
      title={paused ? 'Resume auto-refresh' : 'Pause auto-refresh'}
    >
      {paused ? 'Resume' : 'Pause'}
    </button>
    {!atTop && !paused && (
      <span className="text-xs text-muted" title="Auto-refresh paused while scrolled">
        (auto-paused)
      </span>
    )}
  </div>
);
