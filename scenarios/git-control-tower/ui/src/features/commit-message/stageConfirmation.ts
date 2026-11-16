import type { QueuedStageSelection } from "./stageQueue";

export type StageConfirmationCheck = "stale-repository" | "queue-changed" | "confirmed";

function identity(selection: QueuedStageSelection) {
	return JSON.stringify([selection.id, selection.snapshotDigest ?? "", [...selection.paths].sort(), selection.jobId ?? "", selection.requestKey ?? "", selection.suggestionMode ?? "", selection.state]);
}

export function checkStageConfirmation(capturedRepo: string, activeRepo: string, confirmed: QueuedStageSelection[], current: QueuedStageSelection[]): StageConfirmationCheck {
  if (capturedRepo !== activeRepo) return "stale-repository";
  return confirmed.map(identity).sort().join("\n") === current.map(identity).sort().join("\n") ? "confirmed" : "queue-changed";
}

export function isSameQueuedSelection(left: QueuedStageSelection, right: QueuedStageSelection) {
  return identity(left) === identity(right);
}

export async function removeQueueSelectionIfUnchanged(
  expected: QueuedStageSelection,
  read: () => Promise<{ revision: bigint; entries: QueuedStageSelection[] }>,
  remove: (revision: bigint) => Promise<void>,
) {
  for (let attempt = 0; attempt < 3; attempt++) {
    const queue = await read();
    const current = queue.entries.find(entry => entry.id === expected.id);
    if (!current) return true;
    if (!isSameQueuedSelection(current, expected)) return false;
    try {
      await remove(queue.revision);
      return true;
    } catch {
      if (attempt === 2) return false;
      await new Promise(resolve => setTimeout(resolve, 40 * (attempt + 1)));
    }
  }
  return false;
}
