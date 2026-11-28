import { useCallback, useRef, useState } from "react";

import type { PacketDraft } from "./model/draft";
import { readAskDraft, writeAskDraft } from "./model/draftStore";

/**
 * The unsent answer, kept in step with its browser-side copy. Updates may be
 * given as a function of the latest draft, so a message from an explainer page
 * and a tap in the form can never overwrite each other.
 */
export function usePacketDraft(subject: string, askId: string) {
  const [draft, setDraft] = useState(() => readAskDraft(subject, askId));
  const latest = useRef(draft);
  const update = useCallback(
    (next: PacketDraft | ((current: PacketDraft) => PacketDraft)) => {
      const value = typeof next === "function" ? next(latest.current) : next;
      latest.current = value;
      setDraft(value);
      writeAskDraft(subject, askId, value);
    },
    [subject, askId],
  );
  return [draft, update] as const;
}
