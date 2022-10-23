import { useCallback, useState } from 'react';
import { IconButton, Tooltip, useTheme } from '@mui/material';
import { ShareButtonProps } from '../types';
import { ShareIcon } from '@shared/icons';
import { ShareObjectDialog } from 'components/dialogs';

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
    )
}