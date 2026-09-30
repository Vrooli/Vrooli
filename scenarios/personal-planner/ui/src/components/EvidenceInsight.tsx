export function EvidenceInsight({ planned, recorded }: { planned: number; recorded: number }) {
  const difference = recorded - planned;
  const message = planned === 0 && recorded === 0
    ? "There is not enough recorded activity to compare a plan with what happened."
    : difference === 0
      ? "Recorded active time matches the accepted plan for this period. That is alignment evidence, not a prediction."
      : difference > 0
        ? `${difference} more minutes were recorded than were accepted on the plan. This may reflect useful work outside the schedule; the source is not inferred.`
        : `${Math.abs(difference)} accepted minutes were not represented in recorded active time. That may be unfinished work or missing capture; the review does not choose between them.`;
  return <section className="review-insight-card" aria-label="Evidence-linked observation"><span className="card-kicker">OBSERVATION · EVIDENCE ONLY</span><p>{message}</p><small>Compared: {planned} planned minutes · {recorded} recorded active minutes.</small></section>;
}
