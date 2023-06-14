import { uuid, VALYXA_ID } from "@local/shared";
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
            sxs={{
                paper: {
                    width: "min(100%, 1000px)",
                },
            }}
        >
            <ChatView
                chatInfo={{
                    invites: [{
                        __typename: "ChatInvite" as const,
                        id: uuid(),
                        user: {
                            __typename: "User" as const,
                            id: VALYXA_ID,
                            name: "Valyxa",
                        },
                    } as any],
                }}
                context={context}
                display="dialog"
                onClose={handleClose}
                task={task}
                zIndex={zIndex}
            />
        </LargeDialog>
    );
};
