/**
 * Used to create/update a link between two routine nodes
 */
import {
    Box,
    Dialog,
    DialogContent,
    Grid,
    Stack,
    Typography,
    useTheme
} from '@mui/material';
import { GridSubmitButtons } from 'components/buttons';
import { DialogTitle } from 'components/dialogs';
import { IntegerInput } from 'components/inputs';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ReorderInputDialogProps } from '../types';

const titleId = 'reorder-input-output-dialog-title';

export const ReorderInputDialog = ({
    handleClose,
    isInput,
    listLength,
    startIndex,
    zIndex,
}: ReorderInputDialogProps) => {
    const { palette } = useTheme();

    const isOpen = useMemo(() => startIndex >= 0, [startIndex]);

    const [toIndex, setToIndex] = useState<number>(1);
    useEffect(() => {
        setToIndex(Math.max(0, startIndex) + 1);
    }, [startIndex]);

    const handleToIndexChange = useCallback((index: number) => {
        setToIndex(index);
    }, [setToIndex]);

    const onSubmit = useCallback(() => {
        handleClose(toIndex - 1);
    }, [handleClose, toIndex]);

    const handleCancel = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Don't close if toIndex !== startIndex (i.e. not initial condition) and clicked outside
        if (toIndex !== startIndex + 1 && reason === 'backdropClick') return;
        // Otherwise, close
        handleClose();
    }, [handleClose, startIndex, toIndex]);

    return (
        <Dialog
            open={isOpen}
            onClose={handleCancel}
            aria-labelledby={titleId}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': { overflow: 'visible' },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            <DialogTitle
                id={titleId}
                onClose={handleCancel}
                title={`Reorder ${isInput ? 'Input' : 'Output'}`}
            />
            <DialogContent>
                <Stack direction="row" justifyContent="center" alignItems="center" pt={2} sx={{ color: palette.background.textPrimary }}>
                    {/* From */}
                    <Typography variant="h6">
                        From: {startIndex + 1}
                    </Typography>
                    {/* Right arrow */}
                    <Box sx={{
                        width: '3em',
                        height: '3em',
                        color: 'black',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                    }}>
                        <Typography variant="h6" textAlign="center" color={palette.background.textPrimary}>
                            ⮕
                        </Typography>
                    </Box>
                    {/* To */}
                    <IntegerInput
                        id="move-input-output-to-position"
                        label="To"
                        min={1}
                        max={listLength + 1}
                        value={toIndex}
                        handleChange={handleToIndexChange}
                    />
                </Stack>
                {/* Action buttons */}
                <Grid container spacing={2} sx={{ padding: 0, paddingTop: '24px' }}>
                    <GridSubmitButtons
                        display="dialog"
                        isCreate={false}
                        onCancel={handleCancel}
                        onSubmit={onSubmit}
                    />
                </Grid>
            </DialogContent>
        </Dialog>
    )
}