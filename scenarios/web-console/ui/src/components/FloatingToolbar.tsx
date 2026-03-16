import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import { List, Settings, Sparkles, Plus, ChevronLeft, ChevronRight, Mic, Loader2, AlertCircle } from "lucide-react";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import type { DragEndInfo } from "../hooks/useDraggablePosition";
import { useLongPress } from "../hooks/useLongPress";
import { Button } from "./ui/button";
import type { StartRecordingOpts } from "../hooks/useVoiceInput";

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
  onOpenSessions: () => void;
  onOpenSettings: () => void;
  onOpenAi: () => void;
  onNewTerminal: () => void;
  onOpenLauncher: () => void;
  isCreating: boolean;
  // Voice input (optional — hidden when not provided)
  voiceSupported?: boolean;
  voiceRecording?: boolean;
  voiceTranscribing?: boolean;
  voiceError?: string | null;
  /** 0–1 audio level for live mic visualization */
  voiceLevel?: number;
  voicePartialTranscript?: string;
  voiceBackend?: string;
  onVoiceStart?: (opts?: StartRecordingOpts) => void;
  onVoiceStop?: () => void;
}

/** Hold duration (ms) that distinguishes tap-to-toggle from push-to-talk. */
const VOICE_LONG_PRESS_MS = 300;

export default function FloatingToolbar({
  onOpenSessions,
  onOpenSettings,
  onOpenAi,
  onNewTerminal,
  onOpenLauncher,
  isCreating,
  voiceSupported,
  voiceRecording,
  voiceTranscribing,
  voiceError,
  voiceLevel = 0,
  voicePartialTranscript,
  voiceBackend,
  onVoiceStart,
  onVoiceStop,
}: FloatingToolbarProps) {
  const [docked, setDocked] = useState<DockedEdge>(loadDockedEdge);
  const [animating, setAnimating] = useState(false);
  /** Measured full width of toolbar, for computing dock offset */
  const [toolbarWidth, setToolbarWidth] = useState(160);
  /** Viewport width for right-dock positioning */
  const [vpWidth, setVpWidth] = useState(() =>
    typeof window !== "undefined" ? window.innerWidth : 1000,
  );

  // Track viewport width for right-dock transform
  useEffect(() => {
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

  const { elementRef, floatingStyle, pointerHandlers, handleClickCapture, position } =
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
    const rect = el.getBoundingClientRect();
    if (rect.width > 0) setToolbarWidth(rect.width);
  }, [docked, elementRef]);

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
    if (!docked) return floatingStyle;
    const y = position.y;
    if (docked === "left") {
      // Slide left so only the rightmost DOCK_TAB_WIDTH pixels are visible
      return { transform: `translate3d(${-(toolbarWidth - DOCK_TAB_WIDTH)}px, ${y}px, 0)` };
    }
    // Slide right so only the leftmost DOCK_TAB_WIDTH pixels are visible
    return { transform: `translate3d(${vpWidth - DOCK_TAB_WIDTH}px, ${y}px, 0)` };
  }, [docked, floatingStyle, position.y, toolbarWidth, vpWidth]);

  const plusHandlers = useLongPress({
    onPress: onNewTerminal,
    onLongPress: onOpenLauncher,
  });

  // Voice button long-press logic
  const voicePressStartRef = useRef(0);
  const voiceWasRecordingRef = useRef(false);

  const handleVoicePointerDown = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
    e.stopPropagation(); // Prevent toolbar drag
    voicePressStartRef.current = Date.now();
    voiceWasRecordingRef.current = !!voiceRecording;
    if (!voiceRecording && !voiceTranscribing) {
      onVoiceStart?.({ vadEnabled: true });
    }
  }, [voiceRecording, voiceTranscribing, onVoiceStart]);

  const handleVoicePointerUp = useCallback((e: React.PointerEvent) => {
    e.stopPropagation();
    if (!voiceRecording) return;
    const duration = Date.now() - voicePressStartRef.current;
    if (voiceWasRecordingRef.current || duration >= VOICE_LONG_PRESS_MS) {
      onVoiceStop?.();
    }
  }, [voiceRecording, onVoiceStop]);

  // The dock tab indicator — renders on the visible edge
  const dockTab = docked ? (
    <div
      data-testid="dock-tab"
      className="flex items-center justify-center shrink-0 px-1 py-1"
      title="Tap to restore toolbar"
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
        data-testid="toolbar-sessions"
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        onClick={onOpenSessions}
        title="Sessions"
        tabIndex={docked ? -1 : undefined}
      >
        <List className="h-4 w-4" />
      </Button>
      <Button
        data-testid="toolbar-settings"
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        onClick={onOpenSettings}
        title="Settings"
        tabIndex={docked ? -1 : undefined}
      >
        <Settings className="h-4 w-4" />
      </Button>
      <Button
        data-testid="toolbar-ai"
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        onClick={onOpenAi}
        title="AI Command"
        tabIndex={docked ? -1 : undefined}
      >
        <Sparkles className="h-4 w-4" />
      </Button>
      {voiceSupported && onVoiceStart && onVoiceStop && (
        <div className="relative">
          <Button
            data-testid="toolbar-voice"
            variant="ghost"
            size="icon"
            className={`h-7 w-7 relative overflow-hidden ${voiceRecording ? "text-red-400" : voiceError ? "text-amber-400" : ""}`}
            onPointerDown={handleVoicePointerDown}
            onPointerUp={handleVoicePointerUp}
            onPointerCancel={handleVoicePointerUp}
            title={
              voiceRecording
                ? "Recording... tap to stop, or hold to talk"
                : voiceTranscribing
                  ? "Transcribing..."
                  : voiceError
                    ? `Voice error: ${voiceError}`
                    : `Tap to record, hold to talk${voiceBackend ? ` (${voiceBackend === "whisper" ? "Whisper" : "Browser"})` : ""}`
            }
            tabIndex={docked ? -1 : undefined}
          >
            {/* Audio level fill — rises from bottom */}
            {voiceRecording && (
              <span
                className="absolute inset-x-0 bottom-0 bg-red-500/30 rounded-[inherit] transition-[height] duration-75"
                style={{ height: `${Math.round(voiceLevel * 100)}%` }}
              />
            )}
            {voiceTranscribing ? (
              <Loader2 className="h-4 w-4 animate-spin relative" />
            ) : voiceError ? (
              <AlertCircle className="h-4 w-4 relative" />
            ) : (
              <Mic className="h-4 w-4 relative" />
            )}
          </Button>
          {voiceRecording && voicePartialTranscript && (
            <div className="absolute left-1/2 -translate-x-1/2 top-full mt-1 max-w-[200px] rounded border border-wc-default bg-wc-surface-raised px-2 py-1 text-[10px] text-wc-text-secondary shadow-lg pointer-events-none whitespace-nowrap overflow-hidden text-ellipsis z-10">
              {voicePartialTranscript}
            </div>
          )}
        </div>
      )}
      <Button
        data-testid="toolbar-new"
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        disabled={isCreating}
        title="New terminal (long-press for launcher)"
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

  return (
    <div
      ref={(node) => { elementRef.current = node; }}
      data-testid="floating-toolbar"
      className={`fixed left-0 top-0 z-[2600] flex items-center rounded-full border border-wc-default bg-wc-surface-raised/95 backdrop-blur-md shadow-lg select-none touch-none ${
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
