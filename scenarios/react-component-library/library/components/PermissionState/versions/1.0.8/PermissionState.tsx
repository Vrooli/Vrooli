/**
 * @libraryId react-component-library:PermissionState
 * @displayName PermissionState
 * @description The authorization composition explaining that content is unavailable without disclosing why access is restricted, and offering request-access or navigation actions.
 * @version 1.0.8
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:PermissionState */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 0.5rem)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export const PermissionState = withClassName(function PermissionState({
  action,
  title,
  description,
  actionLabel,
}: {
  action?: () => void;
  title?: string;
  description?: string;
  actionLabel?: string;
}) {
  const strings = useStrings();
  return (
    <div role="status" style={{ ...panel, display: "grid", gap: 10 }}>
      <strong>
        {title ?? strings("feedback.permission-state.permission-required", "Permission required")}
      </strong>
      <span style={muted}>
        {description ??
          strings(
            "feedback.permission-state.request-access-to-continue",
            "Request access to continue.",
          )}
      </span>
      {action && (
        <button data-testid="feedback.permission-state" type="button" onClick={action}>
          {actionLabel ?? strings("feedback.permission-state.request-access", "Request access")}
        </button>
      )}
    </div>
  );
});
