import { WelcomeView } from "views";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { WelcomeDialogProps } from "../types";

export const WelcomeDialog = ({
    isOpen,
    onClose,
    session,
}: WelcomeDialogProps) => {
    return (
        <LargeDialog
            id="welcome-dialog"
            isOpen={isOpen}
            onClose={onClose}
            zIndex={10000}
        >
            <WelcomeView display="dialog" session={session}/>
        </LargeDialog>
    )
}