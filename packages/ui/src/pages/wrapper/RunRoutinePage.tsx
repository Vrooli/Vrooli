import { Dialog } from "@mui/material";
import { RunRoutineView } from "components";
import { RunRoutinePageProps } from "./types";

export const RunRoutinePage = ({
    session
}: RunRoutinePageProps) => {
    const onClose = () => { window.history.back() };

    return (
        <>
            <Dialog
                id="run-routine-view-dialog"
                fullScreen
                open={true}
                onClose={onClose}
            >
                <RunRoutineView
                    handleClose={onClose}
                    session={session}
                />
            </Dialog>
        </>
    )
}