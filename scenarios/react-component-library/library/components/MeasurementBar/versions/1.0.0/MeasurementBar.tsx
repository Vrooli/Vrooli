/** @vrooliComponentSource react-component-library:MeasurementBar */
import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { Surface } from "../../../../primitives/Surface/versions/1.0.0/Surface";
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
    <Surface
      elevation="raised"
      className={SURFACE_ELEVATIONS.raised}
      aria-label={`${label} measurement`}
      data-status={status}
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
        />
        <Text tone="muted" numeric>
          {observed}
          {unit} observed · {required}
          {unit} required
        </Text>
      </Stack>
    </Surface>
  );
}
