/**
 * @libraryId react-component-library:VerdictSummary
 * @displayName VerdictSummary
 * @description A compact pass, fail, and unmeasured verdict census.
 * @version 1.0.5
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:VerdictSummary */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { Badge } from "@vrooli/react-component-library/Badge/1";
import { Stack } from "@vrooli/react-component-library/Stack/1";
import { Text } from "@vrooli/react-component-library/Text/1";
import { SURFACE_ELEVATIONS } from "@vrooli/react-component-library/VisualRecipes/1";
export const VerdictSummary = withClassName(function VerdictSummary({
  pass = 0,
  fail = 0,
  unmeasured = 0,
}: {
  pass?: number;
  fail?: number;
  unmeasured?: number;
}) {
  const strings = useStrings();
  return (
    <section
      className={SURFACE_ELEVATIONS.raised}
      aria-label={strings("data-display.verdict-summary.verdict-summary", "Verdict summary")}
      data-rcl-asset="data-display.verdict-summary"
      data-rcl-version="1.0.3"
      data-rcl-stamp="source"
      data-testid="data-display-verdict-summary"
      style={{ boxShadow: "var(--elev-raised)", padding: "var(--space-xs)" }}
    >
      <Stack gap="xs">
        <Text as="strong" textStyle="label">
          {strings("data-display.verdict-summary.verdict-census", "Verdict census")}
        </Text>
        <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-xs)" }}>
          <Badge tone="success">{pass} pass</Badge>
          <Badge tone="danger">{fail} fail</Badge>
          <Badge tone="warning">{unmeasured} unmeasured</Badge>
        </div>
      </Stack>
    </section>
  );
});
