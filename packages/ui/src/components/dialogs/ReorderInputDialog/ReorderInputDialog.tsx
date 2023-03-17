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
import { GridSubmitButtons } from 'components/buttons/GridSubmitButtons/GridSubmitButtons';
import { IntegerInput } from 'components/inputs/IntegerInput/IntegerInput';
import { useFormik } from 'formik';
import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { DialogTitle } from '../DialogTitle/DialogTitle';
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
    const { t } = useTranslation();

    const isOpen = useMemo(() => startIndex >= 0, [startIndex]);

    const formik = useFormik({
        initialValues: {
            toIndex: Math.max(0, startIndex) + 1, // Add 1 because non-programmers start counting at 1 (weirdos :P)
        },
        enableReinitialize: true,
        onSubmit: (values) => {
            handleClose(values.toIndex - 1);
        },
    });

    const handleCancel = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Don't close if toIndex !== startIndex (i.e. not initial condition) and clicked outside
        if (formik.values.toIndex !== startIndex + 1 && reason === 'backdropClick') return;
        // Otherwise, close
        handleClose();
    }, [formik.values.toIndex, handleClose, startIndex]);

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
                        {t('From')}: {startIndex + 1}
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
                        label={t('To')}
                        min={1}
                        max={listLength + 1}
                        name="to"
                    />
                </Stack>
                {/* Action buttons */}
                <Grid container spacing={2} sx={{ padding: 0, paddingTop: '24px' }}>
                    <GridSubmitButtons
                        display="dialog"
                        isCreate={false}
                        onCancel={handleCancel}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </DialogContent>
        </Dialog>
    )
}