import { EndNodeDialogProps } from '../types';
import { Checkbox, Dialog, FormControlLabel, Grid, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { getTranslation, updateTranslationFields } from 'utils';
import { nodeEndForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { DialogTitle } from 'components/dialogs';
import Markdown from 'markdown-to-jsx';
import { useCallback } from 'react';
import { Node } from 'types';
import { uuid } from '@shared/uuid';
import { GridSubmitButtons } from 'components/buttons';
import { linkColors } from 'styles';

const titleAria = 'end-node-dialog-title';

export const EndNodeDialog = ({
    handleClose,
    isEditing,
    isOpen,
    node,
    language,
    zIndex,
}: EndNodeDialogProps) => {
    const { palette } = useTheme();

    const formik = useFormik({
        initialValues: {
            name: !getTranslation(node, [language], false).name ? 'End' : getTranslation(node, [language], false).name as string,
            description: getTranslation(node, [language], false).description ?? '',
            wasSuccessful: node.data?.wasSuccessful ?? true,
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            const newTranslations = updateTranslationFields(node, language, { 
                name: values.name,
                description: values.description,
            });
            handleClose({
                ...node,
                translations: newTranslations as Node['translations'],
                data: {
                    __typename: 'NodeEnd',
                    id: node.data?.id ?? uuid(),
                    wasSuccessful: values.wasSuccessful,
                }
            })
        },
    });

    const handleCancel = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Don't close if formik is dirty and clicked outside
        if (formik.dirty && reason === 'backdropClick') return;
        // Otherwise, close
        formik.resetForm();
        handleClose();
    }, [formik, handleClose]);

    return (
        <Dialog
            open={isOpen}
            onClose={handleCancel}
            aria-labelledby={titleAria}
            sx={{
                zIndex,
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                onClose={handleCancel}
                title={isEditing ? "Edit End Node" : 'End Node Information'}
            />
            <form onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, ...linkColors(palette) }}>
                    <Grid item xs={12}>
                        <Typography variant="h6">Label</Typography>
                        {
                            isEditing ? (
                                <TextField
                                    autoFocus
                                    fullWidth
                                    id="name"
                                    name="name"
                                    value={formik.values.name}
                                    multiline
                                    maxRows={3}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.name && Boolean(formik.errors.name)}
                                    helperText={formik.touched.name && formik.errors.name}
                                />
                            ) : (
                                <Markdown>{formik.values.name}</Markdown>
                            )
                        }
                    </Grid>
                    <Grid item xs={12}>
                        <Typography variant="h6">Description</Typography>
                        {
                            isEditing ? (
                                <TextField
                                    fullWidth
                                    id="description"
                                    name="description"
                                    value={formik.values.description}
                                    multiline
                                    maxRows={3}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.description && Boolean(formik.errors.description)}
                                    helperText={formik.touched.description && formik.errors.description}
                                />
                            ) : (
                                <Markdown>{formik.values.description}</Markdown>
                            )
                        }
                    </Grid>
                    <Grid item xs={12}>
                        <Tooltip placement={'top'} title='If a routine finishes at this node, should it be considered a success?'>
                            <FormControlLabel
                                disabled={!isEditing}
                                label='Success?'
                                control={
                                    <Checkbox
                                        id={`end-node-was-successful`}
                                        size="medium"
                                        name='wasSuccessful'
                                        color='secondary'
                                        checked={formik.values.wasSuccessful}
                                        onChange={(_e, checked) => { formik.setFieldValue('wasSuccessful', checked) }}
                                    />
                                }
                            />
                        </Tooltip>
                    </Grid>
                    {isEditing && <GridSubmitButtons
                        errors={formik.errors}
                        isCreate={false}
                        loading={formik.isSubmitting}
                        onCancel={handleCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />}
                </Grid>
            </form>
        </Dialog>
    )
}