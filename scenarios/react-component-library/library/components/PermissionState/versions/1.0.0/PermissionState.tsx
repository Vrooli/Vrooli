/** @vrooliComponentSource react-component-library:PermissionState */
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
      <strong>Permission required</strong>
      <span style={muted}>Request access to continue.</span>
      {action && (
        <button type="button" onClick={action}>
          Request access
        </button>
      )}
    </div>
  );
}
