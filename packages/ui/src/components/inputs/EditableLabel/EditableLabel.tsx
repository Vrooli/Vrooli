/**
 * Label that turns into a text input when clicked. 
 * Stores new text until committed.
 */
import { DialogContent, DialogContentText, IconButton, Stack, TextField } from '@mui/material';
import { EditIcon } from '@shared/icons';
import { GridSubmitButtons } from 'components/buttons/GridSubmitButtons/GridSubmitButtons';
import { DialogTitle } from 'components/dialogs/DialogTitle/DialogTitle';
import { LargeDialog } from 'components/dialogs/LargeDialog/LargeDialog';
import { useFormik } from 'formik';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import * as yup from 'yup';
import { EditableLabelProps } from '../types';

const titleId = 'editable-label-dialog-title';
const descriptionAria = 'editable-label-dialog-description';

export const EditableLabel = ({
    canUpdate,
    handleUpdate,
    placeholder,
    onDialogOpen,
    renderLabel,
    sxs,
    text,
    validationSchema,
    zIndex,
}: EditableLabelProps) => {
    const { t } = useTranslation();

    /**
     * Random string for unique ID
     */
    const [id] = useState(Math.random().toString(36).substr(2, 9));

    const formik = useFormik({
        initialValues: {
            text,
        },
        enableReinitialize: true,
        validationSchema: validationSchema ? yup.object().shape({
            text: validationSchema,
        }) : undefined,
        onSubmit: (values) => {
            handleUpdate(values.text);
            setActive(false);
            formik.setSubmitting(false);
        },
    });

    // Used for editing the title of the routine
    const [active, setActive] = useState<boolean>(false);
    useEffect(() => {
        if (typeof onDialogOpen === 'function') {
            onDialogOpen(active);
        }
    }, [active, onDialogOpen]);

    const toggleActive = useCallback((event: any) => {
        if (!canUpdate) return;
        setActive(!active)
    }, [active, canUpdate]);

    const handleCancel = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Don't close if formik is dirty and clicked outside
        if (formik.dirty && reason === 'backdropClick') return;
        // Otherwise, close
        setActive(false);
        formik.resetForm();
    }, [formik]);

    return (
        <>
            {/* Dialog with TextField for editing label */}
            <LargeDialog
                id="edit-label-dialog"
                onClose={handleCancel}
                isOpen={active}
                titleId={titleId}
                zIndex={zIndex + 1}
            >
                <DialogTitle
                    id={titleId}
                    onClose={handleCancel}
                    title={t('EditLabel')}
                />
                <DialogContent sx={{ paddingBottom: '80px' }}>
                    <DialogContentText id={descriptionAria}>
                        <TextField
                            autoFocus
                            margin="dense"
                            id="text"
                            type="text"
                            fullWidth
                            value={formik.values.text}
                            onChange={formik.handleChange}
                            error={Boolean(formik.errors.text)}
                            helperText={formik.errors.text}
                        />
                    </DialogContentText>
                </DialogContent>
                {/* Save and cancel buttons */}
                <GridSubmitButtons
                    display="dialog"
                    errors={formik.errors}
                    isCreate={false}
                    loading={formik.isSubmitting}
                    onCancel={handleCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </LargeDialog>
            {/* Non-popup elements */}
            <Stack direction="row" spacing={0} alignItems="center" sx={{ ...(sxs?.stack ?? {}) }}>
                {/* Label */}
                {renderLabel(text.trim().length > 0 ? text : (placeholder ?? ''))}
                {/* Edit icon */}
                {canUpdate && (
                    <IconButton
                        id={`edit-label-icon-button-${id}`}
                        onClick={toggleActive}
                        onTouchStart={toggleActive}
                        sx={{ color: 'inherit' }}
                    >
                        <EditIcon id={`edit-label-icon-${id}`} />
                    </IconButton>
                )}
            </Stack>
        </>
    )
};