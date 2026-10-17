type Calibration = { avgErrorPercent: number; sampleSize: number; accuracyTrend: number[] };

export function CalibrationCard({ calibration, isLoading }: { calibration?: Calibration; isLoading: boolean }) {
  if (isLoading) return <section className="calibration-card" role="status"><span className="card-kicker">CALIBRATION</span><h2>Reading the instrument…</h2></section>;
  if (!calibration || calibration.sampleSize === 0) return <section className="calibration-card"><span className="card-kicker">CALIBRATION</span><h2>Your instrument is waiting for a first reading.</h2><p>Record an actual against a planned block. Review will compare the two from stored evidence.</p></section>;
  const percent = Math.round(Math.abs(calibration.avgErrorPercent));
  const trend = Array.isArray(calibration.accuracyTrend) && calibration.accuracyTrend.length > 0 ? calibration.accuracyTrend : [calibration.avgErrorPercent];
  const minimum = Math.min(...trend);
  const maximum = Math.max(...trend);
  const points = trend.map((value, index) => `${trend.length === 1 ? 50 : (index / (trend.length - 1)) * 100},${88 - ((value - minimum) / Math.max(1, maximum - minimum)) * 76}`).join(" ");
  return <section className="calibration-card"><span className="card-kicker">CALIBRATION · STORED EVIDENCE</span><h2>{calibration.avgErrorPercent > 0 ? `You are averaging ${percent}% over the accepted estimate.` : calibration.avgErrorPercent < 0 ? `You are averaging ${percent}% under the accepted estimate.` : "Your accepted estimates are matching actual time."}</h2><p>{calibration.avgErrorPercent > 0 ? "A warm signal to protect a little more room for this kind of work." : calibration.avgErrorPercent < 0 ? "A positive signal — your recent plans leave useful room." : "Keep collecting readings; patterns become more useful with repetition."}</p><div className="calibration-sparkline" role="img" aria-label={`Stored calibration trend with ${calibration.sampleSize} observations`}><svg viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true"><polyline points={points} /></svg></div><small>{calibration.sampleSize} linked plan-to-actual readings</small></section>;
}
