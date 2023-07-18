import { IconButton, Tooltip, useTheme } from "@mui/material";
import { ReportDialog } from "components/dialogs/ReportDialog/ReportDialog";
import { ReportIcon } from "icons";
import { useCallback, useState } from "react";
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
