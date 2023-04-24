import { ShareIcon } from "@local/shared";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { ShareObjectDialog } from "../../dialogs/ShareObjectDialog/ShareObjectDialog";
import { ShareButtonProps } from "../types";

export const ShareButton = ({
    object,
    zIndex,
}: ShareButtonProps) => {
    const { palette } = useTheme();

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
            <Tooltip title="Share">
                <IconButton aria-label="Share" size="small" onClick={openDialog}>
                    <ShareIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
        </>
    );
};
