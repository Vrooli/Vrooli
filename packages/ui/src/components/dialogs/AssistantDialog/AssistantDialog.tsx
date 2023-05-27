import { useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
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
    const theme = useTheme();
    const { t } = useTranslation();

    return (
        <LargeDialog
            id="assistant-dialog"
            isOpen={isOpen}
            onClose={handleClose}
            titleId={titleId}
            zIndex={zIndex}
        >
            <ChatView />
        </LargeDialog>
    );
};
