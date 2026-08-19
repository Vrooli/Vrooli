/** @vrooliComponentSource react-component-library:BudgetBar */
import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { Text } from "../../../../primitives/Text/versions/1.0.0/Text";
import { SURFACE_ELEVATIONS } from "../../../../foundations/VisualRecipes/versions/1.0.0/VisualRecipes";
export function BudgetBar({
  label = "Mount",
  value = 42,
  budget = 100,
  unit = "ms",
}: {
  label?: string;
  value?: number;
  budget?: number;
  unit?: string;
}) {
  const ratio = Math.max(0, Math.min(1, value / Math.max(budget, 1)));
  return (
    <section
      className={SURFACE_ELEVATIONS.raised}
      aria-label={`${label} budget`}
      data-status={ratio <= 1 ? "within-budget" : "over-budget"}
      data-rcl-asset="data-display.budget-bar"
      data-rcl-version="1.0.2"
      data-rcl-stamp="source"
      style={{ boxShadow: "var(--elev-raised)", padding: "var(--space-xs)" }}
    >
      <Stack gap="2xs">
        <Text as="strong" textStyle="label">
          {label}
        </Text>
        <meter
          min={0}
          max={budget}
          value={Math.min(value, budget)}
          aria-label={`${label} cost`}
        />
        <Text tone="muted" numeric>
          {value}
          {unit} / {budget}
          {unit} budget
        </Text>
      </Stack>
    </section>
  );
}
