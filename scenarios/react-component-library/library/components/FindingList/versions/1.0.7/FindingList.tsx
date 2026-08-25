/** @vrooliComponentSource react-component-library:FindingList */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

export interface Finding {
  id: string;
  assetId?: string;
  severity?: string;
  message: string;
  remediation?: string;
}
import { Surface } from "../../../../primitives/Surface/versions/1.0.0/Surface";
import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { SURFACE_ELEVATIONS } from "../../../../foundations/VisualRecipes/versions/1.0.0/VisualRecipes";
export function FindingList({ findings = [] }: { findings?: Finding[] }) {
  return (
    <ul
      aria-label={translate("data-display.finding-list.aria-label.1", "Gate findings")}
      className={SURFACE_ELEVATIONS.raised}
      data-rcl-asset="data-display.finding-list"
      data-rcl-version="1.0.7"
      data-rcl-stamp="source"
      data-testid="data-display-finding-list"
      style={{
        display: "grid",
        gap: "var(--space-xs)",
        padding: 0,
        listStyle: "none",
        boxShadow: "var(--elev-raised)",
      }}
    >
      {findings.length ? (
        findings.map((finding) => (
          <Surface
            elevation="raised"
            key={finding.id}
            data-severity={finding.severity ?? "info"}
            className={SURFACE_ELEVATIONS.raised}
            style={{
              border: "1px solid var(--color-border)",
              borderRadius: "var(--radius-control)",
              padding: "var(--space-xs)",
            }}
          >
            <Stack gap="2xs">
              <strong>{finding.assetId ?? "Catalog"}</strong>
              <span> · {finding.severity ?? "info"}</span>
              <p>{finding.message}</p>
              {finding.remediation ? (
                <small>{finding.remediation}</small>
              ) : null}
            </Stack>
          </Surface>
        ))
      ) : (
        <li
          role="status"
          data-bespoke="empty result row preserves list semantics"
        >
          {translate("data-display.finding-list.text.1", "No findings.")}
        </li>
      )}
    </ul>
  );
}
