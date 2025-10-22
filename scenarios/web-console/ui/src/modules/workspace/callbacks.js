const workspaceCallbacks = {
  queueSessionOverviewRefresh: null,
};

export function configureWorkspace(options = {}) {
  workspaceCallbacks.queueSessionOverviewRefresh =
    typeof options.queueSessionOverviewRefresh === "function"
      ? options.queueSessionOverviewRefresh
      : null;
}

export function queueSessionOverviewRefresh(delay = 0) {
  workspaceCallbacks.queueSessionOverviewRefresh?.(delay);
}

export { workspaceCallbacks };
