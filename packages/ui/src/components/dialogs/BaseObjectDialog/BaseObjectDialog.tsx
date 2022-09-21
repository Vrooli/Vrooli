import {
    Box,
    Dialog,
    useTheme,
} from '@mui/material';
import { useCallback } from 'react';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { BaseObjectDialogProps, ObjectDialogAction } from '../types';

const titleAria = 'base-object-dialog-title';

/**
 * Dialog for displaying any "Add" form
 * @returns 
 */
export const BaseObjectDialog = ({
    children,
    onAction,
    open = true,
    title = '',
    zIndex,
}: BaseObjectDialogProps) => {
    const { palette } = useTheme();

    const onClose = useCallback(() => onAction(ObjectDialogAction.Close), [onAction]);

    return (
        <Dialog
            fullScreen
            open={open}
            onClose={onClose}
            aria-labelledby={titleAria}
            sx={{
                zIndex,
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                onClose={onClose}
                title={title}
            />
            <Box
                sx={{
                    background: palette.mode === 'light' ? '#c2cadd' : palette.background.default,
                    flex: 'auto',
                    padding: 0,
                    width: '100%',
                }}
            >
                {children}
            </Box>
        </Dialog>
    );
}