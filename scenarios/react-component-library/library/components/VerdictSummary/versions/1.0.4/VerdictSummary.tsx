/**
 * @libraryId react-component-library:VerdictSummary
 * @displayName VerdictSummary
 * @description A compact pass, fail, and unmeasured verdict census.
 * @version 1.0.4
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:VerdictSummary */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { Badge } from "../../../../primitives/Badge/versions/1.0.0/Badge";
import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { Text } from "../../../../primitives/Text/versions/1.0.0/Text";
import { SURFACE_ELEVATIONS } from "../../../../foundations/VisualRecipes/versions/1.0.0/VisualRecipes";
export function VerdictSummary({
  pass = 0,
  fail = 0,
  unmeasured = 0,
}: {
  pass?: number;
  fail?: number;
  unmeasured?: number;
}) {
  return (
    <section
      className={SURFACE_ELEVATIONS.raised}
      aria-label={translate("data-display.verdict-summary.aria-label.1", "Verdict summary")}
      data-rcl-asset="data-display.verdict-summary"
      data-rcl-version="1.0.3"
      data-rcl-stamp="source"
      data-testid="data-display-verdict-summary"
      style={{ boxShadow: "var(--elev-raised)", padding: "var(--space-xs)" }}
    >
      <Stack gap="xs">
        <Text as="strong" textStyle="label">
          {translate("data-display.verdict-summary.text.1", "Verdict census")}
        </Text>
        <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-xs)" }}>
          <Badge tone="success">{pass} pass</Badge>
          <Badge tone="danger">{fail} fail</Badge>
          <Badge tone="warning">{unmeasured} unmeasured</Badge>
        </div>
      </Stack>
    </section>
  );
}
