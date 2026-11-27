import { Card } from "@vrooli/react-component-library/Card/1";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import type { CSSProperties, HTMLAttributes, ReactNode } from "react";

/**
 * The small Daylight grammar shared by every surface.
 *
 * These are compositions, not a second component library: the card is the
 * shared `Card` asset in its product variant, and every colour, radius and
 * shadow below resolves to the tokens in `src/theme/daylight.css`.
 */

export type Tone = "neutral" | "blue" | "lavender" | "peach" | "mint" | "lime" | "ink";

export function Panel({
  children,
  className,
  tone,
  interactive,
  ...rest
}: HTMLAttributes<HTMLDivElement> & {
  children: ReactNode;
  tone?: Exclude<Tone, "neutral">;
  interactive?: boolean;
}) {
  return (
    <Card
      variant={tone ? "pastel" : "surface"}
      tone={tone === "lime" || tone === "ink" ? undefined : tone}
      padding="comfortable"
      interactive={interactive}
      className={className}
      {...rest}
    >
      {children}
    </Card>
  );
}

export function PanelHead({
  title,
  note,
  action,
  id,
}: {
  title: ReactNode;
  note?: ReactNode;
  action?: ReactNode;
  id?: string;
}) {
  return (
    <div className="rd-card-head">
      <div>
        <h2 className="rd-card-title" id={id}>{title}</h2>
        {note ? <p className="rd-card-note">{note}</p> : null}
      </div>
      {action ? <div className="rd-card-action">{action}</div> : null}
    </div>
  );
}

export function IconTile({
  children,
  tone = "neutral",
  size = "md",
  state,
  className,
}: {
  children: ReactNode;
  tone?: Tone;
  size?: "sm" | "md" | "lg";
  state?: "live" | "wait" | "stop" | "quiet";
  className?: string;
}) {
  const toneClass = tone === "neutral" ? "" : ` rd-tile--${tone}`;
  const sizeClass = size === "md" ? "" : ` rd-tile--${size}`;
  return (
    <span
      aria-hidden="true"
      className={`rd-tile${toneClass}${sizeClass}${state ? " rd-tile-dot" : ""}${className ? ` ${className}` : ""}`}
      data-state={state === "live" ? undefined : state}
    >
      {children}
    </span>
  );
}

export function Ring({
  value,
  label,
  tone = "blue",
  size = "md",
}: {
  /** 0–100. Values are clamped; a negative or unmeasured value draws an empty track. */
  value: number;
  label: ReactNode;
  tone?: "blue" | "amber" | "red" | "muted";
  size?: "md" | "lg";
}) {
  const clamped = Number.isFinite(value) ? Math.min(100, Math.max(0, value)) : 0;
  const toneClass = tone === "blue" ? "" : ` rd-ring--${tone}`;
  return (
    <span
      className={`rd-ring${toneClass}${size === "lg" ? " rd-ring--lg" : ""}`}
      style={{ "--rd-ring-value": `${clamped}%` } as CSSProperties}
      aria-hidden="true"
    >
      <b>{label}</b>
    </span>
  );
}

export function StatPill({
  value,
  label,
  tone = "neutral",
  dot,
  href,
}: {
  value: ReactNode;
  label: ReactNode;
  tone?: "neutral" | "live" | "wait" | "quiet";
  dot?: boolean;
  href?: string;
}) {
  const content = (
    <>
      <span className="rd-stat-value">{value}</span>
      <span className="rd-stat-label">
        {dot ? <i className={`rd-dot${tone === "live" ? "" : tone === "wait" ? " rd-dot--wait" : tone === "quiet" ? " rd-dot--quiet" : ""}`} aria-hidden="true" /> : null}
        {label}
      </span>
    </>
  );
  return href ? (
    <a className="rd-stat" href={href}>{content}</a>
  ) : (
    <span className="rd-stat">{content}</span>
  );
}

export function StatusPill({
  children,
  tone = "quiet",
  dot,
}: {
  children: ReactNode;
  tone?: "quiet" | "blue" | "ink" | "lime" | "mint" | "peach" | "lav";
  dot?: "live" | "wait" | "stop" | "quiet" | "blue";
}) {
  return (
    <span className={`rd-pill rd-pill--${tone}`}>
      {dot ? <i className={`rd-dot${dot === "live" ? "" : ` rd-dot--${dot}`}`} aria-hidden="true" /> : null}
      {children}
    </span>
  );
}

export function RowLine({
  tile,
  title,
  meta,
  end,
  className,
}: {
  tile?: ReactNode;
  title: ReactNode;
  meta?: ReactNode;
  end?: ReactNode;
  className?: string;
}) {
  return (
    <span className={`rd-row-line${tile ? "" : " rd-row-line--flush"}${className ? ` ${className}` : ""}`}>
      {tile}
      <span className="rd-row-copy">
        <span className="rd-row-title">{title}</span>
        {meta ? <span className="rd-row-meta">{meta}</span> : null}
      </span>
      {end ? <span className="rd-row-end">{end}</span> : null}
    </span>
  );
}

export interface WeekDay {
  key: string;
  letter: string;
  date: number;
  count: number;
  today?: boolean;
  off?: boolean;
}

export function WeekStrip({ days, total, note }: { days: WeekDay[]; total?: number; note?: ReactNode }) {
  const { t } = useTranslation();
  const max = Math.max(1, ...days.map((day) => day.count));
  return (
    <div>
      <div className="rd-days" role="list" aria-label={t(strings.week.label)}>
        {days.map((day) => (
          <div
            className="rd-day"
            key={day.key}
            role="listitem"
            data-today={day.today ? "true" : undefined}
            data-off={day.off ? "true" : undefined}
            aria-label={t(strings.week.day, { letter: day.letter, date: day.date, count: day.count })}
          >
            <span aria-hidden="true">{day.letter}</span>
            <b aria-hidden="true">{day.date}</b>
            <i aria-hidden="true" />
          </div>
        ))}
      </div>
      <div className="rd-bars" aria-hidden="true">
        {days.map((day) => (
          <i
            key={day.key}
            data-today={day.today ? "true" : undefined}
            style={{ blockSize: `${Math.max(6, Math.round((day.count / max) * 100))}%` }}
          />
        ))}
      </div>
      <div className="rd-bars-foot">
        <span>{t(strings.week.writes)}</span>
        <span>{typeof total === "number" ? t(strings.week.total, { count: total }) : note}</span>
      </div>
    </div>
  );
}

export function MiniBars({ values }: { values: Array<{ count: number; today?: boolean }> }) {
  const max = Math.max(1, ...values.map((v) => v.count));
  return (
    <span className="rd-minibars" aria-hidden="true">
      {values.map((v, n) => (
        <i
          key={n}
          data-today={v.today ? "true" : undefined}
          style={{ blockSize: `${Math.max(18, Math.round((v.count / max) * 100))}%` }}
        />
      ))}
    </span>
  );
}
