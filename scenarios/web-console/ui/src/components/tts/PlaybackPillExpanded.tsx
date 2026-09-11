import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { ChevronLeft, ChevronRight, Settings2, X } from "lucide-react";
import { Chip } from "@vrooli/react-component-library/Chip/1";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { strings } from "../../consts/strings";
import type { PlaybackTransport } from "../../domains/tts-playback/transport";
import { AudioSettingsContent } from "./AudioSettingsContent";
import { PlaybackModeControl } from "./PlaybackModeControl";
import type { PlaybackPillProps } from "./PlaybackPill";
import { formatClock, nextRate } from "./pillFormat";
import { getScrubClasses } from "./scrubStyles";

/** Only an automatic fallback to browser speech earns a notice. */
function isBrowserFallback(reason: string | undefined): boolean {
  return Boolean(reason?.includes("browser speech synthesis is active") || reason?.includes("Browser handled playback"));
}

interface PlaybackPillExpandedProps {
  pill: PlaybackPillProps;
  transport: PlaybackTransport | null;
  label: string;
  equalizer: ReactNode;
  playPause: ReactNode;
}

/** The pill's full controls: header, scrub, transport, and chips. */
export function PlaybackPillExpanded({ pill, transport, label, equalizer, playPause }: PlaybackPillExpandedProps) {
  const { t } = useTranslation();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const currentTime = transport?.currentTime ?? 0;
  const duration = transport?.duration ?? null;
  const rate = transport?.playbackRate ?? 1;
  const canSeek = (transport?.capabilities.canSeek ?? false) && duration !== null && duration > 0;

  return (
    <div className="flex flex-col gap-2 px-2 pt-1.5 pb-2">
      <div className="flex items-center gap-1.5">
        {equalizer}
        <button
          type="button"
          data-testid="pill-summary"
          aria-expanded
          aria-label={t(strings.playbackPill.collapse)}
          onClick={() => { pill.onExpandedChange(false); }}
          className="flex min-h-11 min-w-0 flex-1 items-center gap-2 text-start"
        >
          <span data-testid="pill-label" className="min-w-0 truncate text-xs font-medium">{label}</span>
          {pill.queuedAfter > 0 && (
            <span data-testid="pill-queued" className="shrink-0 text-[11px] text-wc-text-muted">
              {t(strings.playbackPill.moreQueued, { count: pill.queuedAfter })}
            </span>
          )}
        </button>
        <IconButton
          data-testid="pill-settings"
          size="sm"
          disabled={!transport}
          aria-label={t(strings.playbackPill.settings)}
          onClick={() => { setSettingsOpen(true); }}
        >
          <Settings2 />
        </IconButton>
        <IconButton data-testid="pill-close" size="sm" aria-label={t(strings.playbackPill.close)} onClick={pill.onStop}>
          <X />
        </IconButton>
      </div>

      {isBrowserFallback(pill.backendReason) && (
        <p data-testid="pill-fallback-notice" className="px-1 text-[11px] text-wc-text-muted">{pill.backendReason}</p>
      )}

      <div className="flex items-center gap-2 px-1 text-[11px] tabular-nums text-wc-text-muted">
        <span>{formatClock(currentTime)}</span>
        <input
          data-testid="pill-scrub"
          type="range"
          min={0}
          max={canSeek ? duration : 0}
          step={0.1}
          value={canSeek ? currentTime : 0}
          disabled={!canSeek}
          onChange={(event) => { pill.onSeek(Number(event.target.value)); }}
          aria-label={t(strings.playbackPill.seek)}
          className={getScrubClasses({ isSummarized: pill.summarize.isSummarized, enabled: canSeek, extra: "flex-1" })}
        />
        <span>{formatClock(duration)}</span>
      </div>

      <div className="flex items-center justify-center gap-4">
        <IconButton data-testid="pill-previous" size="sm" disabled={!pill.canPrevious} aria-label={t(strings.playbackPill.previous)} onClick={pill.onPrevious}>
          <ChevronLeft />
        </IconButton>
        {playPause}
        <IconButton data-testid="pill-next" size="sm" disabled={!pill.canNext} aria-label={t(strings.playbackPill.next)} onClick={pill.onNext}>
          <ChevronRight />
        </IconButton>
      </div>

      <div className="flex flex-wrap items-center gap-1.5 px-1">
        {transport?.capabilities.canAdjustSpeed && (
          <span data-testid="pill-rate">
            <Chip aria-label={t(strings.playbackPill.speed)} onClick={() => { pill.onSetPlaybackRate(nextRate(rate)); }}>
              {`${String(rate)}×`}
            </Chip>
          </span>
        )}
        <PlaybackModeControl testIdPrefix="pill" {...pill.summarize} />
        {pill.voiceName && (
          <span data-testid="pill-voice" className="max-w-[10rem] truncate rounded-full bg-wc-surface-input/60 px-2.5 py-1 text-[11px] text-wc-text-muted">
            {pill.voiceName}
          </span>
        )}
        <span data-testid="pill-jump">
          <Chip onClick={pill.onJumpToMessage}>{t(strings.playbackPill.jumpToMessage)}</Chip>
        </span>
      </div>

      {settingsOpen && transport && (
        <ResponsiveDialog
          open
          onClose={() => { setSettingsOpen(false); }}
          size="md"
          title={t(strings.playbackPill.settingsTitle)}
          closeLabel={t(strings.playbackPill.settingsClose)}
          testId="pill-settings-dialog"
          // Volume, mute, and speed: no text field, so there is no keyboard to avoid.
          avoidKeyboard={false}
        >
          <AudioSettingsContent
            testIdPrefix="pill"
            volume={transport.volume}
            isMuted={transport.isMuted}
            playbackRate={transport.playbackRate}
            isSummarized={pill.summarize.isSummarized}
            capabilities={transport.capabilities}
            onVolumeChange={pill.onSetVolume}
            onSetMuted={pill.onSetMuted}
            onSetPlaybackRate={pill.onSetPlaybackRate}
          />
        </ResponsiveDialog>
      )}
    </div>
  );
}
