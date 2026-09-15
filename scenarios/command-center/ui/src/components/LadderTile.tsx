import type { CSSProperties } from "react";
import type { Reading } from "../lib/api";
import { qualify, resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import { ladderOf, nextRung, statusLabel, stillIdeas } from "../lib/ladder";
import { TileQualifier } from "./TileQualifier";

const STAGE: Record<string, number> = { SHIPPED: 1, ACTIVE: 0.8, TRIGGER_MET: 0.6, PROPOSED: 0.6, CANDIDATE: 0.4 };

/** The release ladder in the supporting strip: the next rung, and one segment per rung showing how far each has moved. */
export function LadderTile({ reading, showOrigin = false }: { reading: Reading; showOrigin?: boolean }) {
  const resolution = resolveReading(reading);
  const qualifier = qualify(reading, resolution);
  const resolved = ladderOf(reading);
  const ladder = resolved?.ladder;
  const rung = ladder ? nextRung(ladder) : null;
  return (
    <li className="cc-reading cc-ladder-tile" data-testid="ladder-tile" data-kind="ladder" data-reading data-metric-id={reading.id} data-origin-env={reading.origin_env} data-ink={resolution.ink} data-provenance={resolved?.sampled ? "sample" : resolution.figure === "measured" ? "measured" : "absent"} data-trust={reading.trust} title={reading.description}>
      <span className="cc-reading-label">{reading.label}</span>
      {rung ? (
        <>
          <span className="cc-ladder-tile__next"><span className="cc-ladder-tile__rank">{rung.rank}</span><span className="cc-ladder-tile__name">{rung.name}</span></span>
          <span className="cc-ladder-tile__status"><span className="cc-ladder-dot" data-status={rung.status} aria-hidden="true" />{statusLabel(rung.status)} · next of {ladder?.rungs.length}</span>
        </>
      ) : <span className="cc-ladder-tile__name">{ladder ? "All scheduled releases shipped" : "—"}</span>}
      {ladder ? (
        <span className="cc-ladder-tile__track" aria-hidden="true">
          {ladder.rungs.map((entry) => <span key={entry.id} className="cc-ladder-tile__seg" data-next={entry.rank === ladder.nextRank || undefined} style={{ "--stage": STAGE[entry.status] ?? 0.2 } as CSSProperties} />)}
        </span>
      ) : null}
      {ladder ? <span className="cc-ladder-tile__ideas">{stillIdeas(ladder)} of {ladder.rungs.length} still ideas{ladder.unscheduled.length ? ` · ${ladder.unscheduled.length} unscheduled` : ""}</span> : null}
      <TileQualifier qualifier={qualifier} reading={reading} showOrigin={showOrigin} />
    </li>
  );
}
