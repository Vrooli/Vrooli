import { IconButton, Tooltip, useTheme } from "@mui/material";
import { ReportIcon } from "icons";
import { useCallback, useState } from "react";
import { ReportUpsert } from "views/objects/report";
import { ReportButtonProps } from "../types";

export const ReportButton = ({
    forId,
    reportFor,
}: ReportButtonProps) => {
    const { palette } = useTheme();

    const [open, setOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setOpen(true); }, []);
    const closeDialog = useCallback(() => { setOpen(false); }, []);

    return (
        <>
            <ReportUpsert
                isCreate={true}
                isOpen={open}
                onCancel={closeDialog}
                onCompleted={closeDialog}
                overrideObject={{ createdFor: { __typename: reportFor, id: forId } }}
            />
            <Tooltip title="Report">
                <IconButton aria-label="Report" size="small" onClick={openDialog}>
                    <ReportIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
        </>
    );
};
