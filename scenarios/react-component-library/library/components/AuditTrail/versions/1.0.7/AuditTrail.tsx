/**
 * @libraryId react-component-library:AuditTrail
 * @displayName AuditTrail
 * @version 1.0.7
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:AuditTrail */
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
const muted = { color: "var(--color-muted-foreground)" };
export const AuditTrail = withClassName(function AuditTrail({
  entries = [],
}: {
  entries?: Array<{ actor: string; action: string }>;
}) {
  const strings = useStrings();
  return (
    <div
      data-testid="data-display.audit-trail"
      aria-label={strings("data-display.audit-trail.audit-trail", "Audit trail")}
      role="list"
      style={{
        display: "grid",
        gap: 8,
        listStyle: "none",
        margin: 0,
        padding: 0,
      }}
    >
      {entries.map((entry, index) => (
        <div key={entry.actor + String(index)} role="listitem" style={panel}>
          <strong>{entry.actor}</strong>
          <span style={{ display: "block", marginTop: 4, ...muted }}>{entry.action}</span>
        </div>
      ))}
    </div>
  );
});
