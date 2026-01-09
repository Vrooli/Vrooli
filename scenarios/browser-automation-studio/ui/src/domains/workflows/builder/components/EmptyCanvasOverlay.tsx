export function EmptyCanvasOverlay() {
  return (
    <div
      className="absolute inset-0 flex items-center justify-center pointer-events-none z-10"
      data-testid="workflow-empty-state"
    >
      <div className="text-center max-w-md px-6">
        <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-flow-node/80 border border-flow-border/60 flex items-center justify-center">
          <svg
            className="w-8 h-8 text-flow-text-muted"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M12 4v16m8-8H4"
            />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-flow-text mb-2">
          Start building your workflow
        </h3>
        <p className="text-sm text-flow-text-muted mb-4">
          Drag a <span className="text-blue-400 font-medium">Navigate</span> node from the
          sidebar to begin, then chain additional actions like Click, Type, or Screenshot.
        </p>
        <div className="flex items-center justify-center gap-2 text-xs text-flow-text-secondary">
          <kbd className="px-2 py-1 bg-flow-node rounded border border-flow-border/60 font-mono">
            Drag
          </kbd>
          <span>from sidebar</span>
          <span className="text-flow-text-muted">→</span>
          <span>drop here</span>
        </div>
      </div>
    </div>
  );
}
