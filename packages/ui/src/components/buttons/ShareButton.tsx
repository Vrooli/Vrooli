import { IconButton, Tooltip, useTheme } from "@mui/material";
import { type ListObject } from "@vrooli/shared";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { ShareObjectDialog } from "../dialogs/ShareObjectDialog/ShareObjectDialog.js";
import { type ShareButtonProps } from "./types.js";

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
                <IconButton
                    onClick={openDialog}
                    size="small"
                >
                    <IconCommon
                        decorative
                        fill={palette.background.textSecondary}
                        name="Share"
                    />
                </IconButton>
            </Tooltip>
        </>
    );
}
