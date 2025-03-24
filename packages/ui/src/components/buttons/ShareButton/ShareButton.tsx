import { ListObject } from "@local/shared";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { ShareIcon } from "../../../icons/common.js";
import { ShareObjectDialog } from "../../dialogs/ShareObjectDialog/ShareObjectDialog.js";
import { ShareButtonProps } from "../types.js";

export function ShareButton({
    object,
}: ShareButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [open, setOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setOpen(true); }, []);
    const closeDialog = useCallback(() => { setOpen(false); }, []);

    return (
        <>
            <ShareObjectDialog
                object={object as ListObject}
                open={open}
                onClose={closeDialog}
            />
            <Tooltip title={t("Share")}>
                <IconButton aria-label={t("Share")} size="small" onClick={openDialog}>
                    <ShareIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
        </>
    );
}
