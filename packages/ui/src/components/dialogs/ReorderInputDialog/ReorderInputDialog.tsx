/**
 * Used to create/update a link between two routine nodes
 */
import {
    Box,
    Button,
    Dialog,
    DialogContent,
    Grid,
    Stack,
    Typography,
    useTheme
} from '@mui/material';
import { CancelIcon, SaveIcon } from '@shared/icons';
import { DialogTitle } from 'components/dialogs';
import { QuantityBox } from 'components/inputs';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ReorderInputDialogProps } from '../types';

const titleAria = 'reorder-input-output-dialog-title';

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

    const onCancel = useCallback(() => {
        handleClose();
    }, [handleClose]);

    return (
        <Dialog
            open={isOpen}
            onClose={onCancel}
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': { overflow: 'visible' },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                onClose={onCancel}
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
                        <Typography variant="h6" textAlign="center">
                            ⮕
                        </Typography>
                    </Box>
                    {/* To */}
                    <QuantityBox
                        id="move-input-output-to-position"
                        label="To"
                        min={1}
                        max={listLength + 1}
                        value={toIndex}
                        handleChange={handleToIndexChange}
                    />
                </Stack>
                {/* Action buttons */}
                <Grid container sx={{ padding: 0, paddingTop: '24px' }}>
                    <Grid item xs={12} sm={6} sx={{ paddingRight: 1 }}>
                        <Button
                            fullWidth
                            onClick={onSubmit}
                            startIcon={<SaveIcon />}
                        >Submit</Button>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                        <Button
                            fullWidth
                            onClick={onCancel}
                            sx={{ paddingLeft: 1 }}
                            startIcon={<CancelIcon />}
                        >Cancel</Button>
                    </Grid>
                </Grid>
            </DialogContent>
        </Dialog>
    )
}