/** @vrooliComponentSource react-component-library:JsonViewer */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
export function JsonViewer({ value = {} }: { value?: unknown }) {
  return (
    <pre
      aria-label={translate("data-display.json-viewer.aria-label.1", "JSON value")}
      style={{
        ...panel,
        overflow: "auto",
        background: "var(--color-surface-muted, #f1f5f9)",
        fontFamily: "var(--font-mono, ui-monospace)",
        lineHeight: 1.65,
        whiteSpace: "pre-wrap",
      }}
    >
      {JSON.stringify(value, null, 2)}
    </pre>
  );
}
