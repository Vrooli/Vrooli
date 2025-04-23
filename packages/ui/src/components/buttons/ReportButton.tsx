import { IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { ReportUpsert } from "../../views/objects/report/ReportUpsert.js";
import { ReportButtonProps } from "./types.js";

export function ReportButton({
    forId,
    reportFor,
}: ReportButtonProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [open, setOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setOpen(true); }, []);
    const closeDialog = useCallback(() => { setOpen(false); }, []);
    const createdFor = useMemo(() => ({ __typename: reportFor, id: forId }), [forId, reportFor]);

    return (
        <>
            <ReportUpsert
                createdFor={createdFor}
                display="Dialog"
                isCreate={true}
                isOpen={open}
                onCancel={closeDialog}
                onClose={closeDialog}
                onCompleted={closeDialog}
                onDeleted={closeDialog}
            />
            <Tooltip title={t("Report")}>
                <IconButton
                    aria-label={t("Report")}
                    size="small"
                    onClick={openDialog}
                >
                    <IconCommon
                        decorative
                        fill={palette.background.textSecondary}
                        name="Report"
                    />
                </IconButton>
            </Tooltip>
        </>
    );
}
