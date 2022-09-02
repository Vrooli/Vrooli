import { EndNodeDialogProps } from '../types';
import { Button, Checkbox, Dialog, FormControlLabel, Grid, TextField, Tooltip, Typography } from '@mui/material';
import { getTranslation, updateTranslationField } from 'utils';
import { useFormik } from 'formik';
import { DialogTitle } from 'components/dialogs';
import Markdown from 'markdown-to-jsx';
import { useCallback } from 'react';
import { CancelIcon, SaveIcon } from '@shared/icons';
import { Node } from 'types';
import { v4 as uuid } from 'uuid';

const titleAria = 'end-node-dialog-title';

export const EndNodeDialog = ({
    handleClose,
    isEditing,
    isOpen,
    node,
    language,
    zIndex,
}: EndNodeDialogProps) => {
    const formik = useFormik({
        initialValues: {
            label: getTranslation(node, 'title', [language], false) ?? '',
            description: getTranslation(node, 'description', [language], false) ?? '',
            wasSuccessful: node.data?.wasSuccessful ?? true,
        },
        enableReinitialize: true,
        // validationSchema,
        onSubmit: (values) => {
            let newTranslations = updateTranslationField(node, 'title', values.label, language);
            newTranslations = updateTranslationField({ translations: newTranslations }, 'description', values.description, language);
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

    const onClose = useCallback(() => {
        handleClose();
    }, [handleClose]);

    return (
        <Dialog
            open={isOpen}
            onClose={onClose}
            aria-labelledby={titleAria}
            sx={{
                zIndex,
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                onClose={onClose}
                title={isEditing ? "Edit End Node" : 'End Node Information'}
            />
            <form onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2 }}>
                    <Grid item xs={12}>
                        <Typography variant="h6">Label</Typography>
                        {
                            isEditing ? (
                                <TextField
                                    fullWidth
                                    id="label"
                                    name="label"
                                    value={formik.values.label}
                                    multiline
                                    maxRows={3}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.label && Boolean(formik.errors.label)}
                                    helperText={formik.touched.label && formik.errors.label}
                                />
                            ) : (
                                <Markdown>{formik.values.label}</Markdown>
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
                                        onTouchStart={() => { formik.setFieldValue('wasSuccessful', !formik.values.wasSuccessful) }}
                                    />
                                }
                            />
                        </Tooltip>
                    </Grid>
                    <Grid item xs={6} marginBottom={4}>
                        <Button
                            disabled={Boolean(formik.isSubmitting || !formik.isValid)}
                            fullWidth
                            type="submit"
                            startIcon={<SaveIcon />}
                        >Save</Button>
                    </Grid>
                    <Grid item xs={6} marginBottom={4}>
                        <Button
                            fullWidth
                            onClick={onClose}
                            startIcon={<CancelIcon />}
                        >Cancel</Button>
                    </Grid>
                </Grid>
            </form>
        </Dialog>
    )
}