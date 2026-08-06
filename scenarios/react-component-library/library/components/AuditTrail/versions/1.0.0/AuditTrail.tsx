/** @vrooliComponentSource react-component-library:AuditTrail */
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function AuditTrail({
  entries = [],
}: {
  entries?: Array<{ actor: string; action: string }>;
}) {
  return (
    <ol
      aria-label="Audit trail"
      style={{
        display: "grid",
        gap: 8,
        listStyle: "none",
        margin: 0,
        padding: 0,
      }}
    >
      {entries.map((entry, index) => (
        <li key={entry.actor + String(index)} style={panel}>
          <strong>{entry.actor}</strong>
          <span style={{ display: "block", marginTop: 4, ...muted }}>
            {entry.action}
          </span>
        </li>
      ))}
    </ol>
  );
}
