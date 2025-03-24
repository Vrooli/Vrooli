import { IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { ReportIcon } from "../../../icons/common.js";
import { ReportUpsert } from "../../../views/objects/report/ReportUpsert.js";
import { ReportButtonProps } from "../types.js";

export function ReportButton({
    forId,
    reportFor,
}: ReportButtonProps) {
    const { palette } = useTheme();

    const [open, setOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setOpen(true); }, []);
    const closeDialog = useCallback(() => { setOpen(false); }, []);
    const createdFor = useMemo(() => ({ __typename: reportFor, id: forId }), [forId, reportFor]);

    return (
        <>
            <ReportUpsert
                createdFor={createdFor}
                display="dialog"
                isCreate={true}
                isOpen={open}
                onCancel={closeDialog}
                onClose={closeDialog}
                onCompleted={closeDialog}
                onDeleted={closeDialog}
            />
            <Tooltip title="Report">
                <IconButton aria-label="Report" size="small" onClick={openDialog}>
                    <ReportIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
        </>
    );
}
