import { useMutation } from '@apollo/client';
import { resourceCreateForm as validationSchema } from '@shared/validation';
import { Button, Dialog, DialogContent, FormControl, Grid, InputLabel, MenuItem, Select, Stack, TextField } from '@mui/material';
import { useFormik } from 'formik';
import { resourceCreateMutation, resourceUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { ResourceDialogProps } from '../types';
import {
    Add as AddIcon,
    Cancel as CancelIcon,
} from '@mui/icons-material';
import { getTranslation, getUserLanguages, PubSub, ResourceShape, ResourceTranslationShape, shapeResourceCreate, shapeResourceUpdate, updateArray } from 'utils';
import { resourceCreate, resourceCreateVariables } from 'graphql/generated/resourceCreate';
import { ResourceUsedFor } from 'graphql/generated/globalTypes';
import { resourceUpdate, resourceUpdateVariables } from 'graphql/generated/resourceUpdate';
import { useCallback, useEffect, useState } from 'react';
import { LanguageInput } from 'components/inputs';
import { Resource } from 'types';
import { v4 as uuid } from 'uuid';
import { DialogTitle } from 'components';

const helpText =
    `## What are resources?

Resources provide context to the object they are attached to, such as a  user, organization, project, or routine.

## Examples
**For a user** - Social media links, GitHub profile, Patreon

**For an organization** - Official website, tools used by your team, news article explaining the vision

**For a project** - Project Catalyst proposal, Donation wallet address

**For a routine** - Guide, external service
`

export const UsedForDisplay = {
    [ResourceUsedFor.Community]: 'Community',
    [ResourceUsedFor.Context]: 'Context',
    [ResourceUsedFor.Developer]: 'Developer',
    [ResourceUsedFor.Donation]: 'Donation',
    [ResourceUsedFor.ExternalService]: 'External Service',
    [ResourceUsedFor.Feed]: 'Feed',
    [ResourceUsedFor.Install]: 'Install',
    [ResourceUsedFor.Learning]: 'Learning',
    [ResourceUsedFor.Notes]: 'Notes',
    [ResourceUsedFor.OfficialWebsite]: 'Official Webiste',
    [ResourceUsedFor.Proposal]: 'Proposal',
    [ResourceUsedFor.Related]: 'Related',
    [ResourceUsedFor.Researching]: 'Researching',
    [ResourceUsedFor.Scheduling]: 'Scheduling',
    [ResourceUsedFor.Social]: 'Social',
    [ResourceUsedFor.Tutorial]: 'Tutorial',
}

const titleAria = "resource-dialog-title";

export const ResourceDialog = ({
    mutate,
    open,
    onClose,
    onCreated,
    onUpdated,
    index,
    partialData,
    session,
    listId,
    zIndex,
}: ResourceDialogProps) => {
    const [addMutation, { loading: addLoading }] = useMutation<resourceCreate, resourceCreateVariables>(resourceCreateMutation);
    const [updateMutation, { loading: updateLoading }] = useMutation<resourceUpdate, resourceUpdateVariables>(resourceUpdateMutation);

    // Handle translations
    type Translation = ResourceTranslationShape;
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        setTranslations([...translations.filter(t => t.language !== language)]);
    }, [translations, setTranslations]);
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(translations, index, translation) : [...translations, translation];
    }, [translations]);
    const updateTranslation = useCallback((language: string, translation: Translation) => {
        setTranslations(getTranslationsUpdate(language, translation));
    }, [getTranslationsUpdate]);

    useEffect(() => {
        setTranslations(partialData?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            description: t.description ?? '',
            title: t.title ?? '',
        })) ?? []);
    }, [partialData]);

    const [language, setLanguage] = useState<string>(index < 0 ? getUserLanguages(session)[0] : '');
    const [languages, setLanguages] = useState<string[]>(index < 0 ? getUserLanguages(session) : []);

    const formik = useFormik({
        initialValues: {
            link: partialData?.link ?? '',
            usedFor: partialData?.usedFor ?? ResourceUsedFor.Context,
            title: getTranslation(partialData, 'title', [language], false) ?? '',
            description: getTranslation(partialData, 'description', [language], false) ?? '',
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                title: values.title,
            })
            const input: ResourceShape = {
                id: partialData?.id ?? uuid(),
                index: Math.max(index, 0),
                listId,
                link: values.link,
                usedFor: values.usedFor,
                translations: allTranslations,
            };
            if (mutate) {
                const onSuccess = (response) => {
                    PubSub.get().publishSnack({ message: (index < 0) ? 'Resource created.' : 'Resource updated.' });
                    (index < 0) ? onCreated(response.data.resourceCreate) : onUpdated(index ?? 0, response.data.resourceUpdate);
                    formik.resetForm();
                    onClose();
                }
                // If index is negative, create
                if (index < 0) {
                    mutationWrapper({
                        mutation: addMutation,
                        input: shapeResourceCreate(input),
                        successCondition: (response) => response.data.resourceCreate !== null,
                        onSuccess,
                        onError: () => { formik.setSubmitting(false) },
                    })
                }
                // Otherwise, update
                else {
                    if (!partialData || !partialData.id || !listId) {
                        PubSub.get().publishSnack({ message: 'Could not find resource to update.', severity: 'error' });
                        return;
                    }
                    mutationWrapper({
                        mutation: updateMutation,
                        input: shapeResourceUpdate({ ...partialData, listId } as ResourceShape, input),
                        successCondition: (response) => response.data.resourceUpdate !== null,
                        onSuccess,
                        onError: () => { formik.setSubmitting(false) },
                    })
                }
            } else {
                onCreated(input as Resource);
                formik.resetForm();
                onClose();
            }
        },
    });

    // Handle languages
    useEffect(() => {
        if (languages.length === 0 && translations.length > 0) {
            setLanguage(translations[0].language);
            setLanguages(translations.map(t => t.language));
            formik.setValues({
                ...formik.values,
                description: translations[0].description ?? '',
                title: translations[0].title ?? '',
            })
        }
    }, [formik, languages, setLanguage, setLanguages, translations])
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
            title: existingTranslation?.title ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            title: formik.values.title,
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, formik.values.title, updateFormikTranslation]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
    }, [handleLanguageSelect, languages, setLanguages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        deleteTranslation(language);
        updateFormikTranslation(newLanguages[0]);
        setLanguage(newLanguages[0]);
        setLanguages(newLanguages);
    }, [deleteTranslation, languages, updateFormikTranslation]);

    const handleClose = () => {
        formik.resetForm();
        onClose();
    }

    return (
        <Dialog
            onClose={handleClose}
            open={open}
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialog-paper': {
                    width: 'min(500px, 100vw)',
                    textAlign: 'center',
                    overflow: 'hidden',
                }
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                title={(index < 0) ? 'Add Resource' : 'Update Resource'}
                helpText={helpText}
                onClose={handleClose}
            />
            <DialogContent>
                <form onSubmit={formik.handleSubmit}>
                    <Stack direction="column" spacing={2} paddingTop={2}>
                        {/* Language select */}
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleLanguageDelete}
                            handleCurrent={handleLanguageSelect}
                            selectedLanguages={languages}
                            session={session}
                            zIndex={zIndex + 1}
                        />
                        {/* Enter link */}
                        <TextField
                            fullWidth
                            id="link"
                            name="link"
                            label="Link"
                            value={formik.values.link}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.link && Boolean(formik.errors.link)}
                            helperText={formik.touched.link && formik.errors.link}
                        />
                        {/* Select resource type */}
                        <FormControl fullWidth>
                            <InputLabel id="rsource-type-label">Reason</InputLabel>
                            <Select
                                labelId="resource-type-label"
                                id="usedFor"
                                value={formik.values.usedFor}
                                label="Resource type"
                                onChange={(e) => formik.setFieldValue('usedFor', e.target.value)}
                            >
                                {Object.entries(UsedForDisplay).map(([key, value]) => (
                                    <MenuItem key={key} value={key}>{value}</MenuItem>
                                ))}
                            </Select>
                        </FormControl>
                        {/* Enter title */}
                        <TextField
                            fullWidth
                            id="title"
                            name="title"
                            label="Title"
                            value={formik.values.title}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.title && Boolean(formik.errors.title)}
                            helperText={(formik.touched.title && formik.errors.title) ?? "Enter title (optional)"}
                        />
                        {/* Enter description */}
                        <TextField
                            fullWidth
                            id="description"
                            name="description"
                            label="Description"
                            value={formik.values.description}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.description && Boolean(formik.errors.description)}
                            helperText={(formik.touched.description && formik.errors.description) ?? "Enter description (optional)"}
                        />
                        {/* Action buttons */}
                        <Grid container spacing={1}>
                            <Grid item xs={6}>
                                <Button
                                    fullWidth
                                    type="submit"
                                    startIcon={<AddIcon />}
                                >Create</Button>
                            </Grid>
                            <Grid item xs={6}>
                                <Button
                                    fullWidth
                                    onClick={handleClose}
                                    startIcon={<CancelIcon />}
                                >Cancel</Button>
                            </Grid>
                        </Grid>
                    </Stack>
                </form>
            </DialogContent>
        </Dialog>
    )
}