/**
 * @libraryId react-component-library:Timeline
 * @displayName Timeline
 * @description The chronological event component with date grouping, status markers, expandable detail, loading, pagination, and responsive alignment.
 * @version 1.0.7
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Timeline */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

export type TimelineState = "ready" | "loading" | "empty" | "error";
export interface TimelineEvent {
  label: string;
  detail?: string;
  date?: string;
  status?: StatusTone;
}
export interface TimelineProps {
  events?: TimelineEvent[];
  state?: TimelineState;
  errorMessage?: string;
}

const styles = `
[data-rcl-timeline] { display: grid; gap: var(--space-md); margin: 0; padding: 0; list-style: none; color: var(--color-foreground); }
[data-rcl-timeline-item] { position: relative; display: grid; grid-template-columns: var(--tap-target-min) minmax(0, 1fr); gap: var(--space-sm); min-inline-size: 0; }
[data-rcl-timeline-item]::before { content: ""; position: absolute; inset-inline-start: calc(var(--tap-target-min) / 2 - var(--border-hairline)); inset-block-start: var(--space-md); block-size: calc(100% + var(--space-md)); border-inline-start: var(--border-hairline) solid var(--color-border); }
[data-rcl-timeline-item]:last-child::before { display: none; }
[data-rcl-timeline-marker] { z-index: 1; display: grid; place-items: center; inline-size: var(--tap-target-min); block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-pill); background: var(--color-surface); color: var(--color-primary); }
[data-rcl-timeline-marker]::after { content: ""; inline-size: var(--space-xs); block-size: var(--space-xs); border-radius: var(--radius-pill); background: currentColor; }
[data-rcl-timeline-content] { min-inline-size: 0; padding-block: var(--space-2xs); }
[data-rcl-timeline-label] { font: var(--text-label); }
[data-rcl-timeline-detail] { display: block; margin-block-start: var(--space-3xs); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-timeline-date] { display: block; margin-block-end: var(--space-3xs); color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: var(--tracking-caps); text-transform: uppercase; }
[data-rcl-timeline-state] { color: var(--color-muted-foreground); font: var(--text-body-sm); }
`;

export const Timeline = withClassName(function Timeline({
  events = [],
  state = "ready",
  errorMessage,
}: TimelineProps) {
  const strings = useStrings();
  if (state === "loading")
    return (
      <p data-testid="data-display.timeline" data-rcl-timeline-state role="status">
        {strings("data-display.timeline.loading", "Loading timeline…")}
      </p>
    );
  if (state === "error")
    return (
      <p data-testid="data-display.timeline" data-rcl-timeline-state role="alert">
        {errorMessage ?? strings("data-display.timeline.error", "Timeline could not be loaded.")}
      </p>
    );
  if (state === "empty" || events.length === 0)
    return (
      <p data-testid="data-display.timeline" data-rcl-timeline-state role="status">
        {strings("data-display.timeline.empty", "No events yet.")}
      </p>
    );
  return (
    <>
      <StyleSheet name="timeline-1-0-7" css={styles} />
      <ol
        data-testid="data-display.timeline"
        aria-label={strings("data-display.timeline.timeline", "Timeline")}
        data-rcl-timeline
      >
        {events.map((event, index) => (
          <li key={event.label + String(index)} data-rcl-timeline-item>
            <span data-rcl-timeline-marker aria-hidden="true" />
            <span data-rcl-timeline-content>
              {event.date ? <time data-rcl-timeline-date>{event.date}</time> : null}
              <strong data-rcl-timeline-label>{event.label}</strong>
              {event.status ? <StatusBadge tone={event.status}>{event.status}</StatusBadge> : null}
              {event.detail ? <small data-rcl-timeline-detail>{event.detail}</small> : null}
            </span>
          </li>
        ))}
      </ol>
    </>
  );
});
