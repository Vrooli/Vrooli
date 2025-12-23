import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { logger } from '@utils/logger';
import type {
  ReplayBackgroundSource,
  ReplayBackgroundTheme,
  ReplayChromeTheme,
  ReplayCursorClickAnimation,
  ReplayCursorInitialPosition,
  ReplayCursorTheme,
  ReplayPresentationMode,
} from '@/domains/replay-style';
import {
  getReplayBackgroundThemeId,
  useReplayStyle,
} from '@/domains/replay-style';
import type { ExportRenderSource } from './exportConfig';

export interface ReplayCustomizationController {
  replayChromeTheme: ReplayChromeTheme;
  replayPresentationMode: ReplayPresentationMode;
  replayBackgroundTheme: ReplayBackgroundTheme;
  replayBackground: ReplayBackgroundSource;
  replayCursorTheme: ReplayCursorTheme;
  replayCursorInitialPosition: ReplayCursorInitialPosition;
  replayCursorClickAnimation: ReplayCursorClickAnimation;
  replayCursorScale: number;
  replayBrowserScale: number;
  replayRenderSource: ExportRenderSource;
  setReplayChromeTheme: (value: ReplayChromeTheme) => void;
  setReplayPresentationMode: (value: ReplayPresentationMode) => void;
  setReplayBackground: (value: ReplayBackgroundSource) => void;
  setReplayCursorTheme: (value: ReplayCursorTheme) => void;
  setReplayCursorInitialPosition: (value: ReplayCursorInitialPosition) => void;
  setReplayCursorClickAnimation: (value: ReplayCursorClickAnimation) => void;
  setReplayCursorScale: (value: number) => void;
  setReplayBrowserScale: (value: number) => void;
  setReplayRenderSource: (value: ExportRenderSource) => void;
  isCustomizationCollapsed: boolean;
  setIsCustomizationCollapsed: (value: boolean) => void;
}

const persistToLocalStorage = (key: string, value: string, context?: Record<string, unknown>) => {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    window.localStorage.setItem(key, value);
  } catch (err) {
    logger.warn('Failed to persist replay customization', { component: 'ReplayCustomization', key, ...context }, err);
  }
};

const isReplayRenderSource = (value: unknown): value is ExportRenderSource => {
  return value === 'auto' || value === 'recorded_video' || value === 'replay_frames';
};

export function useReplayCustomization(params: { executionId: string }): ReplayCustomizationController {
  const { executionId } = params;
  const [replayRenderSource, setReplayRenderSourceState] = useState<ExportRenderSource>(() => {
    if (typeof window === 'undefined') {
      return 'auto';
    }
    const stored = window.localStorage.getItem('browserAutomation.replayRenderSource');
    return isReplayRenderSource(stored) ? stored : 'auto';
  });
  const [hasUserEditedRenderSource, setHasUserEditedRenderSource] = useState(false);
  const hasAppliedServerRenderSource = useRef(false);

  const setReplayRenderSource = useCallback((value: ExportRenderSource) => {
    setHasUserEditedRenderSource(true);
    setReplayRenderSourceState(value);
  }, []);

  const extraConfig = useMemo(() => ({ renderSource: replayRenderSource }), [replayRenderSource]);
  const {
    style,
    setChromeTheme,
    setBackground,
    setPresentationMode,
    setCursorTheme,
    setCursorInitialPosition,
    setCursorClickAnimation,
    setCursorScale,
    setBrowserScale,
    serverExtraConfig,
    isServerReady,
  } = useReplayStyle({ executionId, extraConfig });

  const [isCustomizationCollapsed, setIsCustomizationCollapsed] = useState(true);

  useEffect(() => {
    persistToLocalStorage('browserAutomation.replayRenderSource', replayRenderSource, { executionId });
  }, [replayRenderSource, executionId]);

  useEffect(() => {
    if (!isServerReady || hasAppliedServerRenderSource.current) {
      return;
    }
    const remoteSource = serverExtraConfig?.renderSource;
    if (!hasUserEditedRenderSource && isReplayRenderSource(remoteSource)) {
      setReplayRenderSourceState(remoteSource);
    }
    hasAppliedServerRenderSource.current = true;
  }, [hasUserEditedRenderSource, isServerReady, serverExtraConfig]);

  return {
    replayChromeTheme: style.chromeTheme,
    replayPresentationMode: style.presentationMode,
    replayBackgroundTheme: getReplayBackgroundThemeId(style.background),
    replayBackground: style.background,
    replayCursorTheme: style.cursorTheme,
    replayCursorInitialPosition: style.cursorInitialPosition,
    replayCursorClickAnimation: style.cursorClickAnimation,
    replayCursorScale: style.cursorScale,
    replayBrowserScale: style.browserScale,
    replayRenderSource,
    setReplayChromeTheme: setChromeTheme,
    setReplayPresentationMode: setPresentationMode,
    setReplayBackground: setBackground,
    setReplayCursorTheme: setCursorTheme,
    setReplayCursorInitialPosition: setCursorInitialPosition,
    setReplayCursorClickAnimation: setCursorClickAnimation,
    setReplayCursorScale: setCursorScale,
    setReplayBrowserScale: setBrowserScale,
    setReplayRenderSource,
    isCustomizationCollapsed,
    setIsCustomizationCollapsed,
  };
}
