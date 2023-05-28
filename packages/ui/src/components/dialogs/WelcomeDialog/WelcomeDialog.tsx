import { WelcomeView } from "views/WelcomeView/WelcomeView";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { WelcomeDialogProps } from "../types";

export const WelcomeDialog = ({
    isOpen,
    onClose,
}: WelcomeDialogProps) => {
    return (
        <LargeDialog
            id="welcome-dialog"
            isOpen={isOpen}
            onClose={onClose}
            zIndex={10000}
        >
            <WelcomeView
                display="dialog"
                onClose={onClose}
                zIndex={10000}
            />
        </LargeDialog>
    );
};
