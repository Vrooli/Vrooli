/**
 * @libraryId react-component-library:RelativeTime
 * @displayName RelativeTime
 * @description The locale-aware timestamp component updating at meaningful intervals and exposing the precise absolute time through accessible and hoverable detail.
 * @version 1.0.5
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:RelativeTime */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

const muted = { color: "var(--color-muted-foreground)" };
export const RelativeTime = withClassName(function RelativeTime({
  value = "just now",
  dateTime,
  title,
}: {
  value?: string;
  /** Machine-readable timestamp, separate from the localized visible value. */
  dateTime?: string;
  /** Precise, human-readable timestamp for native hover and assistive detail. */
  title?: string;
}) {
  return (
    <time
      data-testid="primitives.relative-time"
      dateTime={dateTime ?? value}
      title={title}
      style={muted}
    >
      {value}
    </time>
  );
});
