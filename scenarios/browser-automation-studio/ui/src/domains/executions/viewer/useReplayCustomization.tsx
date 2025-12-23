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
  REPLAY_CHROME_OPTIONS,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
  REPLAY_CURSOR_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
  clampReplayBrowserScale,
  clampReplayCursorScale,
  getReplayBackgroundThemeId,
  useReplayStyle,
  type ClickAnimationOption,
  type CursorOption,
  type CursorPositionOption,
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
  selectedChromeOption: (typeof REPLAY_CHROME_OPTIONS)[number];
  selectedCursorOption: CursorOption;
  selectedCursorPositionOption: CursorPositionOption;
  selectedCursorClickAnimationOption: ClickAnimationOption;
  cursorOptionsByGroup: Record<CursorOption['group'], CursorOption[]>;
  isCustomizationCollapsed: boolean;
  setIsCustomizationCollapsed: (value: boolean) => void;
  isCursorMenuOpen: boolean;
  setIsCursorMenuOpen: (value: boolean) => void;
  isCursorPositionMenuOpen: boolean;
  setIsCursorPositionMenuOpen: (value: boolean) => void;
  isCursorClickAnimationMenuOpen: boolean;
  setIsCursorClickAnimationMenuOpen: (value: boolean) => void;
  cursorSelectorRef: React.MutableRefObject<HTMLDivElement | null>;
  cursorPositionSelectorRef: React.MutableRefObject<HTMLDivElement | null>;
  cursorClickAnimationSelectorRef: React.MutableRefObject<HTMLDivElement | null>;
  handleCursorThemeSelect: (value: ReplayCursorTheme) => void;
  handleCursorPositionSelect: (value: ReplayCursorInitialPosition) => void;
  handleCursorClickAnimationSelect: (value: ReplayCursorClickAnimation) => void;
  handleCursorScaleChange: (value: number) => void;
  handleBrowserScaleChange: (value: number) => void;
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

const setupDismissListeners = (
  isOpen: boolean,
  ref: React.MutableRefObject<HTMLElement | null>,
  onClose: () => void,
) => {
  if (!isOpen || typeof document === 'undefined') {
    return undefined;
  }
  const handlePointerDown = (event: MouseEvent) => {
    const target = event.target as Node | null;
    if (!target) return;
    if (ref.current && !ref.current.contains(target)) {
      onClose();
    }
  };
  const handleKeyDown = (event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      onClose();
    }
  };
  document.addEventListener('mousedown', handlePointerDown);
  document.addEventListener('keydown', handleKeyDown);
  return () => {
    document.removeEventListener('mousedown', handlePointerDown);
    document.removeEventListener('keydown', handleKeyDown);
  };
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
  const [isCursorMenuOpen, setIsCursorMenuOpen] = useState(false);
  const [isCursorPositionMenuOpen, setIsCursorPositionMenuOpen] = useState(false);
  const [isCursorClickAnimationMenuOpen, setIsCursorClickAnimationMenuOpen] = useState(false);
  const cursorSelectorRef = useRef<HTMLDivElement | null>(null);
  const cursorPositionSelectorRef = useRef<HTMLDivElement | null>(null);
  const cursorClickAnimationSelectorRef = useRef<HTMLDivElement | null>(null);

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

  useEffect(
    () => setupDismissListeners(isCursorMenuOpen, cursorSelectorRef, () => setIsCursorMenuOpen(false)),
    [isCursorMenuOpen],
  );
  useEffect(
    () =>
      setupDismissListeners(
        isCursorPositionMenuOpen,
        cursorPositionSelectorRef,
        () => setIsCursorPositionMenuOpen(false),
      ),
    [isCursorPositionMenuOpen],
  );
  useEffect(
    () =>
      setupDismissListeners(
        isCursorClickAnimationMenuOpen,
        cursorClickAnimationSelectorRef,
        () => setIsCursorClickAnimationMenuOpen(false),
      ),
    [isCursorClickAnimationMenuOpen],
  );

  useEffect(() => {
    if (!isCustomizationCollapsed) {
      return;
    }
    setIsCursorMenuOpen(false);
    setIsCursorPositionMenuOpen(false);
    setIsCursorClickAnimationMenuOpen(false);
  }, [isCustomizationCollapsed]);

  const cursorOptionsByGroup = useMemo(() => {
    const base: Record<CursorOption['group'], CursorOption[]> = {
      hidden: [],
      halo: [],
      arrow: [],
      hand: [],
    };
    for (const option of REPLAY_CURSOR_OPTIONS) {
      base[option.group].push(option);
    }
    return base;
  }, []);

  const selectedChromeOption = useMemo(
    () => REPLAY_CHROME_OPTIONS.find((option) => option.id === style.chromeTheme) || REPLAY_CHROME_OPTIONS[0],
    [style.chromeTheme],
  );

  const selectedCursorOption = useMemo<CursorOption>(
    () => REPLAY_CURSOR_OPTIONS.find((option) => option.id === style.cursorTheme) || REPLAY_CURSOR_OPTIONS[0],
    [style.cursorTheme],
  );

  const selectedCursorPositionOption = useMemo(
    () =>
      REPLAY_CURSOR_POSITIONS.find((option) => option.id === style.cursorInitialPosition) ||
      REPLAY_CURSOR_POSITIONS[0],
    [style.cursorInitialPosition],
  );

  const selectedCursorClickAnimationOption = useMemo<ClickAnimationOption>(
    () =>
      REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.find((option) => option.id === style.cursorClickAnimation) ||
      REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS[0],
    [style.cursorClickAnimation],
  );

  const handleCursorThemeSelect = useCallback((value: ReplayCursorTheme) => {
    setCursorTheme(value);
    setIsCursorMenuOpen(false);
  }, [setCursorTheme]);

  const handleCursorPositionSelect = useCallback((value: ReplayCursorInitialPosition) => {
    setCursorInitialPosition(value);
    setIsCursorPositionMenuOpen(false);
  }, [setCursorInitialPosition]);

  const handleCursorClickAnimationSelect = useCallback((value: ReplayCursorClickAnimation) => {
    setCursorClickAnimation(value);
    setIsCursorClickAnimationMenuOpen(false);
  }, [setCursorClickAnimation]);

  const handleCursorScaleChange = useCallback((value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }
    setCursorScale(clampReplayCursorScale(value));
  }, [setCursorScale]);

  const handleBrowserScaleChange = useCallback((value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }
    setBrowserScale(clampReplayBrowserScale(value));
  }, [setBrowserScale]);

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
    selectedChromeOption,
    selectedCursorOption,
    selectedCursorPositionOption,
    selectedCursorClickAnimationOption,
    cursorOptionsByGroup,
    isCustomizationCollapsed,
    setIsCustomizationCollapsed,
    isCursorMenuOpen,
    setIsCursorMenuOpen,
    isCursorPositionMenuOpen,
    setIsCursorPositionMenuOpen,
    isCursorClickAnimationMenuOpen,
    setIsCursorClickAnimationMenuOpen,
    cursorSelectorRef,
    cursorPositionSelectorRef,
    cursorClickAnimationSelectorRef,
    handleCursorThemeSelect,
    handleCursorPositionSelect,
    handleCursorClickAnimationSelect,
    handleCursorScaleChange,
    handleBrowserScaleChange,
  };
}
