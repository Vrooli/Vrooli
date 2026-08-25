/** @vrooliComponentSource react-component-library:PermissionState */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function PermissionState({ action }: { action?: () => void }) {
  return (
    <div role="status" style={{ ...panel, display: "grid", gap: 10 }}>
      <strong>{translate("feedback.permission-state.text.1", "Permission required")}</strong>
      <span style={muted}>{translate("feedback.permission-state.text.2", "Request access to continue.")}</span>
      {action && (
        <button data-testid="feedback.permission-state" type="button" onClick={action}>
          {translate("feedback.permission-state.text.3", "Request access")}
        </button>
      )}
    </div>
  );
}
