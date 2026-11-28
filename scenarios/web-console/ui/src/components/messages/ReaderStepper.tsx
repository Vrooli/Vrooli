import { useEffect, useState } from "react";
import type { CSSProperties, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, Undo2 } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { strings } from "../../consts/strings";
import { useHoldToScrub, type ScrubState, type ScrubTier } from "../../hooks/useHoldToScrub";
import type { ReplyPosition } from "./replyTrack";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-reader

/** How long the speed readout stays after release, so the landing is readable. */
const HUD_LINGER_MS = 900;
/** A quick tap never shows the hold ring; only a press that might become a hold. */
const RING_SHOW_AFTER_MS = 120;

type Direction = "prev" | "next";

interface ReaderStepperProps {
  /** Move `delta` replies (negative for back); returns how many it moved. */
  onSeek: (delta: number) => number;
  canPrev: boolean;
  canNext: boolean;
  position: ReplyPosition;
  /** Replies that arrived after everything the reader has shown. */
  newReplyCount: number;
  onReturn: () => void;
}

const STEPPER_CSS = `
[data-reader-scrub] { position: relative; display: inline-grid; touch-action: none; -webkit-touch-callout: none; user-select: none; -webkit-user-select: none; }
[data-reader-scrub] [data-rcl-icon-button] { transition: transform var(--dur-quick, 180ms) var(--ease-standard, ease), background-color var(--dur-quick, 180ms) ease; }
[data-reader-scrub][data-phase="scrubbing"] [data-rcl-icon-button] { transform: scale(1.08); background: var(--color-primary); border-color: var(--color-primary); color: var(--color-primary-foreground); }
[data-reader-scrub-ring] { position: absolute; inset: -4px; border-radius: 9999px; pointer-events: none; opacity: 0;
  background: conic-gradient(var(--color-primary) calc(var(--reader-ring-progress, 0) * 1turn), transparent 0);
  -webkit-mask: radial-gradient(farthest-side, transparent calc(100% - 3px), #000 calc(100% - 2px));
          mask: radial-gradient(farthest-side, transparent calc(100% - 3px), #000 calc(100% - 2px)); }
@property --reader-ring-progress { syntax: "<number>"; inherits: false; initial-value: 0; }
[data-reader-scrub][data-phase="pressed"] [data-reader-scrub-ring] { animation: reader-ring-fill var(--reader-hold-ms) linear forwards, reader-ring-show 1ms linear ${RING_SHOW_AFTER_MS}ms forwards; }
@keyframes reader-ring-fill { from { --reader-ring-progress: 0; } to { --reader-ring-progress: 1; } }
@keyframes reader-ring-show { to { opacity: 1; } }

[data-reader-hud] { position: absolute; inset-inline: 0; inset-block-end: calc(100% + 0.75rem); display: flex; justify-content: center; pointer-events: none;
  opacity: 0; transform: translateY(6px) scale(0.96); transition: opacity 160ms ease, transform 160ms var(--ease-standard, ease); }
[data-reader-hud][data-visible="true"] { opacity: 1; transform: none; }
[data-reader-hud-pill][data-edge="true"] { animation: reader-hud-bump 260ms var(--ease-standard, ease); }
@keyframes reader-hud-bump { 30% { transform: translateX(var(--reader-bump, 0)); } }
[data-reader-hud-track] { position: relative; block-size: 3px; border-radius: 9999px; background: color-mix(in srgb, currentColor 18%, transparent); overflow: hidden; }
[data-reader-hud-fill] { position: absolute; inset-block: 0; inset-inline-start: 0; border-radius: inherit; background: var(--color-primary); transition: inline-size 90ms linear; }

@media (prefers-reduced-motion: reduce) {
  [data-reader-scrub] [data-rcl-icon-button], [data-reader-hud], [data-reader-hud-fill] { transition: none; }
  [data-reader-scrub][data-phase="scrubbing"] [data-rcl-icon-button] { transform: none; }
  [data-reader-hud-pill][data-edge="true"] { animation: none; }
}
`;

interface ScrubButtonProps {
  direction: Direction;
  state: ScrubState;
  holdDelay: number;
  disabled: boolean;
  label: string;
  pressProps: ReturnType<typeof useHoldToScrub>["pressProps"];
  children?: ReactNode;
  newCount?: number;
}

function ScrubButton({ direction, state, holdDelay, disabled, label, pressProps, children, newCount }: ScrubButtonProps) {
  const scrubbing = state.phase === "scrubbing";
  const Glyph = direction === "prev" ? (scrubbing ? ChevronsLeft : ChevronLeft) : (scrubbing ? ChevronsRight : ChevronRight);
  return (
    <span
      data-reader-scrub
      data-phase={state.phase}
      style={{ "--reader-hold-ms": `${String(holdDelay)}ms` } as CSSProperties}
    >
      <IconButton
        data-testid={`reader-${direction}-reply`}
        data-scrub-phase={state.phase}
        data-new-count={newCount && newCount > 0 ? newCount : undefined}
        aria-label={label}
        title={label}
        surface={newCount && newCount > 0 ? "solid" : "soft"}
        shape="circle"
        size="md"
        // An end disables the button, but never under a finger that is still
        // holding it: focus and the release must stay with the control.
        disabled={disabled && state.phase === "idle"}
        {...pressProps}
      >
        <Glyph />
      </IconButton>
      <span data-reader-scrub-ring aria-hidden />
      {children}
    </span>
  );
}

/**
 * The reader's footer: step back, return to the list, step forward. A tap on
 * either arrow steps one reply; a hold scrubs through them like a video
 * player's fast-forward, with a readout of speed and place above the footer.
 */
export function ReaderStepper({ onSeek, canPrev, canNext, position, newReplyCount, onReturn }: ReaderStepperProps) {
  const { t } = useTranslation();
  const [linger, setLinger] = useState<{ direction: Direction; state: ScrubState } | null>(null);

  const prev = useHoldToScrub({
    step: (count) => onSeek(-count),
    canStep: canPrev,
    onScrubEnd: (final) => { setLinger({ direction: "prev", state: final }); },
  });
  const next = useHoldToScrub({
    step: (count) => onSeek(count),
    canStep: canNext,
    onScrubEnd: (final) => { setLinger({ direction: "next", state: final }); },
  });

  useEffect(() => {
    if (!linger) return;
    const timer = window.setTimeout(() => { setLinger(null); }, HUD_LINGER_MS);
    return () => { window.clearTimeout(timer); };
  }, [linger]);

  const live: { direction: Direction; state: ScrubState } | null =
    prev.state.phase === "scrubbing" ? { direction: "prev", state: prev.state }
    : next.state.phase === "scrubbing" ? { direction: "next", state: next.state }
    : null;
  const shown = live ?? linger;
  const tiers: readonly ScrubTier[] = (shown?.direction === "prev" ? prev : next).tiers;

  const nextLabel = newReplyCount > 0
    ? t(strings.reader.nextReplyNew, { count: newReplyCount })
    : t(strings.reader.nextReplyHold);

  return (
    <>
      <style>{STEPPER_CSS}</style>
      <ScrubHud
        shown={shown}
        visible={live != null}
        tiers={tiers}
        position={position}
      />
      <ScrubButton
        direction="prev"
        state={prev.state}
        holdDelay={prev.holdDelay}
        disabled={!canPrev}
        label={t(strings.reader.prevReplyHold)}
        pressProps={prev.pressProps}
      />
      <Button
        type="button"
        variant="secondary"
        data-testid="reader-return"
        title={t(strings.reader.returnTitle)}
        onClick={onReturn}
        className="min-h-11 min-w-0 flex-1 md:min-h-9"
      >
        <Undo2 aria-hidden className="me-2 h-4 w-4 shrink-0" />
        <span className="truncate">{t(strings.reader.return)}</span>
      </Button>
      <ScrubButton
        direction="next"
        state={next.state}
        holdDelay={next.holdDelay}
        disabled={!canNext}
        label={nextLabel}
        pressProps={next.pressProps}
        newCount={newReplyCount}
      >
        {newReplyCount > 0 && (
          <span
            aria-hidden
            className="pointer-events-none absolute -end-1 -top-1 min-w-4 rounded-full bg-wc-accent px-1 text-center text-[10px] font-semibold leading-4 text-wc-accent-fg"
          >
            {newReplyCount}
          </span>
        )}
      </ScrubButton>
    </>
  );
}

interface ScrubHudProps {
  shown: { direction: Direction; state: ScrubState } | null;
  /** False while lingering after release: fading, but still readable. */
  visible: boolean;
  tiers: readonly ScrubTier[];
  position: ReplyPosition;
}

function ScrubHud({ shown, visible, tiers, position }: ScrubHudProps) {
  const { t } = useTranslation();
  const direction = shown?.direction ?? "next";
  const tier = shown?.state.tier ?? 0;
  const atEdge = shown?.state.atEdge ?? false;
  const speed = tiers[tier]?.speed ?? 1;
  const Glyph = direction === "prev" ? ChevronsLeft : ChevronsRight;
  const fraction = position.total > 1 ? (position.ordinal - 1) / (position.total - 1) : 1;
  const place = t(strings.reader.scrubPosition, { current: position.ordinal, total: position.total });
  return (
    <div
      data-reader-hud
      data-testid="reader-scrub-hud"
      data-visible={visible}
      data-direction={direction}
      aria-hidden={!shown}
    >
      <div
        data-reader-hud-pill
        data-edge={atEdge}
        style={{ "--reader-bump": direction === "prev" ? "-6px" : "6px" } as CSSProperties}
        className="flex min-w-44 flex-col gap-1.5 rounded-2xl border border-wc-default bg-wc-surface-raised/95 px-4 py-2 text-wc-text-primary shadow-xl backdrop-blur-md"
      >
        <div className="flex items-center justify-between gap-3">
          <span className="flex items-center gap-1.5 text-sm font-semibold">
            <Glyph aria-hidden className="h-4 w-4 text-wc-accent" />
            {atEdge ? (
              <span data-testid="reader-scrub-edge" data-edge={direction === "prev" ? "first" : "latest"}>{t(direction === "prev" ? strings.reader.scrubFirst : strings.reader.scrubLatest)}</span>
            ) : (
              <>
                <span className="sr-only">{t(direction === "prev" ? strings.reader.rewinding : strings.reader.fastForwarding)}</span>
                <span data-testid="reader-scrub-speed" data-speed={speed} className="font-mono tabular-nums">{t(strings.reader.scrubSpeed, { speed })}</span>
              </>
            )}
          </span>
          <span aria-hidden className="flex items-center gap-0.5">
            {tiers.map((candidate, index) => (
              <span
                key={candidate.speed}
                className={`h-1.5 w-1.5 rounded-full ${index <= tier ? "bg-wc-accent" : "bg-wc-text-faint/40"}`}
              />
            ))}
          </span>
        </div>
        <div data-reader-hud-track aria-hidden className="text-wc-text-faint">
          <div data-reader-hud-fill style={{ inlineSize: `${String(Math.round(fraction * 1000) / 10)}%` }} />
        </div>
        <span data-testid="reader-scrub-position" data-current={position.ordinal} data-total={position.total} className="text-center font-mono text-[11px] tabular-nums text-wc-text-secondary">
          {place}
        </span>
      </div>
      {/* Spoken once the scrub settles, never per step. */}
      <span role="status" className="sr-only">
        {shown && !visible ? t(strings.reader.scrubPositionSpoken, { current: position.ordinal, total: position.total }) : ""}
      </span>
    </div>
  );
}
