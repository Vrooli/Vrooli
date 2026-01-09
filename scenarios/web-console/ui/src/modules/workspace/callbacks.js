// @ts-check

/** @type {{ queueSessionOverviewRefresh: ((delay?: number) => void) | null }} */
const workspaceCallbacks = {
  queueSessionOverviewRefresh: null,
};

/**
 * @param {{ queueSessionOverviewRefresh?: (delay?: number) => void }} [options]
 */
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
