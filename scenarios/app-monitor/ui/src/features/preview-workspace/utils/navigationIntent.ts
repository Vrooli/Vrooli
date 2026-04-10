export const WORKSPACE_INTENT_APP_ID_KEY = 'workspaceAppId';
export const WORKSPACE_INTENT_MODE_KEY = 'workspaceMode';

export type WorkspaceOpenMode = 'replace-focused' | 'add-pane';

export interface WorkspaceIntent {
  appId: string;
  mode: WorkspaceOpenMode;
}

const isWorkspaceOpenMode = (value: string | null): value is WorkspaceOpenMode => (
  value === 'replace-focused' || value === 'add-pane'
);

const normalizeAppId = (value: string | null): string | null => {
  if (!value) {
    return null;
  }
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
};

export const readWorkspaceIntent = (searchParams: URLSearchParams): WorkspaceIntent | null => {
  const appId = normalizeAppId(searchParams.get(WORKSPACE_INTENT_APP_ID_KEY));
  const modeRaw = searchParams.get(WORKSPACE_INTENT_MODE_KEY);
  if (!appId || !isWorkspaceOpenMode(modeRaw)) {
    return null;
  }
  return {
    appId,
    mode: modeRaw,
  };
};

export const clearWorkspaceIntent = (searchParams: URLSearchParams): URLSearchParams => {
  const next = new URLSearchParams(searchParams);
  next.delete(WORKSPACE_INTENT_APP_ID_KEY);
  next.delete(WORKSPACE_INTENT_MODE_KEY);
  return next;
};

