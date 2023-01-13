import { EndNodeDialogProps } from '../types';
import { Checkbox, Dialog, FormControlLabel, Grid, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { useTranslatedFields } from 'utils';
import { nodeEndValidation, nodeTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { DialogTitle } from 'components/dialogs';
import Markdown from 'markdown-to-jsx';
import { useCallback } from 'react';
import { DUMMY_ID, uuid } from '@shared/uuid';
import { GridSubmitButtons } from 'components/buttons';
import { linkColors } from 'styles';
import { useTranslation } from 'react-i18next';
import { GqlModelType, Node } from '@shared/consts';

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
    const { t } = useTranslation();

    const formik = useFormik({
        initialValues: {
            translationsUpdate: node.translations ?? [{
                id: DUMMY_ID,
                language,
                name: 'End',
                description: '',
            }],
            wasSuccessful: node.end?.wasSuccessful ?? true,
        },
        enableReinitialize: true,
        validationSchema: nodeEndValidation.update(),
        onSubmit: (values) => {
            handleClose({
                ...node,
                end: {
                    id: node.end?.id ?? uuid(),
                    node: { id: node.id } as Node,
                    suggestedNextRoutineVersions: [],
                    type: GqlModelType.NodeEnd,
                    wasSuccessful: values.wasSuccessful,
                },
                translations: values.translationsUpdate.map(t => ({
                    ...t,
                    id: t.id === DUMMY_ID ? uuid() : t.id,
                })),
            })
        },
    });

    const translations = useTranslatedFields({
        fields: ['description', 'name'],
        formik,
        formikField: 'translationsUpdate',
        language,
        validationSchema: nodeTranslationValidation.update(),
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
                title={t(`common:${isEditing ? 'NodeEndEdit' : 'NodeEndInfo'}`, { lng: language })}
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
                                    value={translations.name}
                                    multiline
                                    maxRows={3}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={translations.touchedName && Boolean(translations.errorName)}
                                    helperText={translations.touchedName && translations.errorName}
                                />
                            ) : (
                                <Markdown>{translations.name}</Markdown>
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
                                    value={translations.description}
                                    multiline
                                    maxRows={3}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={translations.touchedDescription && Boolean(translations.errorDescription)}
                                    helperText={translations.touchedDescription && translations.errorDescription}
                                />
                            ) : (
                                <Markdown>{translations.description}</Markdown>
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
                        errors={translations.errorsWithTranslations}
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