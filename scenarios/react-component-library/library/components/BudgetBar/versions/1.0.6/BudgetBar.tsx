/**
 * @libraryId react-component-library:BudgetBar
 * @displayName BudgetBar
 * @description A runtime budget bar for mount and rerender cost.
 * @version 1.0.6
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

/** @vrooliComponentSource react-component-library:BudgetBar */
import { BoundedMeter } from "../../../BoundedMeter/versions/1.0.1/BoundedMeter";
export function BudgetBar({
  label = translate("data-display.budget-bar.label.1", "Mount"),
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
    <BoundedMeter
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
}