import { ChatView } from "views/ChatView/ChatView";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { AssistantDialogProps } from "../types";

const titleId = "advanced-search-dialog-title";

export const AssistantDialog = ({
    context,
    task,
    handleClose,
    handleComplete,
    isOpen,
    zIndex,
}: AssistantDialogProps) => {

    return (
        <LargeDialog
            id="assistant-dialog"
            isOpen={isOpen}
            onClose={handleClose}
            titleId={titleId}
            zIndex={zIndex}
        >
            <ChatView
                chatId="Valyxa"
                context={context}
                display="dialog"
                onClose={handleClose}
                task={task}
                zIndex={zIndex}
            />
        </LargeDialog>
    );
};
