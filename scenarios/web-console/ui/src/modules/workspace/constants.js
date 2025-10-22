export const debugWorkspace =
  typeof window !== "undefined" &&
  window.__WEB_CONSOLE_DEBUG__ &&
  window.__WEB_CONSOLE_DEBUG__.workspace === true;

export const workspaceAnomalyState = {
  toastShown: false,
};

export const IDLE_TIMEOUT_DEFAULT_MINUTES = 15;
export const IDLE_TIMEOUT_MAX_MINUTES = 1440;
