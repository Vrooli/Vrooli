/**
 * @libraryId react-component-library:AuditTrail
 * @displayName AuditTrail
 * @description An ordered activity surface that makes actor, action, and sequence legible.
 * @version 1.0.5
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:AuditTrail */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
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
