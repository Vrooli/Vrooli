/**
 * @libraryId react-component-library:Timeline
 * @displayName Timeline
 * @description A chronological surface that preserves event order and readable detail in dense histories.
 * @version 1.0.5
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Timeline */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

const muted = { color: "var(--color-muted-foreground, #64748b)" };
export const Timeline = withClassName(function Timeline({
  events = [],
}: {
  events?: Array<{ label: string; detail?: string }>;
}) {
  const strings = useStrings();
  return (
    <ol
      data-testid="data-display.timeline"
      aria-label={strings("data-display.timeline.timeline", "Timeline")}
      style={{
        display: "grid",
        gap: 16,
        listStyle: "none",
        margin: 0,
        padding: 0,
      }}
    >
      {events.map((event, index) => (
        <li
          key={event.label + String(index)}
          style={{ display: "grid", gridTemplateColumns: "16px 1fr", gap: 12 }}
        >
          <span
            aria-hidden
            style={{
              width: 10,
              height: 10,
              marginTop: 5,
              borderRadius: "50%",
              background: "var(--color-primary, #2563eb)",
            }}
          />
          <span>
            <strong>{event.label}</strong>
            {event.detail && (
              <small style={{ display: "block", marginTop: 4, ...muted }}>{event.detail}</small>
            )}
          </span>
        </li>
      ))}
    </ol>
  );
});
