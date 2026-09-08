/**
 * @libraryId react-component-library:JsonViewer
 * @displayName JsonViewer
 * @version 1.0.7
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:JsonViewer */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

const panel = {
  border: "1px solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface)",
  color: "var(--color-foreground)",
  padding: "var(--space-md)",
  boxShadow: "var(--elev-raised)",
};
export const JsonViewer = withClassName(function JsonViewer({ value = {} }: { value?: unknown }) {
  const strings = useStrings();
  return (
    <pre
      data-testid="data-display.json-viewer"
      aria-label={strings("data-display.json-viewer.json-value", "JSON value")}
      style={{
        ...panel,
        overflow: "auto",
        background: "var(--color-surface-muted)",
        fontFamily:
          'var(--font-mono)',
        lineHeight: 1.65,
        whiteSpace: "pre-wrap",
      }}
    >
      {JSON.stringify(value, null, 2)}
    </pre>
  );
});
