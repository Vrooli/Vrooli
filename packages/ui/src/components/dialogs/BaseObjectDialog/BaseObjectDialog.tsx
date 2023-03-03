import {
    Box,
    useTheme,
} from '@mui/material';
import { useCallback } from 'react';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { LargeDialog } from '../LargeDialog/LargeDialog';
import { BaseObjectDialogProps, ObjectDialogAction } from '../types';

const titleId = 'base-object-dialog-title';

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
        <LargeDialog
            id="object-dialog"
            isOpen={open}
            onClose={onClose}
            titleId={titleId}
            zIndex={zIndex}
        >
            <DialogTitle
                id={titleId}
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
        </LargeDialog>
    );
}