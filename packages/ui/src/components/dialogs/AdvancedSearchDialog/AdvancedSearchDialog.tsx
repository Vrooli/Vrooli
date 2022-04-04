/**
 * Displays all search options for an organization
 */
 import {
    Box,
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    Typography
} from '@mui/material';
import { HelpButton } from 'components';
import { useMemo } from 'react';
import { AdvancedSearchDialogProps } from '../types';
import {
    Close as CloseIcon
} from '@mui/icons-material';

const helpText =
    `TODO`

export const AdvancedSearchDialog = ({
    handleClose,
    isOpen,
    session,
}: AdvancedSearchDialogProps) => {

    /**
     * Title bar with help button and close icon
     */
    const titleBar = useMemo(() => (
        <Box sx={{
            background: (t) => t.palette.primary.dark,
            color: (t) => t.palette.primary.contrastText,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: 2,
        }}>
            <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto' }}>
                {'Advanced Search'}
                <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
            </Typography>
            <Box sx={{ marginLeft: 'auto' }}>
                <IconButton
                    edge="start"
                    onClick={(e) => { handleClose() }}
                >
                    <CloseIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                </IconButton>
            </Box>
        </Box>
    ), [])

    return (
        <Dialog
            open={isOpen}
            onClose={handleClose}
            sx={{
                '& .MuiDialogContent-root': { overflow: 'visible', background: '#cdd6df' },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            {titleBar}
            <DialogContent>
                <Stack direction="column" spacing={4}>
                    ghjgkjgfhfghjTODO
                </Stack>
            </DialogContent>
        </Dialog>
    )
}