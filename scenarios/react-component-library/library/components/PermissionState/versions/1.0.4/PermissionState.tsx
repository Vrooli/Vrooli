/**
 * @libraryId react-component-library:PermissionState
 * @displayName PermissionState
 * @description A permission-aware state that gives the user a clear next action without exposing unauthorized content.
 * @version 1.0.4
 * @tags ["feedback","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:PermissionState */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export const PermissionState = withClassName(function PermissionState({
  action,
}: {
  action?: () => void;
}) {
  return (
    <div role="status" style={{ ...panel, display: "grid", gap: 10 }}>
      <strong>{translate("feedback.permission-state.text.1", "Permission required")}</strong>
      <span style={muted}>
        {translate("feedback.permission-state.text.2", "Request access to continue.")}
      </span>
      {action && (
        <button data-testid="feedback.permission-state" type="button" onClick={action}>
          {translate("feedback.permission-state.text.3", "Request access")}
        </button>
      )}
    </div>
  );
});
