import { useState, useCallback, useEffect, useLayoutEffect, useMemo, useRef } from "react";
import { Settings, Sparkles, Plus, ChevronLeft, ChevronRight, Maximize2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import type { DragEndInfo } from "../hooks/useDraggablePosition";
import { useLongPress } from "../hooks/useLongPress";
import { useFloatingPosition } from "../hooks/useFloatingPosition";
import { strings } from "../consts/strings";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { Button } from "./ui/button";
import VoiceMicButton from "./VoiceMicButton";
import type { StartRecordingOpts, VoiceActivitySnapshot } from "../audio-integration";
import { readSafeAreaInsets } from "../lib/safeArea";

/** Minimum fling speed (px/s) to trigger a dock */
const FLING_VELOCITY_THRESHOLD = 400;
/** How close to an edge (px) a release must be to dock without fling */
const EDGE_PROXIMITY_THRESHOLD = 40;
/** Width of the visible tab when docked (px) */
const DOCK_TAB_WIDTH = 24;
/** Duration of dock/undock animation (ms) — keep in sync with CSS transition */
const DOCK_ANIMATION_MS = 300;

type DockedEdge = "left" | "right" | null;

const DOCK_STORAGE_KEY = "wc-toolbar-dock";

function loadDockedEdge(): DockedEdge {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(DOCK_STORAGE_KEY);
    if (raw === "left" || raw === "right") return raw;
  } catch { /* ignore */ }
  return null;
}

function saveDockedEdge(edge: DockedEdge) {
  if (typeof window === "undefined") return;
  try {
    if (edge) {
      localStorage.setItem(DOCK_STORAGE_KEY, edge);
    } else {
      localStorage.removeItem(DOCK_STORAGE_KEY);
    }
  } catch { /* ignore */ }
}

interface FloatingToolbarProps {
  onOpenSettings: () => void;
  onOpenAi: () => void;
  onNewTerminal: () => void;
  onOpenLauncher: () => void;
  /** Open the full-screen composer for the active session (desktop entry). */
  onExpandComposer?: () => void;
  isCreating: boolean;
  /** When true the toolbar is not rendered at all (e.g. mobile tab mode). */
  hidden?: boolean;
  // Voice input (optional — hidden when not provided)
  voiceSupported?: boolean;
  voicePreparing?: boolean;
  voiceRecording?: boolean;
  voicePersistentMode?: boolean;
  voiceListening?: boolean;
  /** True when passive wake-word listening currently holds the mic. */
  voicePassive?: boolean;
  voiceTranscribing?: boolean;
  voiceError?: string | null;
  /** 0–1 audio level for live mic visualization */
  voiceLevel?: number;
  voiceActivity?: VoiceActivitySnapshot;
  voiceBackend?: string;
  onVoiceStart?: (opts?: StartRecordingOpts) => void;
  onVoicePrepare?: () => void;
  onVoiceStop?: () => void;
  /** Exit passive wake-word listening (tapping the passive mic button). */
  onVoiceExitPassive?: () => void;
}

export default function FloatingToolbar({
  onOpenSettings,
  onOpenAi,
  onNewTerminal,
  onOpenLauncher,
  onExpandComposer,
  isCreating,
  hidden,
  voiceSupported,
  voicePreparing,
  voiceRecording,
  voicePersistentMode,
  voiceListening,
  voicePassive,
  voiceTranscribing,
  voiceError,
  voiceLevel = 0,
  voiceActivity,
  voiceBackend,
  onVoiceStart,
  onVoicePrepare,
  onVoiceStop,
  onVoiceExitPassive,
}: FloatingToolbarProps) {
  const { t } = useTranslation();
  const plusButtonBehavior = useWorkspaceStore((s) => s.plusButtonBehavior);
  const [docked, setDocked] = useState<DockedEdge>(loadDockedEdge);
  const [animating, setAnimating] = useState(false);
  /** Measured full width of toolbar, for computing dock offset */
  // Use a conservative pre-measure width so the first paint is safe even when
  // a persisted coordinate was recorded on a wider viewport. The measured
  // width replaces this estimate immediately after layout.
  const [toolbarWidth, setToolbarWidth] = useState(220);
  /** Viewport width for right-dock positioning */
  const [vpWidth, setVpWidth] = useState(() =>
    typeof window !== "undefined" ? window.innerWidth : 1000,
  );

  // Track viewport width for right-dock transform
  useLayoutEffect(() => {
    const onResize = () => setVpWidth(window.innerWidth);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  const dockedRef = useRef(docked);
  dockedRef.current = docked;

  const handleDragEnd = useCallback((info: DragEndInfo) => {
    const { position, velocity, elementSize } = info;
    const vw = typeof window !== "undefined" ? window.innerWidth : 1000;

    const flungLeft = velocity.vx < -FLING_VELOCITY_THRESHOLD;
    const flungRight = velocity.vx > FLING_VELOCITY_THRESHOLD;
    const nearLeft = position.x < EDGE_PROXIMITY_THRESHOLD;
    const nearRight = position.x + elementSize.width > vw - EDGE_PROXIMITY_THRESHOLD;

    let edge: DockedEdge = null;
    if (flungLeft || nearLeft) edge = "left";
    else if (flungRight || nearRight) edge = "right";

    if (edge) {
      setAnimating(true);
      setDocked(edge);
      saveDockedEdge(edge);
      setTimeout(() => setAnimating(false), DOCK_ANIMATION_MS);
    }
  }, []);

  const { clampPosition } = useFloatingPosition();
  const { elementRef, floatingStyle, pointerHandlers, handleClickCapture, position, moveTo } =
    useDraggablePosition({
      isActive: true,
      storageKey: "wc-toolbar-pos",
      defaultPosition: () => {
        if (typeof window === "undefined") return { x: 100, y: 12 };
        return { x: window.innerWidth - 180, y: 12 };
      },
      onDragStart: () => {
        if (dockedRef.current) {
          setDocked(null);
          saveDockedEdge(null);
        }
      },
      onDragEnd: handleDragEnd,
    });

  // Measure toolbar width so we know how far to slide off-screen.
  // We measure continuously (not just when undocked) because the element
  // stays full-width even when docked — buttons are hidden by being off-screen,
  // not by collapsing.
  useEffect(() => {
    const el = elementRef.current;
    if (!el) return;
    const syncMeasuredBounds = () => {
      const rect = el.getBoundingClientRect();
      if (rect.width <= 0) return;
      setToolbarWidth(rect.width);
      // A stored position can outlive a viewport or responsive width change.
      // Re-clamp from the measured box so the toolbar never extends beyond the
      // capture viewport before the user interacts with it.
      moveTo(clampPosition(position.x, position.y, {
        width: rect.width,
        height: rect.height,
      }));
    };
    syncMeasuredBounds();
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(syncMeasuredBounds);
    observer?.observe(el);
    return () => observer?.disconnect();
  }, [clampPosition, docked, elementRef, moveTo, position.x, position.y]);

  const undock = useCallback(() => {
    setAnimating(true);
    setDocked(null);
    saveDockedEdge(null);
    setTimeout(() => setAnimating(false), DOCK_ANIMATION_MS);
  }, []);

  // Compute style: when docked, override transform to slide off-screen,
  // leaving only DOCK_TAB_WIDTH pixels visible at the edge.
  //
  // Left dock:  toolbar slides left, right edge is visible → tab renders at right end
  // Right dock: toolbar slides right, left edge is visible → tab renders at left end
  const computedStyle = useMemo(() => {
    if (!docked) {
      // Clamp the rendered transform as well as the stored state. Layout
      // effects repair the persisted coordinate, but the first paint must not
      // briefly extend past the capture viewport while that repair runs.
      const safePosition = clampPosition(position.x, position.y, {
        width: toolbarWidth,
        height: 54,
      });
      return {
        ...floatingStyle,
        transform: `translate3d(${Math.round(safePosition.x)}px, ${Math.round(safePosition.y)}px, 0)`,
      };
    }
    const y = position.y;
    if (docked === "left") {
      // Slide left so only the rightmost DOCK_TAB_WIDTH pixels are visible
      const safe = readSafeAreaInsets();
      return { transform: `translate3d(${safe.left - (toolbarWidth - DOCK_TAB_WIDTH)}px, ${Math.max(y, safe.top + 12)}px, 0)` };
    }
    // Slide right so only the leftmost DOCK_TAB_WIDTH pixels are visible
    const safe = readSafeAreaInsets();
    return { transform: `translate3d(${vpWidth - safe.right - DOCK_TAB_WIDTH}px, ${Math.max(y, safe.top + 12)}px, 0)` };
  }, [clampPosition, docked, floatingStyle, position.x, position.y, toolbarWidth, vpWidth]);

  const plusHandlers = useLongPress({
    onPress: plusButtonBehavior === "launcher" ? onOpenLauncher : onNewTerminal,
    onLongPress: plusButtonBehavior === "launcher" ? onNewTerminal : onOpenLauncher,
  });

  // The dock tab indicator — renders on the visible edge
  const dockTab = docked ? (
    <div
      data-testid="dock-tab"
      className="flex items-center justify-center shrink-0 px-1 py-1"
      title={t(strings.floatingToolbar.tapToRestore)}
    >
      {docked === "left" ? (
        <ChevronRight className="h-4 w-4 text-wc-text-muted" />
      ) : (
        <ChevronLeft className="h-4 w-4 text-wc-text-muted" />
      )}
    </div>
  ) : null;

  // Toolbar buttons — stay full-width always. When docked, they're hidden
  // by being pushed off-screen (not collapsed), so the element width stays
  // consistent with the dock offset calculation.
  const buttons = (
    <div
      className="flex items-center gap-1 px-2 py-1"
      aria-hidden={!!docked}
    >
      <Button
        data-testid="toolbar-settings"
        variant="ghost"
        size="icon"
        className="h-11 w-11 md:h-8 md:w-8"
        onClick={onOpenSettings}
        title={t(strings.floatingToolbar.settingsTitle)}
        tabIndex={docked ? -1 : undefined}
      >
        <Settings className="h-4 w-4" />
      </Button>
      {/* AI and mic buttons stay out of the narrow toolbar breakpoint because
       * the bottom touch toolbar already provides both there. Wider touch
       * viewports may show both surfaces so keyboardless devices retain the
       * terminal key controls without changing the desktop toolbar contract. */}
      <Button
        data-testid="toolbar-ai"
        variant="ghost"
        size="icon"
        className="h-8 w-8 hidden md:inline-flex"
        onClick={onOpenAi}
        title={t(strings.floatingToolbar.aiCommandTitle)}
        tabIndex={docked ? -1 : undefined}
      >
        <Sparkles className="h-4 w-4" />
      </Button>
      {onExpandComposer && (
        <Button
          data-testid="toolbar-expand-composer"
          variant="ghost"
          size="icon"
          className="h-8 w-8 hidden md:inline-flex"
          onClick={onExpandComposer}
          title={t(strings.floatingToolbar.expandComposerTitle)}
          tabIndex={docked ? -1 : undefined}
        >
          <Maximize2 className="h-4 w-4" />
        </Button>
      )}
      {voiceSupported && onVoiceStart && onVoiceStop && (
        <VoiceMicButton
          testId="voice-mic-btn"
          supported={voiceSupported}
          isPreparing={voicePreparing ?? false}
          isRecording={voiceRecording ?? false}
          persistentMode={voicePersistentMode ?? false}
          isListening={voiceListening ?? false}
          isPassive={voicePassive ?? false}
          isTranscribing={voiceTranscribing ?? false}
          size="xs"
          error={voiceError ?? null}
          audioLevel={voiceLevel}
          voiceActivity={voiceActivity}
          backend={voiceBackend}
          onPrepare={onVoicePrepare}
          onStart={onVoiceStart}
          onStop={onVoiceStop}
          onExitPassive={onVoiceExitPassive}
          className="hidden md:flex"
          buttonClassName="h-8 w-8"
        />
      )}
      <Button
        data-testid="toolbar-new"
        variant="ghost"
        size="icon"
        className="h-11 w-11 md:h-8 md:w-8"
        disabled={isCreating}
        title={plusButtonBehavior === "launcher" ? t(strings.floatingToolbar.launcherFirstTitle) : t(strings.floatingToolbar.terminalFirstTitle)}
        tabIndex={docked ? -1 : undefined}
        onPointerDown={plusHandlers.onPointerDown}
        onPointerUp={plusHandlers.onPointerUp}
        onPointerCancel={plusHandlers.onPointerCancel}
        onContextMenu={plusHandlers.onContextMenu}
      >
        <Plus className="h-4 w-4" />
      </Button>
    </div>
  );

  if (hidden) return null;

  return (
    <div
      ref={(node) => { elementRef.current = node; }}
      data-testid="floating-toolbar"
      className={`fixed left-0 top-0 z-wc-toolbar flex items-center rounded-full border border-wc-default bg-wc-surface-raised/95 backdrop-blur-md shadow-lg select-none touch-none ${
        docked ? "cursor-pointer" : "cursor-grab active:cursor-grabbing"
      } ${animating ? "wc-dock-transition" : ""}`}
      style={computedStyle}
      onPointerDown={docked ? undefined : pointerHandlers.onPointerDown}
      onPointerMove={docked ? undefined : pointerHandlers.onPointerMove}
      onPointerUp={docked ? undefined : pointerHandlers.onPointerUp}
      onPointerCancel={docked ? undefined : pointerHandlers.onPointerCancel}
      onClickCapture={docked ? undefined : handleClickCapture}
      onClick={docked ? undock : undefined}
    >
      {/*
        Layout order depends on dock side:
        - Right dock → [tab] [buttons]  (tab is on the visible left edge)
        - Left dock  → [buttons] [tab]  (tab is on the visible right edge)
        - Undocked   → [buttons] only   (no tab)
      */}
      {docked === "right" && dockTab}
      {buttons}
      {docked === "left" && dockTab}
    </div>
  );
}
