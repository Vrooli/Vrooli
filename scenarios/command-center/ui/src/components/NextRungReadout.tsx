import { forwardRef } from "react";
import type { Reading } from "../lib/api";
import { InkMark, qualify, resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import { finishLabel, ladderOf, nextRung, rungsAfterNext, statusLabel, type LadderRung, type LadderWork } from "../lib/ladder";
import { AutoScroll } from "./AutoScroll";

const THEN_COUNT = 3;
/** Below the head, each of these is a row the details step through when the hero is too short to hold them all. */
const DETAIL_ROWS = ".cc-next-rung__blockers h3, .cc-next-rung__blocker, .cc-next-rung__clear, .cc-next-rung__facts, .cc-next-rung__then, .cc-ladder-gap";

const why = (work: LadderWork): string => (work.direct ? "direct enabler" : `due by rank ${work.urgency}`);

function Opens({ rung }: { rung: LadderRung }) {
  const tokens = [...rung.ramps.map((name) => ({ kind: "ramp", name })), ...rung.streams.map((name) => ({ kind: "stream", name }))];
  return (
    <p className="cc-next-rung__opens">
      {tokens.length ? <>Opens {tokens.map((token) => <span key={`${token.kind}:${token.name}`} className="cc-ladder-token" data-kind={token.kind} title={token.kind}>{token.name}</span>)}</> : "Opens no new ramp or stream"}
      {rung.audiences.length ? <> for <span className="cc-next-rung__audience">{rung.audiences.join(" · ")}</span></> : null}
    </p>
  );
}

/**
 * The release ladder in the hero column: the lowest unshipped rung, what it
 * opens, and the enabling work still in its way. The rest of the schedule is
 * drawn by the funnel-cascade scene in the band beside it. The head always
 * shows; the details below it auto-scroll on a screen too short to hold them.
 */
export const NextRungReadout = forwardRef<HTMLDivElement, { reading: Reading }>(function NextRungReadout({ reading }, ref) {
  const resolution = resolveReading(reading);
  const qualifier = qualify(reading, resolution);
  const resolved = ladderOf(reading);
  const rung = resolved ? nextRung(resolved.ladder) : null;
  const after = resolved ? rungsAfterNext(resolved.ladder) : [];
  const blockers = rung?.blockers ?? [];
  const gaps = (
    <>
      {resolved?.ladder.unscheduled.length ? <p className="cc-ladder-gap">Unscheduled: {resolved.ladder.unscheduled.map((node) => node.name).join(", ")} · marketed, no rank</p> : null}
      {resolved?.ladder.unavailable?.map((gap) => <p key={gap.source} className="cc-ladder-gap">{gap.source} unavailable · {gap.reason}</p>)}
    </>
  );
  return (
    <div ref={ref} className="cc-ladder cc-next-rung" data-testid="ladder-next-rung" data-kind="ladder" data-view="next-rung" data-reading data-metric-id={reading.id} data-ink={resolution.ink} data-provenance={resolved?.sampled ? "sample" : resolution.figure === "measured" ? "measured" : "absent"} data-trust={reading.trust}>
      <span className="cc-hero-label">{reading.label}</span>
      {!resolved ? (
        <p className="cc-next-rung__none">No schedule to show.</p>
      ) : !rung ? (
        <p className="cc-next-rung__none">Every scheduled release has shipped.</p>
      ) : (
        <>
          <div className="cc-next-rung__head">
            <span className="cc-next-rung__rank" aria-label={`Rank ${rung.rank}`}>{rung.rank}</span>
            <div className="cc-next-rung__who">
              <span className="cc-next-rung__kicker">Next release{rung.finishBar ? ` · ${finishLabel(rung.finishBar)}` : ""}</span>
              <span className="cc-next-rung__name">{rung.name}</span>
              <span className="cc-ladder-status" data-status={rung.status}><span className="cc-ladder-dot" data-status={rung.status} aria-hidden="true" />{statusLabel(rung.status)}</span>
            </div>
          </div>
          <Opens rung={rung} />
          <AutoScroll className="cc-next-rung__details" rowSelector={DETAIL_ROWS} counter={false} label={`Rank ${rung.rank} details`}>
            <section className="cc-next-rung__blockers" aria-label={`Enabling work due by rank ${rung.rank}`}>
              <h3><span>Enabling work due by rank {rung.rank}</span><span>{blockers.length} open</span></h3>
              {blockers.length ? (
                <ul>
                  {blockers.map((work) => (
                    <li key={work.name} className="cc-next-rung__blocker">
                      <span className="cc-ladder-dot" data-status={work.status} aria-hidden="true" />
                      <span className="cc-next-rung__work">{work.name}</span>
                      <span className="cc-next-rung__why">{statusLabel(work.status)} · {why(work)}</span>
                    </li>
                  ))}
                </ul>
              ) : <p className="cc-next-rung__clear">Nothing enabling is still open.</p>}
            </section>
            <dl className="cc-next-rung__facts">
              <div>
                <dt>Goals riding on it</dt>
                <dd>{rung.goals.length ? rung.goals.map((goal) => goal.title || goal.name).join(" · ") : "none linked"}</dd>
              </div>
              <div>
                <dt>Readiness</dt>
                <dd data-gap={rung.readiness.reported ? undefined : true}>{rung.readiness.reported ? (rung.readiness.goalClosed ? "readiness goal closed" : "readiness goal open") + (rung.readiness.approvedCommit ? ` · ${rung.readiness.approvedCommit.slice(0, 8)}` : "") : "not reported"}</dd>
              </div>
            </dl>
            {after.length ? (
              <p className="cc-next-rung__then">Then {after.slice(0, THEN_COUNT).map((next, index) => <span key={next.id}>{index ? " · " : ""}<b>{next.rank}</b> {next.name}</span>)}{after.length > THEN_COUNT ? ` · +${after.length - THEN_COUNT} more` : ""}</p>
            ) : null}
            {gaps}
          </AutoScroll>
        </>
      )}
      {rung ? null : gaps}
      <span className="cc-qualifier" data-qualifier data-tone={qualifier.tone}>
        {resolved?.sampled && resolution.ink !== "none" ? <InkMark ink={resolution.ink}>illustrative</InkMark> : null}
        {qualifier.text}
      </span>
    </div>
  );
});
