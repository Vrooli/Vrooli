/**
 * HeaderPrimaryAction
 *
 * Renders the primary CTA button (Finalize / Run / Workshop) for the backlog
 * detail header. Reads item state from BacklogDetailContext.
 */

import { MessageSquareText, Play, Sparkles } from "lucide-react";
import { Button } from "../ui/button";
import { selectors } from "../../consts/selectors";
import { useBacklogDetail } from "../../contexts/BacklogDetailContext";
import { useBacklogDetailUIStore } from "../../stores";

interface HeaderPrimaryActionProps {
  className?: string;
  onFinalizeWorkshop: () => void;
  onRunWorkshop: () => void;
}

export function HeaderPrimaryAction({ className, onFinalizeWorkshop, onRunWorkshop }: HeaderPrimaryActionProps) {
  const { itemActions, agentRunningLabel, workshopActionLabel, isRunningAgent } = useBacklogDetail();
  const openRunModal = useBacklogDetailUIStore((s) => s.openRunModal);

  if (!itemActions) return null;

  switch (itemActions.primaryCta) {
    case "finalize":
      if (!itemActions.canFinalize && !itemActions.finalizeDisabled) return null;
      return (
        <Button variant="default" size="sm" className={className} onClick={onFinalizeWorkshop} disabled={itemActions.finalizeDisabled || isRunningAgent} title={(itemActions.finalizeDisabled || isRunningAgent) && itemActions.disabledReason ? itemActions.disabledReason : undefined}>
          <Sparkles className="mr-1.5 h-4 w-4" />
          {itemActions.agentRunning ? agentRunningLabel : isRunningAgent ? "Starting..." : "Finalize"}
        </Button>
      );
    case "run":
      if (!itemActions.canRun && !itemActions.runDisabled) return null;
      return (
        <Button variant="default" size="sm" className={className} onClick={openRunModal} disabled={itemActions.runDisabled} data-testid={selectors.backlogDetails.queueButton} title={itemActions.runDisabled && itemActions.disabledReason ? itemActions.disabledReason : undefined}>
          <Play className="mr-1.5 h-4 w-4" />
          {itemActions.agentRunning ? agentRunningLabel : "Run"}
        </Button>
      );
    case "workshop":
      if (!itemActions.canWorkshop && !itemActions.workshopDisabled) return null;
      return (
        <Button variant="default" size="sm" className={className} onClick={onRunWorkshop} disabled={itemActions.workshopDisabled || isRunningAgent} title={(itemActions.workshopDisabled || isRunningAgent) && itemActions.disabledReason ? itemActions.disabledReason : undefined}>
          <MessageSquareText className="mr-1.5 h-4 w-4" />
          {itemActions.agentRunning ? agentRunningLabel : isRunningAgent ? "Starting..." : workshopActionLabel}
        </Button>
      );
    default:
      return null;
  }
}
