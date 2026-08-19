/** @vrooliComponentSource react-component-library:VerdictSummary */
import { Badge } from "../../../../primitives/Badge/versions/1.0.0/Badge";
import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { Surface } from "../../../../primitives/Surface/versions/1.0.0/Surface";
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
    <Surface
      elevation="raised"
      className={SURFACE_ELEVATIONS.raised}
      aria-label="Verdict summary"
    >
      <Stack gap="xs">
        <Text as="strong" textStyle="label">
          Verdict census
        </Text>
        <div
          style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-xs)" }}
        >
          <Badge tone="success">{pass} pass</Badge>
          <Badge tone="danger">{fail} fail</Badge>
          <Badge tone="warning">{unmeasured} unmeasured</Badge>
        </div>
      </Stack>
    </Surface>
  );
}
