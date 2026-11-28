/**
 * @libraryId react-component-library:TimeMention
 * @displayName TimeMention
 * @description Times, dates and durations in prose, put into the reader's frame: local time beside the original, live deadline countdowns, and a details card.
 * @version 1.0.2
 * @tags ["data-display","markdown","time","i18n"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/**
 * TimeMention turns a time, date or duration written in prose into something
 * the reader can use: their local time beside the original, a live countdown
 * for deadlines, and a details card with the moment in every frame.
 *
 * A mention's meaning depends on three frames — where the reader is, when the
 * text was written, and when it is read. The host supplies the second as an
 * optional anchor; without it, anything that depends on it silently drops
 * away rather than showing an "unknown".
 */
import {
  useMemo,
  useState,
  type ComponentPropsWithoutRef,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";
import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton/1";
import { Popover, PopoverParts } from "@vrooli/react-component-library/Popover/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useLocale, useStrings } from "@vrooli/react-component-library/useLocale/1";
import {
  detectTimeMentions,
  type TimeMentionData,
  type TimeMentionRole,
} from "./timeMentionGrammar";
import {
  TIME_MENTION_DISPLAYS,
  calendarDay,
  deadlineState,
  deadlineText,
  formatClock,
  formatDate,
  formatDayDistance,
  formatDayLabel,
  formatDuration,
  isNight,
  relativeText,
  resolveTimeMention,
  zoneName,
  zoneOffsetAt,
  type ResolvedTimeMention,
  type TimeFrame,
  type TimeMentionDisplay,
  type TimeMentionText,
} from "./timeMentionResolve";
import {
  defaultTimeZone,
  normalizeAnchor,
  useTimeMentionDisplay,
  useTimeMentionNow,
  useTimeMentionSettings,
  type TimeAnchorInput,
} from "./timeMentionState";
import { TIME_MENTION_ATTRIBUTE, readTimeMentionPayload } from "./remarkTimeMentions";
import { timeMentionStyles } from "./timeMentionStyles";

export * from "./timeMentionGrammar";
export * from "./timeMentionResolve";
export * from "./timeMentionState";
export * from "./remarkTimeMentions";
export { timeMentionStyles } from "./timeMentionStyles";

const STYLE_VERSION = "1.0.2";
const DAY = 86_400_000;

type Translate = (key: string, fallback: string) => string;

function fill(translate: Translate, text: TimeMentionText): string {
  return translate(text.key, text.fallback).replace(
    /\{(\w+)\}/g,
    (token, name: string) => text.values?.[name] ?? token,
  );
}

const DISPLAY_LABELS: Record<TimeMentionDisplay, TimeMentionText> = {
  annotate: { key: "data-display.time-mention.display.annotate", fallback: "Annotate" },
  local: { key: "data-display.time-mention.display.local", fallback: "Local" },
  relative: { key: "data-display.time-mention.display.relative", fallback: "Relative" },
};

function kindLabel(resolved: ResolvedTimeMention, data: TimeMentionData): TimeMentionText {
  if (data.kind === "instant" && data.source === "epoch")
    return { key: "data-display.time-mention.kind.epoch", fallback: "Unix timestamp" };
  const labels: Record<TimeMentionData["kind"], TimeMentionText> = {
    instant: { key: "data-display.time-mention.kind.instant", fallback: "Exact time" },
    clock: resolved.dateKnown
      ? { key: "data-display.time-mention.kind.clock", fallback: "Time, date inferred" }
      : { key: "data-display.time-mention.kind.clock-only", fallback: "Time of day" },
    date: { key: "data-display.time-mention.kind.date", fallback: "Date" },
    "relative-day": {
      key: "data-display.time-mention.kind.relative-day",
      fallback: "Relative day",
    },
    "relative-offset": {
      key: "data-display.time-mention.kind.relative-offset",
      fallback: "Relative time",
    },
    duration: { key: "data-display.time-mention.kind.duration", fallback: "Duration" },
    unzoned: { key: "data-display.time-mention.kind.unzoned", fallback: "Time without a zone" },
  };
  return labels[data.kind];
}

interface Presentation {
  face: string;
  tag: { text: string; tone: "default" | "lint" | "stale" | "unzoned"; night: boolean } | null;
  badge: { text: string; state: "upcoming" | "soon" | "past" } | null;
  tone: "default" | "lint" | "unzoned";
}

function localLabel(resolved: ResolvedTimeMention, frame: TimeFrame): string {
  const instant = resolved.instant ?? frame.now;
  const day = resolved.dateKnown ? formatDayLabel(instant, frame) : "";
  return day ? `${formatClock(instant, frame)} ${day}` : formatClock(instant, frame);
}

function present(
  text: string,
  data: TimeMentionData,
  resolved: ResolvedTimeMention,
  frame: TimeFrame,
  display: TimeMentionDisplay,
  translate: Translate,
): Presentation {
  const presentation: Presentation = { face: text, tag: null, badge: null, tone: "default" };
  const { instant } = resolved;
  if (instant !== null) {
    if (display === "relative" && resolved.dateKnown)
      presentation.face = fill(translate, relativeText(instant, frame.now));
    else if (display !== "annotate") presentation.face = localLabel(resolved, frame);
    const sameZone =
      resolved.writtenOffset !== null &&
      resolved.writtenOffset === zoneOffsetAt(instant, frame.timeZone);
    if (display === "annotate" && !sameZone)
      presentation.tag = {
        text: localLabel(resolved, frame),
        tone: "default",
        night: isNight(instant, frame.timeZone),
      };
    if (resolved.role && resolved.dateKnown)
      presentation.badge = {
        text: fill(translate, deadlineText(instant, resolved.role, frame.now)),
        state: deadlineState(instant, frame.now),
      };
    return presentation;
  }
  if (resolved.lint) {
    presentation.tone = "lint";
    presentation.tag = { text: fill(translate, resolved.lint), tone: "lint", night: false };
  } else if (data.kind === "relative-day" && resolved.day !== null) {
    presentation.tag = {
      text: new Intl.DateTimeFormat(frame.locale, {
        timeZone: "UTC",
        weekday: "short",
        month: "short",
        day: "numeric",
      }).format(resolved.day),
      tone: "stale",
      night: false,
    };
  } else if (data.kind === "date" && resolved.day !== null) {
    const distance = Math.abs(resolved.day - calendarDay(frame.now, frame.timeZone)) / DAY;
    if (distance <= 7)
      presentation.tag = {
        text: formatDayDistance(resolved.day, frame, false),
        tone: "default",
        night: false,
      };
  } else if (resolved.durationMs !== null) {
    presentation.tag = { text: formatDuration(resolved.durationMs), tone: "default", night: false };
  } else if (data.kind === "unzoned") {
    presentation.tone = "unzoned";
    presentation.tag = {
      text: translate("data-display.time-mention.tag.unzoned", "zone?"),
      tone: "unzoned",
      night: false,
    };
  }
  return presentation;
}

function Glyph({ name }: { name: "moon" | "info" | "warning" }) {
  const paths = {
    moon: (
      <path fill="currentColor" d="M20.5 14.6A8.5 8.5 0 0 1 9.4 3.5a8.5 8.5 0 1 0 11.1 11.1Z" />
    ),
    info: (
      <>
        <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" strokeWidth="2" />
        <path d="M12 11v6M12 7.5v.01" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      </>
    ),
    warning: (
      <>
        <path
          d="M12 3.5 2.5 20h19L12 3.5Z"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinejoin="round"
        />
        <path
          d="M12 10v4.5M12 17.5v.01"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
        />
      </>
    ),
  };
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      focusable="false"
      data-night={name === "moon" ? "" : undefined}
    >
      {paths[name]}
    </svg>
  );
}

function isoUtc(ms: number, seconds: boolean): string {
  const iso = new Date(ms).toISOString();
  return seconds ? iso.replace(/\.\d{3}Z$/, "Z") : iso.replace(/:\d{2}\.\d{3}Z$/, "Z");
}

interface DetailsProps {
  text: string;
  data: TimeMentionData;
  resolved: ResolvedTimeMention;
  frame: TimeFrame;
  display: TimeMentionDisplay;
  onDisplayChange: (display: TimeMentionDisplay) => void;
  translate: Translate;
}

function TimeMentionDetails({
  text,
  data,
  resolved,
  frame,
  display,
  onDisplayChange,
  translate,
}: DetailsProps) {
  const t = (value: TimeMentionText) => fill(translate, value);
  const { instant } = resolved;
  let primary: ReactNode = text;
  let secondary = "";
  const rows: Array<{ term: string; value: string; copy?: { value: string; label: string } }> = [];

  if (instant !== null) {
    primary = (
      <>
        {isNight(instant, frame.timeZone) ? <Glyph name="moon" /> : null}
        {formatClock(instant, frame, resolved.showSeconds)}
      </>
    );
    const zone = zoneName(instant, frame);
    if (resolved.dateKnown) {
      const day = formatDayLabel(instant, frame);
      secondary = [formatDate(instant, frame, "long"), zone, day].filter(Boolean).join(" · ");
      const iso = isoUtc(instant, resolved.showSeconds);
      rows.push(
        {
          term: translate("data-display.time-mention.row.utc", "UTC"),
          value: iso,
          copy: {
            value: iso,
            label: translate("data-display.time-mention.copy-iso", "Copy UTC timestamp"),
          },
        },
        {
          term: translate("data-display.time-mention.row.from-now", "From now"),
          value: t(relativeText(instant, frame.now)),
        },
      );
      if (frame.anchor !== null && data.kind !== "relative-offset") {
        const after = instant >= frame.anchor;
        rows.push({
          term: translate("data-display.time-mention.row.from-sent", "From sending"),
          value: t({
            key: after
              ? "data-display.time-mention.after-sent"
              : "data-display.time-mention.before-sent",
            fallback: after ? "{duration} after" : "{duration} before",
            values: { duration: formatDuration(instant - frame.anchor) },
          }),
        });
      }
    } else {
      secondary = `${zone} · ${translate("data-display.time-mention.day-unknown", "day unknown")}`;
      rows.push({
        term: translate("data-display.time-mention.row.utc", "UTC"),
        value: isoUtc(instant, true).slice(11, resolved.showSeconds ? 19 : 16),
      });
    }
  } else if (resolved.day !== null && data.kind !== "unzoned") {
    primary = new Intl.DateTimeFormat(frame.locale, {
      timeZone: "UTC",
      month: "short",
      day: "numeric",
      year: "numeric",
    }).format(resolved.day);
    secondary = [
      new Intl.DateTimeFormat(frame.locale, { timeZone: "UTC", weekday: "long" }).format(
        resolved.day,
      ),
      formatDayDistance(resolved.day, frame, false),
    ].join(" · ");
  } else if (resolved.durationMs !== null) {
    primary = formatDuration(resolved.durationMs);
    secondary = t({
      key: "data-display.time-mention.seconds",
      fallback: "{count} seconds",
      values: { count: (resolved.durationMs / 1000).toLocaleString(frame.locale) },
    });
  } else {
    secondary = translate("data-display.time-mention.no-zone", "No time zone written");
  }

  const showDisplay = instant !== null;
  return (
    <div data-rcl-time-mention-card="">
      <header data-rcl-time-mention-card-head="">
        <p data-rcl-time-mention-card-primary="">{primary}</p>
        {secondary ? <p data-rcl-time-mention-card-secondary="">{secondary}</p> : null}
        <p data-rcl-time-mention-card-source="">
          {t(kindLabel(resolved, data))} ·{" "}
          {translate("data-display.time-mention.written-as", "written as")} <code>{text}</code>
        </p>
      </header>
      {rows.length > 0 ? (
        <dl data-rcl-time-mention-card-rows="">
          {rows.map((row) => (
            <div key={row.term} style={{ display: "contents" }}>
              <dt>{row.term}</dt>
              <dd>
                <span>{row.value}</span>
                {row.copy ? (
                  <CopyIconButton size="xs" value={row.copy.value} aria-label={row.copy.label} />
                ) : null}
              </dd>
            </div>
          ))}
        </dl>
      ) : null}
      {resolved.notes.map((note) => (
        <p key={note.key} data-rcl-time-mention-card-note="" data-tone={note.tone}>
          <Glyph name={note.tone === "warning" ? "warning" : "info"} />
          <span>{t(note)}</span>
        </p>
      ))}
      {showDisplay ? (
        <section data-rcl-time-mention-card-display="">
          <div data-rcl-time-mention-card-display-label="">
            <span>{translate("data-display.time-mention.display.label", "Show times as")}</span>
            <span>
              {translate("data-display.time-mention.display.scope", "Applies everywhere")}
            </span>
          </div>
          <ButtonGroup
            variant="segmented"
            collapse="never"
            label={translate("data-display.time-mention.display.label", "Show times as")}
          >
            {TIME_MENTION_DISPLAYS.map((option) => (
              <Button
                key={option}
                variant="ghost"
                size="sm"
                aria-pressed={display === option}
                onClick={() => onDisplayChange(option)}
              >
                {t(DISPLAY_LABELS[option])}
              </Button>
            ))}
          </ButtonGroup>
        </section>
      ) : null}
    </div>
  );
}

export interface TimeMentionProps {
  /** The text exactly as written. */
  text: string;
  data: TimeMentionData;
  role?: TimeMentionRole | null;
  /** When the text was written; overrides the provider's anchor. */
  anchor?: TimeAnchorInput;
}

export function TimeMention({ text, data, role = null, anchor }: TimeMentionProps) {
  useLibraryStyleSheet("react-component-library:TimeMention", STYLE_VERSION, timeMentionStyles);
  const settings = useTimeMentionSettings();
  const translate = useStrings();
  const documentLocale = useLocale();
  const now = useTimeMentionNow();
  const [display, setDisplay] = useTimeMentionDisplay();
  const [open, setOpen] = useState(false);
  const anchorMs = normalizeAnchor(anchor !== undefined ? anchor : settings.anchor);
  const timeZone = settings.timeZone ?? defaultTimeZone();
  const locale = settings.locale ?? documentLocale;
  const frame = useMemo<TimeFrame>(
    () => ({ anchor: anchorMs, now, timeZone, locale }),
    [anchorMs, now, timeZone, locale],
  );
  const resolved = useMemo(() => resolveTimeMention(data, role, frame), [data, role, frame]);

  if (settings.enabled === false || !resolved) return <>{text}</>;
  const presentation = present(text, data, resolved, frame, display, translate);
  const accessibleName = [
    presentation.face === text ? text : `${presentation.face} (${text})`,
    presentation.tag?.text,
    presentation.badge?.text,
  ]
    .filter(Boolean)
    .join(", ");
  const onKeyDown = (event: KeyboardEvent<HTMLSpanElement>) => {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    setOpen(!open);
  };

  return (
    <Popover inline open={open} onOpenChange={setOpen} placement="bottom-start">
      <PopoverParts.Trigger asChild>
        <span
          role="button"
          tabIndex={0}
          data-rcl-time-mention=""
          data-kind={data.kind}
          data-tone={presentation.tone}
          aria-label={accessibleName}
          onKeyDown={onKeyDown}
        >
          <time
            dateTime={
              resolved.instant !== null && resolved.dateKnown
                ? new Date(resolved.instant).toISOString()
                : undefined
            }
          >
            {presentation.face}
          </time>
          {presentation.tag ? (
            <span data-rcl-time-mention-tag="" data-tone={presentation.tag.tone} aria-hidden="true">
              {presentation.tag.night ? <Glyph name="moon" /> : null}
              {presentation.tag.tone === "lint" ? <Glyph name="warning" /> : null}
              {presentation.tag.text}
            </span>
          ) : null}
          {presentation.badge ? (
            <span
              data-rcl-time-mention-badge=""
              data-state={presentation.badge.state}
              aria-hidden="true"
            >
              {presentation.badge.text}
            </span>
          ) : null}
        </span>
      </PopoverParts.Trigger>
      <PopoverParts.Content
        aria-label={translate("data-display.time-mention.details", "Time details")}
        initialFocus="content"
      >
        <TimeMentionDetails
          text={text}
          data={data}
          resolved={resolved}
          frame={frame}
          display={display}
          onDisplayChange={setDisplay}
          translate={translate}
        />
      </PopoverParts.Content>
    </Popover>
  );
}

/**
 * Plain text with its time mentions made interactive, for surfaces that are
 * not markdown — a log line, a title, a status message.
 */
export function TimeMentionProse({ text }: { text: string }) {
  const mentions = useMemo(() => detectTimeMentions(text), [text]);
  const parts: ReactNode[] = [];
  let cursor = 0;
  for (const mention of mentions) {
    if (mention.start > cursor) parts.push(text.slice(cursor, mention.start));
    parts.push(
      <TimeMention
        key={mention.start}
        text={mention.text}
        data={mention.data}
        role={mention.role}
      />,
    );
    cursor = mention.end;
  }
  if (cursor < text.length) parts.push(text.slice(cursor));
  return <>{parts}</>;
}

/**
 * The `time` element renderer for react-markdown. Elements written by
 * `remarkTimeMentions` become mentions; any other `time` renders unchanged.
 */
export type MarkdownTimeMentionProps = ComponentPropsWithoutRef<"time"> & { node?: unknown };

export function MarkdownTimeMention({
  children,
  node: _node,
  dateTime,
  ...props
}: MarkdownTimeMentionProps) {
  const payload = readTimeMentionPayload(
    (props as Record<string, unknown>)[TIME_MENTION_ATTRIBUTE],
  );
  if (!payload) return <time dateTime={dateTime}>{children}</time>;
  return <TimeMention text={payload.text} data={payload.data} role={payload.role} />;
}

export default TimeMention;
