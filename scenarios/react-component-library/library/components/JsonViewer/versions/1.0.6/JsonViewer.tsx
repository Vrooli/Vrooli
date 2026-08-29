/**
 * @libraryId react-component-library:JsonViewer
 * @displayName JsonViewer
 * @description A bounded structured-data viewer with readable overflow and machine-honest semantics.
 * @version 1.0.6
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:JsonViewer */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.1.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";

const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 0.5rem)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10))",
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
        background: "var(--color-surface-muted, #f1f5f9)",
        fontFamily:
          'var(--font-mono, "JetBrains Mono", "Fira Code", "SF Mono", Consolas, "Liberation Mono", Menlo, monospace)',
        lineHeight: 1.65,
        whiteSpace: "pre-wrap",
      }}
    >
      {JSON.stringify(value, null, 2)}
    </pre>
  );
});
