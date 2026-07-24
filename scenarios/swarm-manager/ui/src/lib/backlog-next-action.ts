import type { BacklogNextAction } from "../services/backlog/types";

export type BacklogNextActionDetailTab = "prompt" | "decide" | "activity" | "related";

/** Maps server-owned action targets to the detail tab that performs the action. */
export function nextActionDetailTab(action: BacklogNextAction): BacklogNextActionDetailTab | undefined {
  switch (action.target) {
    case "plan_author":
    case "plan_accept":
    case "plan_repair":
      return "prompt";
    case "review":
	case "decision_stream":
	case "follow_up_dispatch":
	case "follow_up_author":
      return "decide";
    case "execution":
      return "activity";
    case "dependencies":
      return "related";
    default:
      return undefined;
  }
}
