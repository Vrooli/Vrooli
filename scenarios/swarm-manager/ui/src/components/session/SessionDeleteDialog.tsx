import { ConfirmDialog } from "../ui/confirm-dialog";
import type { AgentSession } from "../../types";

interface SessionDeleteDialogProps {
  session: AgentSession;
  isOpen: boolean;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export function SessionDeleteDialog({
  session,
  isOpen,
  isDeleting,
  onClose,
  onConfirm,
}: SessionDeleteDialogProps) {
  const confirmationText = session.title.trim() || session.id;

  return (
    <ConfirmDialog
      isOpen={isOpen}
      onClose={onClose}
      onConfirm={onConfirm}
      title="Delete Session"
      description="This removes the conversation, session details, proposal drafts, and session artifact links. Created backlog items, initiatives, captures, files, and agent activity records stay in Swarm Manager."
      confirmationText={confirmationText}
      confirmLabel="Delete Session"
      isLoading={isDeleting}
      testIds={{
        dialog: "session-delete-dialog",
        confirmButton: "session-delete-confirm",
        cancelButton: "session-delete-cancel",
      }}
    />
  );
}
