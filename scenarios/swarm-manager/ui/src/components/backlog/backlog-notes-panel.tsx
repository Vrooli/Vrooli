import { useState } from "react";
import { formatBacklogStatus } from "../../types";
import { useBacklogDetail } from "../../contexts/BacklogDetailContext";
import { planWorkshopService } from "../../services/plan-workshop-service";
import { PlanWorkshopPanel } from "../plan-workshop/plan-workshop-panel";

export function BacklogNotesPanel() {
  const { backlogKind, item, isLocked, isTerminal } = useBacklogDetail();
  const [acceptingPlan, setAcceptingPlan] = useState(false);
  const [acceptanceMessage, setAcceptanceMessage] = useState<string | null>(null);

  return (
    <div className="space-y-3 mt-4 border-t border-slate-800 pt-4">
      {item?.planRef && backlogKind !== "research" && !isTerminal && (
        <div className="flex items-center justify-between gap-3 rounded-lg border border-sky-500/20 bg-sky-500/5 px-4 py-3">
          <div>
            <p className="text-sm font-medium text-sky-100">Canonical plan acceptance</p>
            <p className="text-xs text-slate-400">
              {item.planAcceptance
                ? `Accepted ${new Date(item.planAcceptance.acceptedAt).toLocaleString()}. Re-accept after any plan or scope change.`
                : "Review the linked canonical plan, then explicitly accept this revision before queueing."}
            </p>
            {acceptanceMessage && <p className="mt-1 text-xs text-emerald-300">{acceptanceMessage}</p>}
          </div>
          <button
            className="shrink-0 rounded-md bg-sky-500 px-3 py-1.5 text-sm font-medium text-slate-950 hover:bg-sky-400 disabled:cursor-not-allowed disabled:opacity-50"
            disabled={isLocked || acceptingPlan}
            onClick={() => {
              if (!item) return;
              setAcceptingPlan(true);
              setAcceptanceMessage(null);
              void planWorkshopService.acceptPlan(backlogKind, item.name)
                .then(() => setAcceptanceMessage("Plan revision accepted. Refresh the item to see its recorded acceptance."))
                .catch((error: unknown) => setAcceptanceMessage(error instanceof Error ? error.message : "Unable to accept the plan revision."))
                .finally(() => setAcceptingPlan(false));
            }}
          >
            {acceptingPlan ? "Accepting…" : item.planAcceptance ? "Re-accept plan" : "Accept plan"}
          </button>
        </div>
      )}
      {isLocked && (
        <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 px-4 py-2 text-sm text-amber-300">
          This item is {item?.status ? formatBacklogStatus(item.status) : "locked"} and cannot be edited.
        </div>
      )}
      {item && <PlanWorkshopPanel subject={{ kind: "backlog_item", ref: `${backlogKind}/${item.name}` }} disabled={isLocked || isTerminal} />}
    </div>
  );
}
