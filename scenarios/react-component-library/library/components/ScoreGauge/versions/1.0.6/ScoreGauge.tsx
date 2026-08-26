/**
 * @libraryId react-component-library:ScoreGauge
 * @displayName ScoreGauge
 * @description A bounded score summary with an explicit threshold and accessible value.
 * @version 1.0.6
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

/** @vrooliComponentSource react-component-library:ScoreGauge */
import { BoundedMeter } from "../../../BoundedMeter/versions/1.0.1/BoundedMeter";

export interface ScoreGaugeProps {
  value?: number;
  label?: string;
  threshold?: number;
}
export const ScoreGauge = withClassName(function ScoreGauge({
  value = 0,
  label = translate("data-display.score-gauge.label.1", "Score"),
  threshold = 90,
}: ScoreGaugeProps) {
  const bounded = Math.max(0, Math.min(100, value));
  const status = bounded >= threshold ? "passing" : bounded >= threshold / 2 ? "watch" : "blocked";
  return (
    <BoundedMeter
      label={label}
      value={bounded}
      max={100}
      valueText={`${bounded.toFixed(0)}%`}
      description={`${status} · threshold ${threshold}%`}
      status={status}
      tone={status === "passing" ? "success" : status === "watch" ? "warning" : "danger"}
      ariaLabel={label}
      meterLabel={`${label} value`}
      assetId="data-display.score-gauge"
      assetVersion="1.0.2"
      testId="data-display-score-gauge"
    />
  );
});
