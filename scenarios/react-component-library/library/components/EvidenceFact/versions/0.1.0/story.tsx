import { EvidenceFact } from "./EvidenceFact";

export function Default() {
  return (
    <dl style={{ display: "grid", gap: "var(--space-sm)", maxInlineSize: "42rem", margin: 0 }}>
      <EvidenceFact
        label="Last research write"
        value="12 minutes ago"
        disclosure={{
          label: "Why?",
          expandedLabel: "Hide evidence",
          content: (
            <ul>
              <li>research/topics/index.json · updated 12 minutes ago</li>
            </ul>
          ),
        }}
      />
      <EvidenceFact label="Launch mode" value="Unknown · no current configuration" />
    </dl>
  );
}
