import { replayConfigClient } from '@/api/replayConfig';
import type { ReplayStyleConfig } from '../model';
import { normalizeReplayStyle, REPLAY_STYLE_VERSION } from '../model';

export interface ReplayStylePayload {
  style: ReplayStyleConfig;
  extra: Record<string, unknown>;
}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === 'object' && !Array.isArray(value);

export const fetchReplayStylePayload = async (): Promise<ReplayStylePayload | null> => {
  const response = await replayConfigClient.get({});
  const config = response.config as Record<string, unknown> | undefined;
  if (!config || !isPlainObject(config)) {
    return null;
  }
  if ('style' in config && isPlainObject(config.style)) {
    const style = normalizeReplayStyle(config.style);
    const extra = isPlainObject(config.extra) ? config.extra : {};
    return { style, extra };
  }
  return null;
};

export const fetchReplayStyleConfig = async (): Promise<ReplayStyleConfig | null> => {
  const payload = await fetchReplayStylePayload();
  return payload?.style ?? null;
};

export const persistReplayStyleConfig = async (
  config: ReplayStyleConfig,
  extraConfig?: Record<string, unknown>,
) => {
  const payload = {
    version: REPLAY_STYLE_VERSION,
    style: config as unknown as Record<string, unknown>,
    extra: extraConfig ?? {},
  };
  await replayConfigClient.put({ config: payload as unknown as Record<string, never> });
};
