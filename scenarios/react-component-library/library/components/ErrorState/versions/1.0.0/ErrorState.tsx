/** @vrooliComponentSource react-component-library:ErrorState */
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function ErrorState({
  title = "Something went wrong",
  onRetry,
}: {
  title?: string;
  onRetry?: () => void;
}) {
  return (
    <div
      role="alert"
      style={{
        ...panel,
        display: "grid",
        gap: 8,
        borderColor: "var(--color-danger, #dc2626)",
      }}
    >
      <strong>{title}</strong>
      <span style={muted}>The operation could not be completed.</span>
      {onRetry && (
        <button type="button" onClick={onRetry}>
          Try again
        </button>
      )}
    </div>
  );
}
