/**
 * @libraryId react-component-library:RelativeTime
 * @displayName RelativeTime
 * @description A semantic time value that presents a concise relative label while preserving machine-readable date context.
 * @version 1.0.3
 * @tags ["primitive","data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:RelativeTime */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

const muted = { color: "var(--color-muted-foreground, #64748b)" };
export const RelativeTime = withClassName(function RelativeTime({
  value = "just now",
}: {
  value?: string;
}) {
  return (
    <time data-testid="primitives.relative-time" dateTime={value} style={muted}>
      {value}
    </time>
  );
});
