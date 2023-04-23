import { ReportIcon } from "@local/icons";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { ReportDialog } from "../../dialogs/ReportDialog/ReportDialog";
import { ReportButtonProps } from "../types";

export const ReportButton = ({
    forId,
    reportFor,
    zIndex,
}: ReportButtonProps) => {
    const { palette } = useTheme();

    const [open, setOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setOpen(true); }, []);
    const closeDialog = useCallback(() => { setOpen(false); }, []);

    return (
        <>
            <ReportDialog
                forId={forId}
                onClose={closeDialog}
                open={open}
                reportFor={reportFor}
                zIndex={zIndex + 1}
            />
            <Tooltip title="Report">
                <IconButton aria-label="Report" size="small" onClick={openDialog}>
                    <ReportIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
        </>
    );
};
