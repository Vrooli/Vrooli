import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { logger } from '@utils/logger';
import type {
  ReplayBackgroundTheme,
  ReplayChromeTheme,
  ReplayCursorClickAnimation,
  ReplayCursorInitialPosition,
  ReplayCursorTheme,
} from '@/domains/replay-style/model';
import {
  REPLAY_BACKGROUND_OPTIONS,
  REPLAY_CHROME_OPTIONS,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
  REPLAY_CURSOR_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
  type BackgroundOption,
  type ClickAnimationOption,
  type CursorOption,
  type CursorPositionOption,
} from '@/domains/replay-style/catalog';
import { clampReplayBrowserScale, clampReplayCursorScale } from '@/domains/replay-style/model';
import { useReplayStyle } from '@/domains/replay-style/useReplayStyle';
import type { ExportRenderSource } from './exportConfig';

export interface ReplayCustomizationController {
  replayChromeTheme: ReplayChromeTheme;
  replayBackgroundTheme: ReplayBackgroundTheme;
  replayCursorTheme: ReplayCursorTheme;
  replayCursorInitialPosition: ReplayCursorInitialPosition;
  replayCursorClickAnimation: ReplayCursorClickAnimation;
  replayCursorScale: number;
  replayBrowserScale: number;
  replayRenderSource: ExportRenderSource;
  setReplayChromeTheme: (value: ReplayChromeTheme) => void;
  setReplayBackgroundTheme: (value: ReplayBackgroundTheme) => void;
  setReplayCursorTheme: (value: ReplayCursorTheme) => void;
  setReplayCursorInitialPosition: (value: ReplayCursorInitialPosition) => void;
  setReplayCursorClickAnimation: (value: ReplayCursorClickAnimation) => void;
  setReplayCursorScale: (value: number) => void;
  setReplayBrowserScale: (value: number) => void;
  setReplayRenderSource: (value: ExportRenderSource) => void;
  selectedChromeOption: (typeof REPLAY_CHROME_OPTIONS)[number];
  selectedBackgroundOption: BackgroundOption;
  selectedCursorOption: CursorOption;
  selectedCursorPositionOption: CursorPositionOption;
  selectedCursorClickAnimationOption: ClickAnimationOption;
  backgroundOptionsByGroup: Record<BackgroundOption['kind'], BackgroundOption[]>;
  cursorOptionsByGroup: Record<CursorOption['group'], CursorOption[]>;
  isCustomizationCollapsed: boolean;
  setIsCustomizationCollapsed: (value: boolean) => void;
  isBackgroundMenuOpen: boolean;
  setIsBackgroundMenuOpen: (value: boolean) => void;
  isCursorMenuOpen: boolean;
  setIsCursorMenuOpen: (value: boolean) => void;
  isCursorPositionMenuOpen: boolean;
  setIsCursorPositionMenuOpen: (value: boolean) => void;
  isCursorClickAnimationMenuOpen: boolean;
  setIsCursorClickAnimationMenuOpen: (value: boolean) => void;
  backgroundSelectorRef: React.MutableRefObject<HTMLDivElement | null>;
  cursorSelectorRef: React.MutableRefObject<HTMLDivElement | null>;
  cursorPositionSelectorRef: React.MutableRefObject<HTMLDivElement | null>;
  cursorClickAnimationSelectorRef: React.MutableRefObject<HTMLDivElement | null>;
  handleBackgroundSelect: (option: BackgroundOption) => void;
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
    setBackgroundTheme,
    setCursorTheme,
    setCursorInitialPosition,
    setCursorClickAnimation,
    setCursorScale,
    setBrowserScale,
    serverExtraConfig,
    isServerReady,
  } = useReplayStyle({ executionId, extraConfig });

  const [isCustomizationCollapsed, setIsCustomizationCollapsed] = useState(true);
  const [isBackgroundMenuOpen, setIsBackgroundMenuOpen] = useState(false);
  const [isCursorMenuOpen, setIsCursorMenuOpen] = useState(false);
  const [isCursorPositionMenuOpen, setIsCursorPositionMenuOpen] = useState(false);
  const [isCursorClickAnimationMenuOpen, setIsCursorClickAnimationMenuOpen] = useState(false);

  const backgroundSelectorRef = useRef<HTMLDivElement | null>(null);
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
    () => setupDismissListeners(isBackgroundMenuOpen, backgroundSelectorRef, () => setIsBackgroundMenuOpen(false)),
    [isBackgroundMenuOpen],
  );
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
    setIsBackgroundMenuOpen(false);
    setIsCursorMenuOpen(false);
    setIsCursorPositionMenuOpen(false);
    setIsCursorClickAnimationMenuOpen(false);
  }, [isCustomizationCollapsed]);

  const backgroundOptionsByGroup = useMemo(() => {
    const base: Record<BackgroundOption['kind'], BackgroundOption[]> = {
      abstract: [],
      solid: [],
      minimal: [],
      geometric: [],
    };
    for (const option of REPLAY_BACKGROUND_OPTIONS) {
      base[option.kind].push(option);
    }
    return base;
  }, []);

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

  const selectedBackgroundOption = useMemo<BackgroundOption>(
    () => REPLAY_BACKGROUND_OPTIONS.find((option) => option.id === style.backgroundTheme) || REPLAY_BACKGROUND_OPTIONS[0],
    [style.backgroundTheme],
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

  const handleBackgroundSelect = useCallback((option: BackgroundOption) => {
    setBackgroundTheme(option.id);
    setIsBackgroundMenuOpen(false);
  }, [setBackgroundTheme]);

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
    replayBackgroundTheme: style.backgroundTheme,
    replayCursorTheme: style.cursorTheme,
    replayCursorInitialPosition: style.cursorInitialPosition,
    replayCursorClickAnimation: style.cursorClickAnimation,
    replayCursorScale: style.cursorScale,
    replayBrowserScale: style.browserScale,
    replayRenderSource,
    setReplayChromeTheme: setChromeTheme,
    setReplayBackgroundTheme: setBackgroundTheme,
    setReplayCursorTheme: setCursorTheme,
    setReplayCursorInitialPosition: setCursorInitialPosition,
    setReplayCursorClickAnimation: setCursorClickAnimation,
    setReplayCursorScale: setCursorScale,
    setReplayBrowserScale: setBrowserScale,
    setReplayRenderSource,
    selectedChromeOption,
    selectedBackgroundOption,
    selectedCursorOption,
    selectedCursorPositionOption,
    selectedCursorClickAnimationOption,
    backgroundOptionsByGroup,
    cursorOptionsByGroup,
    isCustomizationCollapsed,
    setIsCustomizationCollapsed,
    isBackgroundMenuOpen,
    setIsBackgroundMenuOpen,
    isCursorMenuOpen,
    setIsCursorMenuOpen,
    isCursorPositionMenuOpen,
    setIsCursorPositionMenuOpen,
    isCursorClickAnimationMenuOpen,
    setIsCursorClickAnimationMenuOpen,
    backgroundSelectorRef,
    cursorSelectorRef,
    cursorPositionSelectorRef,
    cursorClickAnimationSelectorRef,
    handleBackgroundSelect,
    handleCursorThemeSelect,
    handleCursorPositionSelect,
    handleCursorClickAnimationSelect,
    handleCursorScaleChange,
    handleBrowserScaleChange,
  };
}
