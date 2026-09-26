import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { Loader2, Pause, Play, X } from "lucide-react";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import type { ConversationEvent } from "../../api/conversation";
import { strings } from "../../consts/strings";
import { usePlaybackTransport } from "../../domains/tts-playback/transport";
import { useMediaQuery } from "../../hooks/useMediaQuery";
import { cn } from "../../lib/classnames";
import { speakerKey, timeLabel } from "../messages/speaker";
import type { PlaybackModeControlProps } from "./PlaybackModeControl";
import { PlaybackPillExpanded } from "./PlaybackPillExpanded";
import { formatClock } from "./pillFormat";
import { usePillGestures } from "./usePillGestures";

// DOC: docs/reference/tts-api.md#the-playback-pill

/** Hidden after a dismiss until playback restarts or the toolbar restores it. */
export type PillState = "hidden" | "collapsed" | "expanded";

export interface PlaybackPillProps {
  sessionId: string;
  /** The message being spoken, or the one a replay would speak. */
  event: ConversationEvent;
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
  /** False between tracks and in replay: the pill then reads as paused. */
  isSpeaking: boolean;
  isLoading?: boolean;
  /** Messages queued after the current one. */
  queuedAfter: number;
  canPrevious: boolean;
  canNext: boolean;
  voiceName?: string;
  backendReason?: string;
  summarize: Pick<
    PlaybackModeControlProps,
    "isSummarized" | "hasOriginalVersion" | "canSummarize" | "isSummarizing" | "currentLevel" | "onToggleSummarized" | "onChangeLevel"
  >;
  onPause: () => void;
  onResume: () => void;
  onSeek: (seconds: number) => void;
  onPrevious: () => void;
  onNext: () => void;
  /** Close and swipe-down: stops playback and hides the pill. */
  onStop: () => void;
  onJumpToMessage: () => void;
  onSetPlaybackRate: (rate: number) => void;
  onSetVolume: (level: number) => void;
  onSetMuted: (next: boolean) => void;
}

const EQUALIZER_BARS = ["h-1.5", "h-3", "h-2", "h-4"];

/**
 * The one playback surface: a floating pill over the pane that shows what is
 * speaking and expands into the full controls. It reads the pane's transport
 * store itself, so position updates re-render the pill and nothing else.
 */
export function PlaybackPill(props: PlaybackPillProps) {
  const { sessionId, event, expanded, onExpandedChange, isSpeaking, isLoading = false, onPause, onResume, onSeek, onNext, onPrevious, onStop } = props;
  const { t } = useTranslation();
  const transport = usePlaybackTransport(sessionId);
  const reducedMotion = useMediaQuery("(prefers-reduced-motion: reduce)");
  const progressRef = useRef<HTMLDivElement>(null);

  const paused = !isSpeaking || (transport?.isPaused ?? false);
  const currentTime = transport?.currentTime ?? 0;
  const duration = transport?.duration ?? null;
  const canSeek = (transport?.capabilities.canSeek ?? false) && duration !== null && duration > 0;
  const progress = duration ? Math.min(1, Math.max(0, currentTime / duration)) : 0;
  const speaker: string = t(speakerKey(event) as never);
  const label = `${speaker} · ${timeLabel(event.createdAt)}`;

  const gestures = usePillGestures({
    onTap: () => { onExpandedChange(!expanded); },
    onDismiss: onStop,
    onNext,
    onPrevious,
    onScrub: (ratio) => { if (canSeek && duration) onSeek(ratio * duration); },
    progressRef,
    swipeTracks: !expanded,
  });

  const equalizer = paused ? null : (
    <span data-testid="pill-equalizer" aria-hidden="true" className="flex h-4 shrink-0 items-end gap-px px-0.5">
      {EQUALIZER_BARS.map((height) => (
        <span key={height} className={cn("w-0.5 animate-pulse rounded-sm bg-wc-accent motion-reduce:animate-none", height)} />
      ))}
    </span>
  );

  const playPause = (
    <IconButton
      data-testid="pill-play-pause"
      size={expanded ? "md" : "sm"}
      onClick={paused ? onResume : onPause}
      disabled={isLoading || !(transport?.capabilities.canPause ?? true)}
      aria-label={isLoading ? t(strings.app.loading) : paused ? t(strings.playbackPill.play) : t(strings.playbackPill.pause)}
      className="shrink-0"
    >
      {isLoading ? <Loader2 className="animate-spin" /> : paused ? <Play /> : <Pause />}
    </IconButton>
  );

  return (
    <div
      data-testid="playback-pill"
      data-state={expanded ? "expanded" : "collapsed"}
      role="region"
      aria-label={t(strings.playbackPill.regionLabel)}
      onKeyDown={(keyEvent) => {
        if (keyEvent.key === "Escape" && expanded) onExpandedChange(false);
      }}
      {...gestures}
      className={cn(
        "wc-stable-theme pointer-events-auto relative touch-none select-none overflow-hidden border border-wc-default bg-wc-surface-raised/95 text-wc-text-primary shadow-lg backdrop-blur",
        expanded ? "w-[min(26rem,calc(100vw-2rem))] rounded-2xl" : "w-[min(24rem,calc(100vw-2rem))] rounded-full",
        !reducedMotion && "animate-in slide-in-from-bottom-2 duration-200",
      )}
    >
      {expanded ? (
        <PlaybackPillExpanded pill={props} transport={transport} label={label} equalizer={equalizer} playPause={playPause} />
      ) : (
        <>
          <div className="flex h-11 items-center gap-1.5 ps-1.5 pe-1">
            {playPause}
            {equalizer}
            <button
              type="button"
              data-testid="pill-summary"
              aria-expanded={false}
              aria-label={t(strings.playbackPill.expand)}
              onClick={() => { onExpandedChange(true); }}
              className="flex h-full min-w-0 flex-1 items-center gap-2 text-start"
            >
              <span data-testid="pill-label" className="min-w-0 truncate text-xs font-medium">{label}</span>
              <span data-testid="pill-time" className="ms-auto shrink-0 text-[11px] tabular-nums text-wc-text-muted">
                {formatClock(currentTime)} / {formatClock(duration)}
              </span>
            </button>
            <IconButton data-testid="pill-close" size="sm" aria-label={t(strings.playbackPill.close)} onClick={onStop} className="shrink-0">
              <X />
            </IconButton>
          </div>
          {/* The hairline's hit zone is taller than the line: a drag along it seeks. */}
          <div ref={progressRef} data-testid="pill-progress" aria-hidden="true" className="absolute inset-x-4 bottom-0 h-3">
            <div className="absolute inset-x-0 bottom-0 h-0.5 bg-wc-text-muted/20">
              <div data-testid="pill-progress-fill" className="h-full bg-wc-accent" style={{ width: `${(progress * 100).toFixed(1)}%` }} />
            </div>
          </div>
        </>
      )}
    </div>
  );
}
