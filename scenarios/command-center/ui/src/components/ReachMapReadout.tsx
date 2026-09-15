import { forwardRef, type CSSProperties } from "react";
import type { Reading } from "../lib/api";
import { InkMark, qualify, resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import { lastOpening, ladderOf, openedBy, statusLabel, type LadderReading, type UnlockKind } from "../lib/ladder";

const GROUPS: Array<{ kind: UnlockKind; title: string }> = [
  { kind: "ramp", title: "Ramps · how people get it" },
  { kind: "stream", title: "Streams · how it earns" },
  { kind: "audience", title: "Audiences · who it serves" },
];

/** The rank a glance summarises to: five rungs into the schedule, or its end. */
const summaryRank = (ladder: LadderReading): number => ladder.rungs[Math.min(4, ladder.rungs.length - 1)]?.rank ?? 0;

/**
 * The schedule as an axis: each ramp, stream and audience lights up at the
 * rung that opens it. Rendered by a wide beat, so it spans both figure columns.
 */
export const ReachMapReadout = forwardRef<HTMLDivElement, { reading: Reading }>(function ReachMapReadout({ reading }, ref) {
  const resolution = resolveReading(reading);
  const qualifier = qualify(reading, resolution);
  const resolved = ladderOf(reading);
  const ladder = resolved?.ladder;
  const rungIndex = new Map(ladder?.rungs.map((rung, index) => [rung.rank, index]) ?? []);
  const nameAt = new Map(ladder?.rungs.map((rung) => [rung.rank, rung.name]) ?? []);
  const at = ladder ? summaryRank(ladder) : 0;
  const opened = ladder ? openedBy(ladder, at) : null;
  const last = ladder ? lastOpening(ladder) : [];
  const goalCount = ladder?.rungs.reduce((sum, rung) => sum + rung.goals.length, 0) ?? 0;
  return (
    <div ref={ref} className="cc-ladder cc-reach" data-testid="ladder-reach" data-kind="ladder" data-view="reach" data-reading data-metric-id={reading.id} data-ink={resolution.ink} data-provenance={resolved?.sampled ? "sample" : resolution.figure === "measured" ? "measured" : "absent"} data-trust={reading.trust}>
      <div className="cc-reach__head">
        <span className="cc-hero-label">What each release opens</span>
        {ladder && opened ? (
          <p className="cc-reach__summary">
            By rank {at}: <span>{opened.ramp.open}/{opened.ramp.total} ramps · {opened.stream.open}/{opened.stream.total} streams · {opened.audience.open}/{opened.audience.total} audiences</span>
            {last[0] ? <><br />Last to open: <span>{last.map((unlock) => unlock.name).join(", ")} at rank {last[0].opensAt}</span></> : null}
          </p>
        ) : null}
      </div>
      {!ladder ? <p className="cc-next-rung__none">No schedule to show.</p> : (
        <div className="cc-reach__scroll">
          <div className="cc-reach__grid" style={{ "--rungs": ladder.rungs.length } as CSSProperties}>
            <div className="cc-reach__cols" aria-hidden="true">
              <span />
              {ladder.rungs.map((rung) => (
                <span key={rung.id} className="cc-reach__col" data-next={rung.rank === ladder.nextRank || undefined} title={`${rung.rank} ${rung.name} · ${statusLabel(rung.status)}`}>
                  <span className="cc-reach__name">{rung.name}</span>
                  <span className="cc-ladder-dot" data-status={rung.status} />
                  <span className="cc-reach__rank">{rung.rank}</span>
                </span>
              ))}
            </div>
            {GROUPS.map((group) => {
              const lanes = ladder.reach.filter((unlock) => unlock.kind === group.kind);
              if (!lanes.length) return null;
              return (
                <section key={group.kind} className="cc-reach__group" aria-label={group.title}>
                  <h3 className="cc-reach__group-title">{group.title}</h3>
                  <ul className="cc-reach__lanes">
                    {lanes.map((unlock) => {
                      const index = rungIndex.get(unlock.opensAt);
                      const opens = unlock.opensAt > 0 && index !== undefined;
                      return (
                        <li key={unlock.name} className="cc-reach__lane" data-kind={unlock.kind} data-unopened={opens ? undefined : true}>
                          <span className="cc-reach__label">{unlock.name} <small>{opens ? `opens at ${unlock.opensAt} · ${nameAt.get(unlock.opensAt) ?? ""}` : "not scheduled"}</small></span>
                          <span className="cc-reach__track" aria-hidden="true" />
                          {opens ? <span className="cc-reach__bar" aria-hidden="true" style={{ gridColumn: `${index + 2} / -1`, "--span": ladder.rungs.length - index } as CSSProperties} /> : null}
                        </li>
                      );
                    })}
                  </ul>
                </section>
              );
            })}
            {goalCount ? (
              <div className="cc-reach__goals">
                <span className="cc-reach__goals-label">{goalCount} goals follow these ranks</span>
                {ladder.rungs.map((rung) => <span key={rung.id} className="cc-reach__pips" aria-hidden="true">{rung.goals.map((goal) => <i key={goal.name} />)}</span>)}
              </div>
            ) : null}
          </div>
        </div>
      )}
      <span className="cc-qualifier" data-qualifier data-tone={qualifier.tone}>
        {resolved?.sampled && resolution.ink !== "none" ? <InkMark ink={resolution.ink}>illustrative</InkMark> : null}
        {qualifier.text}
      </span>
    </div>
  );
});
