import { IconButton, Tooltip, useTheme } from "@mui/material";
import { ShareObjectDialog } from "components/dialogs/ShareObjectDialog/ShareObjectDialog";
import { ShareIcon } from "icons";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { ShareButtonProps } from "../types";

export const ShareButton = ({
    object,
    zIndex,
}: ShareButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [open, setOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setOpen(true); }, []);
    const closeDialog = useCallback(() => { setOpen(false); }, []);

    return (
        <>
            <ShareObjectDialog
                object={object}
                open={open}
                onClose={closeDialog}
                zIndex={zIndex + 1}
            />
            <Tooltip title={t("Share")}>
                <IconButton aria-label={t("Share")} size="small" onClick={openDialog}>
                    <ShareIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
        </>
    );
};
