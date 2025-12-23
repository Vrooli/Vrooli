import { getConfig } from '@/config';
import type { ReplayStyleConfig } from '../model';
import { normalizeReplayStyle } from '../model';

const STYLE_KEYS = new Set([
  'chromeTheme',
  'backgroundTheme',
  'cursorTheme',
  'cursorInitialPosition',
  'cursorClickAnimation',
  'cursorScale',
  'browserScale',
  'version',
  'replayChromeTheme',
  'replayBackgroundTheme',
  'replayCursorTheme',
  'replayCursorInitialPosition',
  'replayCursorClickAnimation',
  'replayCursorScale',
  'replayBrowserScale',
  'chrome_theme',
  'background_theme',
  'cursor_theme',
  'cursor_initial_position',
  'cursor_click_animation',
  'cursor_scale',
  'browser_scale',
]);

export interface ReplayStylePayload {
  style: ReplayStyleConfig;
  extraConfig: Record<string, unknown>;
}

export const fetchReplayStylePayload = async (): Promise<ReplayStylePayload | null> => {
  const { API_URL } = await getConfig();
  const response = await fetch(`${API_URL}/replay-config`);
  if (!response.ok) {
    throw new Error(`Failed to fetch replay config (${response.status})`);
  }
  const payload = (await response.json()) as { config?: Record<string, unknown> };
  if (!payload.config) {
    return null;
  }
  const style = normalizeReplayStyle(payload.config);
  const extraConfig = Object.fromEntries(
    Object.entries(payload.config).filter(([key]) => !STYLE_KEYS.has(key)),
  );
  return { style, extraConfig };
};

export const fetchReplayStyleConfig = async (): Promise<ReplayStyleConfig | null> => {
  const payload = await fetchReplayStylePayload();
  return payload?.style ?? null;
};

export const persistReplayStyleConfig = async (
  config: ReplayStyleConfig,
  extraConfig?: Record<string, unknown>,
) => {
  const { API_URL } = await getConfig();
  const payload = {
    chromeTheme: config.chromeTheme,
    backgroundTheme: config.backgroundTheme,
    cursorTheme: config.cursorTheme,
    cursorInitialPosition: config.cursorInitialPosition,
    cursorClickAnimation: config.cursorClickAnimation,
    cursorScale: config.cursorScale,
    browserScale: config.browserScale,
    version: config.version,
    ...(extraConfig ?? {}),
  };
  const response = await fetch(`${API_URL}/replay-config`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ config: payload }),
  });
  if (!response.ok) {
    throw new Error(`Failed to persist replay config (${response.status})`);
  }
};
