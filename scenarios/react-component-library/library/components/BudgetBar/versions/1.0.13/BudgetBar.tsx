/**
 * @libraryId react-component-library:BudgetBar
 * @displayName BudgetBar
 * @description
 * @version 1.0.13
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource react-component-library:BudgetBar */
import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1.0.2";
export const BudgetBar = withClassName(function BudgetBar({
  label,
  value = 42,
  budget = 100,
  unit = "ms",
}: {
  label?: string;
  value?: number;
  budget?: number;
  unit?: string;
}) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("data-display.budget-bar.mount", "Mount");
  const ratio = Math.max(0, Math.min(1, value / Math.max(budget, 1)));
  return (
    <BoundedMeter
      data-testid="data-display.budget-bar"
      label={label}
      value={value}
      max={budget}
      valueText={`${value}${unit}`}
      description={`${value}${unit} / ${budget}${unit} budget`}
      data-status={ratio <= 1 ? "within-budget" : "over-budget"}
      tone={ratio <= 1 ? "success" : "danger"}
      ariaLabel={`${label} budget`}
      meterLabel={`${label} cost`}
      assetId="data-display.budget-bar"
      assetVersion="1.0.6"
      testId="data-display-budget-bar"
    />
  );
});
