import { EndNodeDialogProps } from '../types';
import { Checkbox, Dialog, FormControlLabel, Grid, TextField, Tooltip, Typography } from '@mui/material';
import { getTranslation, updateTranslationField } from 'utils';
import { nodeEndForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { DialogTitle } from 'components/dialogs';
import Markdown from 'markdown-to-jsx';
import { useCallback } from 'react';
import { Node } from 'types';
import { v4 as uuid } from 'uuid';
import { GridSubmitButtons } from 'components/buttons';

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
            title: !getTranslation(node, 'title', [language], false) ? 'End' : getTranslation(node, 'title', [language], false) as string,
            description: getTranslation(node, 'description', [language], false) ?? '',
            wasSuccessful: node.data?.wasSuccessful ?? true,
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            //TODO probs not working
            let newTranslations = updateTranslationField(node, 'title', values.title, language);
            newTranslations = updateTranslationField({ translations: newTranslations }, 'description', values.description, language);
            console.log('new translations', newTranslations);
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

    console.log('isvalid', formik.isValid, formik.errors);

    const onClose = useCallback(() => {
        formik.resetForm();
        handleClose();
    }, [formik, handleClose]);

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
                                    autoFocus
                                    fullWidth
                                    id="title"
                                    name="title"
                                    value={formik.values.title}
                                    multiline
                                    maxRows={3}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.title && Boolean(formik.errors.title)}
                                    helperText={formik.touched.title && formik.errors.title}
                                />
                            ) : (
                                <Markdown>{formik.values.title}</Markdown>
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
                    <GridSubmitButtons
                        disabledCancel={formik.isSubmitting}
                        disabledSubmit={formik.isSubmitting || !formik.isValid}
                        errors={formik.errors}
                        isCreate={false}
                        onCancel={onClose}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </form>
        </Dialog>
    )
}