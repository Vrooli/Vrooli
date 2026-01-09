const callbacks = {
  onActiveTabNeedsUpdate: null,
  onSessionActionsChanged: null,
  queueSessionOverviewRefresh: null,
};

export function configureSessionService(options = {}) {
  callbacks.onActiveTabNeedsUpdate =
    typeof options.onActiveTabNeedsUpdate === "function"
      ? options.onActiveTabNeedsUpdate
      : null;
  callbacks.onSessionActionsChanged =
    typeof options.onSessionActionsChanged === "function"
      ? options.onSessionActionsChanged
      : null;
  callbacks.queueSessionOverviewRefresh =
    typeof options.queueSessionOverviewRefresh === "function"
      ? options.queueSessionOverviewRefresh
      : null;
}

export function notifyActiveTabUpdate() {
  callbacks.onActiveTabNeedsUpdate?.();
}

export function notifySessionActionsChanged() {
  callbacks.onSessionActionsChanged?.();
}

export function queueSessionOverviewRefresh(delay = 0) {
  callbacks.queueSessionOverviewRefresh?.(delay);
}
