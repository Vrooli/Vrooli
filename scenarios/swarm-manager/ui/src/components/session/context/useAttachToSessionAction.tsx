import { useCallback, useMemo, useState, type ReactNode } from "react";
import { Button } from "../../ui/button";
import type { ActionMenuItem } from "../../ui/action-menu";
import { EntityAttachToSessionSheet, AttachToSessionActionIcon } from "./EntityAttachToSessionSheet";
import type { SessionContextOption } from "./session-context-refs";
import { selectors } from "../../../consts/selectors";

interface UseAttachToSessionActionOptions {
  currentSessionId?: string;
}

export interface AttachToSessionAction {
  actionItem: ActionMenuItem;
  button: ReactNode;
  sheet: ReactNode;
  open: () => void;
}

export function useAttachToSessionAction(
  option: SessionContextOption | null | undefined,
  options: UseAttachToSessionActionOptions = {},
): AttachToSessionAction {
  const [open, setOpen] = useState(false);
  const available = Boolean(option);
  const openSheet = useCallback(() => {
    if (option) setOpen(true);
  }, [option]);

  const actionItem = useMemo<ActionMenuItem>(() => ({
    label: "Attach to session",
    icon: <AttachToSessionActionIcon />,
    disabled: !available,
    onSelect: openSheet,
    testId: selectors.agentSessions.entityAttachAction,
  }), [available, openSheet]);

  return {
    actionItem,
    open: openSheet,
    button: (
      <Button
        variant="ghost"
        size="sm"
        className="h-7 rounded-md px-2 text-xs"
        onClick={openSheet}
        disabled={!available}
        data-testid={selectors.agentSessions.entityAttachAction}
      >
        <AttachToSessionActionIcon />
        Attach
      </Button>
    ),
    sheet: option ? (
      <EntityAttachToSessionSheet
        isOpen={open}
        onClose={() => setOpen(false)}
        option={option}
        currentSessionId={options.currentSessionId}
      />
    ) : null,
  };
}
