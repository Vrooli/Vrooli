import { Loader2, Play, RefreshCw, Square } from "lucide-react";
import { Button } from "../ui/button";
import { ActionMenuSheetContent, type ActionMenuItem } from "../ui/action-menu";
import { selectors } from "../../consts/selectors";

export interface ScenarioLifecycleActionsProps {
  isRunning: boolean;
  isStopped: boolean;
  actionPending: boolean;
  actionInFlight: string | null;
  onAction: (action: "start" | "stop" | "restart") => void;
  /** Render as full-width mobile rows when true (default false). */
  mobile?: boolean;
  /** Close sheet after action (mobile). */
  onCloseSheet?: () => void;
}

export function ScenarioLifecycleActions({
  isRunning,
  isStopped,
  actionPending,
  actionInFlight,
  onAction,
  mobile = false,
  onCloseSheet,
}: ScenarioLifecycleActionsProps) {
  const runAction = (action: "start" | "stop" | "restart") => {
    onCloseSheet?.();
    onAction(action);
  };

  if (mobile) {
    const items: ActionMenuItem[] = [
      {
        label: "Start",
        icon: <Play />,
        loading: actionInFlight === "start",
        onSelect: () => runAction("start"),
        disabled: actionPending || isRunning,
      },
      {
        label: "Stop",
        icon: <Square />,
        loading: actionInFlight === "stop",
        onSelect: () => runAction("stop"),
        disabled: actionPending || isStopped,
      },
      {
        label: "Restart",
        icon: <RefreshCw />,
        loading: actionInFlight === "restart",
        onSelect: () => runAction("restart"),
        disabled: actionPending,
      },
    ];

    return <ActionMenuSheetContent items={items} />;
  }

  return (
    <div className="flex flex-wrap items-center justify-end gap-2" data-testid={selectors.scenarioDetails.actionsSection}>
      <Button
        variant={isRunning ? "outline" : "default"}
        size="sm"
        onClick={() => onAction("start")}
        disabled={actionPending || isRunning}
        data-testid={selectors.scenarioDetails.startButton}
      >
        {actionInFlight === "start" ? (
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <Play className="mr-2 h-4 w-4" />
        )}
        Start
      </Button>
      <Button
        variant={isStopped ? "outline" : "default"}
        size="sm"
        onClick={() => onAction("stop")}
        disabled={actionPending || isStopped}
        data-testid={selectors.scenarioDetails.stopButton}
      >
        {actionInFlight === "stop" ? (
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <Square className="mr-2 h-4 w-4" />
        )}
        Stop
      </Button>
      <Button
        variant="outline"
        size="sm"
        onClick={() => onAction("restart")}
        disabled={actionPending}
        data-testid={selectors.scenarioDetails.restartButton}
      >
        {actionInFlight === "restart" ? (
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <RefreshCw className="mr-2 h-4 w-4" />
        )}
        Restart
      </Button>
    </div>
  );
}
