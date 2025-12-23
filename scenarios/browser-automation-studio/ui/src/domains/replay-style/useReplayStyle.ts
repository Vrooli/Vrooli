import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { logger } from '@/utils/logger';
import type { ReplayBackgroundSource, ReplayBackgroundTheme, ReplayStyleConfig, ReplayStyleOverrides } from './model';
import { normalizeReplayStyle, resolveReplayStyle } from './model';
import { readReplayStyleFromStorage, writeReplayStyleToStorage } from './adapters/storage';
import { fetchReplayStylePayload, persistReplayStyleConfig } from './adapters/api';

interface UseReplayStyleParams {
  executionId?: string;
  overrides?: ReplayStyleOverrides;
  extraConfig?: Record<string, unknown>;
}

export interface ReplayStyleController {
  style: ReplayStyleConfig;
  setStyle: (updater: ReplayStyleOverrides | ((prev: ReplayStyleConfig) => ReplayStyleOverrides)) => void;
  setChromeTheme: (value: ReplayStyleConfig['chromeTheme']) => void;
  setBackground: (value: ReplayBackgroundSource) => void;
  setBackgroundTheme: (value: ReplayBackgroundTheme) => void;
  setCursorTheme: (value: ReplayStyleConfig['cursorTheme']) => void;
  setCursorInitialPosition: (value: ReplayStyleConfig['cursorInitialPosition']) => void;
  setCursorClickAnimation: (value: ReplayStyleConfig['cursorClickAnimation']) => void;
  setCursorScale: (value: number) => void;
  setBrowserScale: (value: number) => void;
  serverExtraConfig: Record<string, unknown> | null;
  isServerReady: boolean;
}

const serializePayload = (style: ReplayStyleConfig, extraConfig?: Record<string, unknown>) =>
  JSON.stringify({ version: style.version, style, extra: extraConfig ?? {} });

export function useReplayStyle({ executionId, overrides, extraConfig }: UseReplayStyleParams = {}): ReplayStyleController {
  const initialStyle = useMemo(() => {
    const stored = readReplayStyleFromStorage();
    return resolveReplayStyle({ stored, overrides });
  }, [overrides]);

  const [style, setStyleState] = useState<ReplayStyleConfig>(initialStyle);
  const [serverExtraConfig, setServerExtraConfig] = useState<Record<string, unknown> | null>(null);
  const [isServerReady, setIsServerReady] = useState(false);
  const lastSyncedConfigRef = useRef<string | null>(null);
  const syncTimerRef = useRef<number | null>(null);
  const overridesRef = useRef(overrides);

  useEffect(() => {
    overridesRef.current = overrides;
  }, [overrides]);

  useEffect(() => {
    setStyleState((prev) => resolveReplayStyle({ stored: prev, overrides }));
  }, [overrides]);

  useEffect(() => {
    let isCancelled = false;
    void (async () => {
      try {
        const payload = await fetchReplayStylePayload();
        if (isCancelled || !payload) {
          return;
        }
        lastSyncedConfigRef.current = serializePayload(payload.style, payload.extra);
        setServerExtraConfig(payload.extra);
        setStyleState((prev) => resolveReplayStyle({ stored: { ...prev, ...payload.style }, overrides: overridesRef.current }));
      } catch (err) {
        logger.warn('Failed to load replay style from API', { component: 'ReplayStyle', executionId }, err);
      } finally {
        if (!isCancelled) {
          setIsServerReady(true);
        }
      }
    })();
    return () => {
      isCancelled = true;
    };
  }, [executionId]);

  useEffect(() => {
    writeReplayStyleToStorage(style);
  }, [style]);

  useEffect(() => {
    if (!isServerReady) {
      return;
    }
    const serialized = serializePayload(style, extraConfig);
    if (lastSyncedConfigRef.current === serialized) {
      return;
    }

    if (syncTimerRef.current !== null) {
      window.clearTimeout(syncTimerRef.current);
    }
    syncTimerRef.current = window.setTimeout(() => {
      void (async () => {
        try {
          await persistReplayStyleConfig(style, extraConfig);
          lastSyncedConfigRef.current = serialized;
        } catch (err) {
          logger.warn('Failed to persist replay style', { component: 'ReplayStyle', executionId }, err);
        }
      })();
    }, 500);
  }, [executionId, extraConfig, isServerReady, style]);

  useEffect(() => {
    return () => {
      if (syncTimerRef.current !== null) {
        window.clearTimeout(syncTimerRef.current);
      }
    };
  }, []);

  const setStyle = useCallback(
    (updater: ReplayStyleOverrides | ((prev: ReplayStyleConfig) => ReplayStyleOverrides)) => {
      setStyleState((prev) => {
        const patch = typeof updater === 'function' ? updater(prev) : updater;
        return normalizeReplayStyle({ ...prev, ...patch }, prev);
      });
    },
    [],
  );

  const setChromeTheme = useCallback(
    (value: ReplayStyleConfig['chromeTheme']) => setStyle({ chromeTheme: value }),
    [setStyle],
  );
  const setBackground = useCallback(
    (value: ReplayBackgroundSource) => setStyle({ background: value }),
    [setStyle],
  );
  const setBackgroundTheme = useCallback(
    (value: ReplayBackgroundTheme) => setBackground({ type: 'theme', id: value }),
    [setBackground],
  );
  const setCursorTheme = useCallback(
    (value: ReplayStyleConfig['cursorTheme']) => setStyle({ cursorTheme: value }),
    [setStyle],
  );
  const setCursorInitialPosition = useCallback(
    (value: ReplayStyleConfig['cursorInitialPosition']) => setStyle({ cursorInitialPosition: value }),
    [setStyle],
  );
  const setCursorClickAnimation = useCallback(
    (value: ReplayStyleConfig['cursorClickAnimation']) => setStyle({ cursorClickAnimation: value }),
    [setStyle],
  );
  const setCursorScale = useCallback(
    (value: number) => setStyle({ cursorScale: value }),
    [setStyle],
  );
  const setBrowserScale = useCallback(
    (value: number) => setStyle({ browserScale: value }),
    [setStyle],
  );

  return {
    style,
    setStyle,
    setChromeTheme,
    setBackground,
    setBackgroundTheme,
    setCursorTheme,
    setCursorInitialPosition,
    setCursorClickAnimation,
    setCursorScale,
    setBrowserScale,
    serverExtraConfig,
    isServerReady,
  };
}
