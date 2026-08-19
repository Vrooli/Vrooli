/**
 * @libraryId react-component-library:MeasurementBar
 * @displayName MeasurementBar
 * @description A labeled measured-versus-required bar for evidence panels.
 * @version 1.0.5
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:MeasurementBar */
import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { Text } from "../../../../primitives/Text/versions/1.0.0/Text";
import { SURFACE_ELEVATIONS } from "../../../../foundations/VisualRecipes/versions/1.0.0/VisualRecipes";
export interface MeasurementBarProps {
  label?: string;
  observed?: number;
  required?: number;
  unit?: string;
}
export function MeasurementBar({
  label = "Coverage",
  observed = 72,
  required = 80,
  unit = "%",
}: MeasurementBarProps) {
  const max = Math.max(required, observed, 1);
  const status = observed >= required ? "pass" : "below-threshold";
  return (
    <section
      className={SURFACE_ELEVATIONS.raised}
      aria-label={`${label} measurement`}
      data-status={status}
      data-rcl-asset="data-display.measurement-bar"
      data-rcl-version="1.0.4"
      data-rcl-stamp="source"
      data-testid="data-display-measurement-bar"
      style={{ boxShadow: "var(--elev-raised)", padding: "var(--space-xs)" }}
    >
      <Stack gap="2xs">
        <Text as="strong" textStyle="label">
          {label}
        </Text>
        <meter
          min={0}
          max={max}
          value={observed}
          aria-label={`${label} observed`}
          data-bespoke="native meter preserves threshold semantics"
        />
        <Text tone="muted" numeric>
          {observed}
          {unit} observed · {required}
          {unit} required
        </Text>
      </Stack>
    </section>
  );
}
