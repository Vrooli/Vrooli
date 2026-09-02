import type { Reading } from "../lib/api";
import { figureValue, isIllustrative, qualify, resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.1";
import { RollingNumber } from "@vrooli/react-component-library/RollingNumber/0.1.2";
import { FreshnessArc } from "@vrooli/react-component-library/FreshnessArc/0.1.1";
import { SampleSeries } from "@vrooli/react-component-library/SampleSeries/0.1.1";

/** A supporting reading: label, figure in its ink, the rule or series beneath it, and its qualifier. */
export function ReadingTile({ reading }: { reading: Reading }) {
  const resolution = resolveReading(reading);
  const value = figureValue(reading, resolution);
  const qualifier = qualify(reading, resolution);
  const live = resolution.ink === "solid" || resolution.ink === "dimmed";
  const illustrative = isIllustrative(resolution);
  return (
    <li className="cc-reading" data-reading data-metric-id={reading.id} data-ink={resolution.ink} data-coverage={reading.coverage} data-trust={reading.trust} data-empirical={reading.empirical} data-provenance={illustrative ? "sample" : resolution.figure === "measured" ? "measured" : "absent"} title={reading.description}>
      <span className="cc-reading-label">{reading.label}</span>
      <RollingNumber value={value} format={reading.format} unit={reading.unit} ink={resolution.ink} scale="display" placeholder={resolution.ink === "unavailable" ? "––" : "—"} />
      {live ? <FreshnessArc observedAt={reading.observedAt} ttlSeconds={reading.ttlSeconds} cached={resolution.ink === "dimmed"} /> : null}
      {illustrative && reading.sample ? <SampleSeries series={reading.sample.series} illustrative /> : null}
      <span className="cc-qualifier" data-qualifier data-tone={qualifier.tone}>{qualifier.text}</span>
    </li>
  );
}
