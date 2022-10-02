/**
 * Label that turns into a text input when clicked. 
 * Stores new text until committed.
 */
import { Dialog, DialogContent, DialogContentText, Grid, IconButton, Stack, TextField } from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import { EditableLabelProps } from '../types';
import { DialogTitle } from 'components/dialogs';
import { EditIcon } from '@shared/icons';
import { useFormik } from 'formik';
import { GridSubmitButtons } from 'components/buttons';
import * as yup from 'yup';

const titleAria = 'editable-label-dialog-title';
const descriptionAria = 'editable-label-dialog-description';

export const EditableLabel = ({
    canEdit,
    handleUpdate,
    placeholder,
    onDialogOpen,
    renderLabel,
    sxs,
    text,
    validationSchema,
}: EditableLabelProps) => {
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
            console.log('submitting title update', values.text)
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
        if (!canEdit) return;
        setActive(!active)
    }, [active, canEdit]);

    const cancel = useCallback(() => {
        setActive(false);
        formik.setFieldValue('text', text);
    }, [formik, text]);

    return (
        <>
            {/* Dialog with TextField for editing label */}
            <Dialog
                open={active}
                disableScrollLock={true}
                onClose={() => { }}
                aria-labelledby={titleAria}
                aria-describedby={descriptionAria}
                sx={{
                    '& .MuiPaper-root': {
                        minWidth: 'min(400px, 100%)',
                        margin: '0 auto',
                    },
                }}
            >
                <DialogTitle
                    ariaLabel={titleAria}
                    onClose={cancel}
                    title="Edit Label"
                />
                <DialogContent>
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
                <Grid container spacing={1} padding={1}>
                    <GridSubmitButtons
                        errors={formik.errors}
                        isCreate={false}
                        loading={formik.isSubmitting}
                        onCancel={cancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </Dialog>
            {/* Non-popup elements */}
            <Stack direction="row" spacing={0} alignItems="center" sx={{ ...(sxs?.stack ?? {}) }}>
                {/* Label */}
                {renderLabel(text.trim().length > 0 ? text : (placeholder ?? ''))}
                {/* Edit icon */}
                {canEdit && (
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