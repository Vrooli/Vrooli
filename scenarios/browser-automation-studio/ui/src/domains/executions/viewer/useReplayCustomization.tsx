import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { logger } from "@utils/logger";
import type {
  ReplayBackgroundTheme,
  ReplayChromeTheme,
  ReplayCursorClickAnimation,
  ReplayCursorInitialPosition,
  ReplayCursorTheme,
} from "../ReplayPlayer";
import {
  CURSOR_SCALE_MAX,
  CURSOR_SCALE_MIN,
  REPLAY_BACKGROUND_OPTIONS,
  REPLAY_CHROME_OPTIONS,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
  REPLAY_CURSOR_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
  isReplayBackgroundTheme,
  isReplayChromeTheme,
  isReplayCursorClickAnimation,
  isReplayCursorInitialPosition,
  isReplayCursorTheme,
  type BackgroundOption,
  type ClickAnimationOption,
  type CursorOption,
  type CursorPositionOption,
} from "../replay/replayThemeOptions";

export interface ReplayCustomizationController {
  replayChromeTheme: ReplayChromeTheme;
  replayBackgroundTheme: ReplayBackgroundTheme;
  replayCursorTheme: ReplayCursorTheme;
  replayCursorInitialPosition: ReplayCursorInitialPosition;
  replayCursorClickAnimation: ReplayCursorClickAnimation;
  replayCursorScale: number;
  setReplayChromeTheme: (value: ReplayChromeTheme) => void;
  setReplayBackgroundTheme: (value: ReplayBackgroundTheme) => void;
  setReplayCursorTheme: (value: ReplayCursorTheme) => void;
  setReplayCursorInitialPosition: (value: ReplayCursorInitialPosition) => void;
  setReplayCursorClickAnimation: (value: ReplayCursorClickAnimation) => void;
  setReplayCursorScale: (value: number) => void;
  selectedChromeOption: (typeof REPLAY_CHROME_OPTIONS)[number];
  selectedBackgroundOption: BackgroundOption;
  selectedCursorOption: CursorOption;
  selectedCursorPositionOption: CursorPositionOption;
  selectedCursorClickAnimationOption: ClickAnimationOption;
  backgroundOptionsByGroup: Record<BackgroundOption["kind"], BackgroundOption[]>;
  cursorOptionsByGroup: Record<CursorOption["group"], CursorOption[]>;
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
}

const persistToLocalStorage = (key: string, value: string, context?: Record<string, unknown>) => {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(key, value);
  } catch (err) {
    logger.warn("Failed to persist replay customization", { component: "ReplayCustomization", key, ...context }, err);
  }
};

const setupDismissListeners = (
  isOpen: boolean,
  ref: React.MutableRefObject<HTMLElement | null>,
  onClose: () => void,
) => {
  if (!isOpen || typeof document === "undefined") {
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
    if (event.key === "Escape") {
      onClose();
    }
  };
  document.addEventListener("mousedown", handlePointerDown);
  document.addEventListener("keydown", handleKeyDown);
  return () => {
    document.removeEventListener("mousedown", handlePointerDown);
    document.removeEventListener("keydown", handleKeyDown);
  };
};

export function useReplayCustomization(params: { executionId: string }): ReplayCustomizationController {
  const { executionId } = params;
  const [replayChromeTheme, setReplayChromeTheme] = useState<ReplayChromeTheme>(() => {
    if (typeof window === "undefined") {
      return "aurora";
    }
    const stored = window.localStorage.getItem("browserAutomation.replayChromeTheme");
    return isReplayChromeTheme(stored) ? stored : "aurora";
  });
  const [replayBackgroundTheme, setReplayBackgroundTheme] = useState<ReplayBackgroundTheme>(() => {
    if (typeof window === "undefined") {
      return "aurora";
    }
    const stored = window.localStorage.getItem("browserAutomation.replayBackgroundTheme");
    return isReplayBackgroundTheme(stored) ? stored : "aurora";
  });
  const [replayCursorTheme, setReplayCursorTheme] = useState<ReplayCursorTheme>(() => {
    if (typeof window === "undefined") {
      return "white";
    }
    const stored = window.localStorage.getItem("browserAutomation.replayCursorTheme");
    return isReplayCursorTheme(stored) ? stored : "white";
  });
  const [replayCursorInitialPosition, setReplayCursorInitialPosition] =
    useState<ReplayCursorInitialPosition>(() => {
      if (typeof window === "undefined") {
        return "center";
      }
      const stored = window.localStorage.getItem("browserAutomation.replayCursorInitialPosition");
      return isReplayCursorInitialPosition(stored) ? stored : "center";
    });
  const [replayCursorClickAnimation, setReplayCursorClickAnimation] =
    useState<ReplayCursorClickAnimation>(() => {
      if (typeof window === "undefined") {
        return "none";
      }
      const stored = window.localStorage.getItem("browserAutomation.replayCursorClickAnimation");
      return isReplayCursorClickAnimation(stored) ? stored : "none";
    });
  const [replayCursorScale, setReplayCursorScale] = useState<number>(() => {
    if (typeof window === "undefined") {
      return 1;
    }
    const stored = window.localStorage.getItem("browserAutomation.replayCursorScale");
    if (!stored) {
      return 1;
    }
    const parsed = Number.parseFloat(stored);
    if (Number.isFinite(parsed)) {
      return Math.min(CURSOR_SCALE_MAX, Math.max(CURSOR_SCALE_MIN, parsed));
    }
    return 1;
  });

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
    persistToLocalStorage("browserAutomation.replayChromeTheme", replayChromeTheme, { executionId });
  }, [replayChromeTheme, executionId]);

  useEffect(() => {
    persistToLocalStorage("browserAutomation.replayBackgroundTheme", replayBackgroundTheme, { executionId });
  }, [replayBackgroundTheme, executionId]);

  useEffect(() => {
    persistToLocalStorage("browserAutomation.replayCursorTheme", replayCursorTheme, { executionId });
  }, [replayCursorTheme, executionId]);

  useEffect(() => {
    persistToLocalStorage("browserAutomation.replayCursorInitialPosition", replayCursorInitialPosition, { executionId });
  }, [replayCursorInitialPosition, executionId]);

  useEffect(() => {
    persistToLocalStorage("browserAutomation.replayCursorClickAnimation", replayCursorClickAnimation, { executionId });
  }, [replayCursorClickAnimation, executionId]);

  useEffect(() => {
    persistToLocalStorage("browserAutomation.replayCursorScale", replayCursorScale.toFixed(2), { executionId });
  }, [replayCursorScale, executionId]);

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
    const base: Record<BackgroundOption["kind"], BackgroundOption[]> = {
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
    const base: Record<CursorOption["group"], CursorOption[]> = {
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
    () => REPLAY_CHROME_OPTIONS.find((option) => option.id === replayChromeTheme) || REPLAY_CHROME_OPTIONS[0],
    [replayChromeTheme],
  );

  const selectedBackgroundOption = useMemo<BackgroundOption>(
    () => REPLAY_BACKGROUND_OPTIONS.find((option) => option.id === replayBackgroundTheme) || REPLAY_BACKGROUND_OPTIONS[0],
    [replayBackgroundTheme],
  );

  const selectedCursorOption = useMemo<CursorOption>(
    () => REPLAY_CURSOR_OPTIONS.find((option) => option.id === replayCursorTheme) || REPLAY_CURSOR_OPTIONS[0],
    [replayCursorTheme],
  );

  const selectedCursorPositionOption = useMemo(
    () =>
      REPLAY_CURSOR_POSITIONS.find((option) => option.id === replayCursorInitialPosition) ||
      REPLAY_CURSOR_POSITIONS[0],
    [replayCursorInitialPosition],
  );

  const selectedCursorClickAnimationOption = useMemo<ClickAnimationOption>(
    () =>
      REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.find((option) => option.id === replayCursorClickAnimation) ||
      REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS[0],
    [replayCursorClickAnimation],
  );

  const handleBackgroundSelect = useCallback((option: BackgroundOption) => {
    setReplayBackgroundTheme(option.id);
    setIsBackgroundMenuOpen(false);
  }, []);

  const handleCursorThemeSelect = useCallback((value: ReplayCursorTheme) => {
    setReplayCursorTheme(value);
    setIsCursorMenuOpen(false);
  }, []);

  const handleCursorPositionSelect = useCallback((value: ReplayCursorInitialPosition) => {
    setReplayCursorInitialPosition(value);
    setIsCursorPositionMenuOpen(false);
  }, []);

  const handleCursorClickAnimationSelect = useCallback((value: ReplayCursorClickAnimation) => {
    setReplayCursorClickAnimation(value);
    setIsCursorClickAnimationMenuOpen(false);
  }, []);

  const handleCursorScaleChange = useCallback((value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }
    const clamped = Math.min(CURSOR_SCALE_MAX, Math.max(CURSOR_SCALE_MIN, value));
    setReplayCursorScale(clamped);
  }, []);

  return {
    replayChromeTheme,
    replayBackgroundTheme,
    replayCursorTheme,
    replayCursorInitialPosition,
    replayCursorClickAnimation,
    replayCursorScale,
    setReplayChromeTheme,
    setReplayBackgroundTheme,
    setReplayCursorTheme,
    setReplayCursorInitialPosition,
    setReplayCursorClickAnimation,
    setReplayCursorScale,
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
  };
}
