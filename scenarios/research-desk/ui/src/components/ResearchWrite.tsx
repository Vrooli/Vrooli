import type { ResearchWrite } from "../api/activity";
import { toDate } from "../lib/format";
import { Unknown } from "./Fact";
import { DerivedWhen } from "./When";

/**
 * When research was last written, and, when its owner was paused, limited,
 * stopped or ended at the time, the API's note saying so ("written while
 * paused: an owner may still be running"), since the operator needs that
 * fact.
 */
export function ResearchWriteWhen({ write }: { write?: ResearchWrite }) {
  if (!toDate(write?.at?.value)) return <Unknown />;
  return (
    <>
      <DerivedWhen time={write?.at} />
      {write?.note ? <span className="rd-write-note block">{write.note}</span> : null}
    </>
  );
}
