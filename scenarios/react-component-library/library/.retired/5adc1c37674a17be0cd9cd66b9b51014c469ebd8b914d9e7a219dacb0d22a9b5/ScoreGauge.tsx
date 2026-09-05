/**
 * @libraryId react-component-library:ScoreGauge
 * @displayName ScoreGauge
 * @description
 * @version 1.0.9
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:ScoreGauge */
import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1";

export interface ScoreGaugeProps {
  value?: number;
  label?: string;
  threshold?: number;
}
export const ScoreGauge = withClassName(function ScoreGauge({
  value = 0,
  label,
  threshold = 90,
}: ScoreGaugeProps) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("data-display.score-gauge.score", "Score");
  const bounded = Math.max(0, Math.min(100, value));
  const status = bounded >= threshold ? "passing" : bounded >= threshold / 2 ? "watch" : "blocked";
  return (
    <BoundedMeter
      data-testid="data-display.score-gauge"
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
