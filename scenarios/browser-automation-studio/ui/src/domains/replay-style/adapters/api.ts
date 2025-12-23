import { getConfig } from '@/config';
import type { ReplayStyleConfig } from '../model';
import { normalizeReplayStyle, REPLAY_STYLE_VERSION } from '../model';

const STYLE_KEYS = new Set([
  'chromeTheme',
  'backgroundTheme',
  'background',
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
  extra: Record<string, unknown>;
}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === 'object' && !Array.isArray(value);

const parseLegacyPayload = (config: Record<string, unknown>): ReplayStylePayload => {
  const style = normalizeReplayStyle(config);
  const extra = Object.fromEntries(
    Object.entries(config).filter(([key]) => !STYLE_KEYS.has(key)),
  );
  return { style, extra };
};

export const fetchReplayStylePayload = async (): Promise<ReplayStylePayload | null> => {
  const { API_URL } = await getConfig();
  const response = await fetch(`${API_URL}/replay-config`);
  if (!response.ok) {
    throw new Error(`Failed to fetch replay config (${response.status})`);
  }
  const payload = (await response.json()) as { config?: Record<string, unknown> };
  if (!payload.config || !isPlainObject(payload.config)) {
    return null;
  }
  const config = payload.config;
  if ('style' in config && isPlainObject(config.style)) {
    const style = normalizeReplayStyle(config.style);
    const extra = isPlainObject(config.extra) ? config.extra : {};
    return { style, extra };
  }
  return parseLegacyPayload(config);
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
    version: REPLAY_STYLE_VERSION,
    style: config,
    extra: extraConfig ?? {},
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
