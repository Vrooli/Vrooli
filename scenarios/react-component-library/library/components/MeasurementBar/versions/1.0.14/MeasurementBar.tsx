/**
 * @libraryId react-component-library:MeasurementBar
 * @displayName MeasurementBar
 * @version 1.0.14
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:MeasurementBar */
import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1";
export interface MeasurementBarProps {
  label?: string;
  observed?: number;
  required?: number;
  unit?: string;
}
export const MeasurementBar = withClassName(function MeasurementBar({
  label,
  observed = 72,
  required = 80,
  unit = "%",
}: MeasurementBarProps) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("data-display.measurement-bar.coverage", "Coverage");
  const max = Math.max(required, observed, 1);
  const status = observed >= required ? "pass" : "below-threshold";
  return (
    <BoundedMeter
      data-testid="data-display.measurement-bar"
      label={label}
      value={observed}
      max={max}
      valueText={`${observed}${unit}`}
      description={`${observed}${unit} observed · ${required}${unit} required`}
      data-status={status}
      tone={status === "pass" ? "success" : "warning"}
      ariaLabel={`${label} measurement`}
      meterLabel={`${label} observed`}
      assetId="data-display.measurement-bar"
      assetVersion="1.0.6"
      testId="data-display-measurement-bar"
    />
  );
});
