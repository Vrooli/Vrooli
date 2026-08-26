/**
 * @libraryId react-component-library:MeasurementBar
 * @displayName MeasurementBar
 * @description A labeled measured-versus-required bar for evidence panels.
 * @version 1.0.10
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

/** @vrooliComponentSource react-component-library:MeasurementBar */
import { BoundedMeter } from "../../../BoundedMeter/versions/1.0.1/BoundedMeter";
export interface MeasurementBarProps {
  label?: string;
  observed?: number;
  required?: number;
  unit?: string;
}
export const MeasurementBar = withClassName(function MeasurementBar({
  label = translate("data-display.measurement-bar.label.1", "Coverage"),
  observed = 72,
  required = 80,
  unit = "%",
}: MeasurementBarProps) {
  const max = Math.max(required, observed, 1);
  const status = observed >= required ? "pass" : "below-threshold";
  return (
    <BoundedMeter
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
