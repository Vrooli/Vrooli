/** @vrooliComponentSource react-component-library:FindingList */
export interface Finding {
  id: string;
  assetId?: string;
  severity?: string;
  message: string;
  remediation?: string;
}
export function FindingList({ findings = [] }: { findings?: Finding[] }) {
  if (!findings.length) return <p role="status">No findings.</p>;
  return (
    <ul
      aria-label="Gate findings"
      style={{
        display: "grid",
        gap: "var(--space-xs)",
        padding: 0,
        listStyle: "none",
      }}
    >
      {findings.map((finding) => (
        <li
          key={finding.id}
          data-severity={finding.severity ?? "info"}
          style={{
            border: "1px solid var(--color-border)",
            borderRadius: "var(--radius-control)",
            padding: "var(--space-xs)",
          }}
        >
          <strong>{finding.assetId ?? "Catalog"}</strong>
          <span> · {finding.severity ?? "info"}</span>
          <p>{finding.message}</p>
          {finding.remediation ? <small>{finding.remediation}</small> : null}
        </li>
      ))}
    </ul>
  );
}
