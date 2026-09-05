import { formatBacklogStatus } from "../../types";
import { useBacklogDetail } from "../../contexts/BacklogDetailContext";

export function BacklogNotesPanel() {
  const { item, isLocked } = useBacklogDetail();

  return (
    <div className="space-y-3 mt-4 border-t border-slate-800 pt-4">
      {isLocked && (
        <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 px-4 py-2 text-sm text-amber-300">
          This item is {item?.status ? formatBacklogStatus(item.status) : "locked"} and cannot be edited.
        </div>
      )}
    </div>
  );
}
