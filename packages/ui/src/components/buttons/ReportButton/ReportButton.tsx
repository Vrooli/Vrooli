import { useCallback, useState } from 'react';
import { IconButton, Tooltip, useTheme } from '@mui/material';
import { ReportButtonProps } from '../types';
import { ReportIcon } from '@shared/icons';
import { ReportDialog } from 'components/dialogs';

export const ReportButton = ({
    forId,
    reportFor,
    session,
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
                session={session}
                zIndex={zIndex + 1}
            />
            <Tooltip title="Report">
                <IconButton aria-label="Report" size="small" onClick={openDialog}>
                    <ReportIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
        </>
    )
}