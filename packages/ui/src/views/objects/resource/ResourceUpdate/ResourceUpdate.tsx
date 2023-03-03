import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ResourceUpdateProps } from "../types";
import { resourceValidation, resourceTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, defaultResourceList, getPreferredLanguage, getUserLanguages, handleTranslationBlur, handleTranslationChange, parseSingleItemUrl, PubSub, removeTranslation, shapeResource, TagShape, usePromptBeforeUnload, useTranslatedFields, useUpdateActions } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, TagSelector, TopBar } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindByIdInput, Resource, ResourceUpdateInput, ResourceList } from "@shared/consts";
import { resourceFindOne } from "api/generated/endpoints/resource_findOne";
import { resourceUpdate } from "api/generated/endpoints/resource_update";

export const ResourceUpdate = ({
    display = 'dialog',
    listId,
    session,
    zIndex = 200,
}: ResourceUpdateProps) => {
    const { onCancel, onUpdated } = useUpdateActions<Resource>();

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: resource, loading }] = useCustomLazyQuery<Resource, FindByIdInput>(resourceFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    // Handle update
    const [mutation] = useCustomMutation<Resource, ResourceUpdateInput>(resourceUpdate);
    const formik = useFormik({
        initialValues: {
            id: resource?.id ?? uuid(),
            translationsUpdate: resource?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                name: '',
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: resourceValidation.update({}),
        onSubmit: (values) => {
            if (!resource) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadResource', severity: 'Error' });
                return;
            }
            // mutationWrapper<Resource, ResourceUpdateInput>({
            //     mutation,
            //     input: shapeResource.update(resource, {
            //         id: resource.id,
            //         isOpenToNewMembers: values.isOpenToNewMembers,
            //         isPrivate: relationships.isPrivate,
            //         resourceList: resourceList,
            //         tags: tags,
            //         translations: values.translationsUpdate,
            //     }),
            //     onSuccess: (data) => { onUpdated(data) },
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const translations = useTranslatedFields({
        fields: ['description', 'name'],
        formik,
        formikField: 'translationsUpdate',
        language,
        validationSchema: resourceTranslationValidation.update({}),
    });
    const languages = useMemo(() => formik.values.translationsUpdate.map(t => t.language), [formik.values.translationsUpdate]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(formik, 'translationsUpdate', newLanguage);
    }, [formik]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        removeTranslation(formik, 'translationsUpdate', language);
    }, [formik, languages]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => handleTranslationBlur(formik, 'translationsUpdate', e, language), [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => handleTranslationChange(formik, 'translationsUpdate', e, language), [formik, language]);

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <LanguageInput
                    currentLanguage={language}
                    handleAdd={handleAddLanguage}
                    handleDelete={handleLanguageDelete}
                    handleCurrent={setLanguage}
                    session={session}
                    translations={formik.values.translationsUpdate}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="name"
                    name="name"
                    label="Name"
                    value={translations.name}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={translations.touchedName && Boolean(translations.errorName)}
                    helperText={translations.touchedName && translations.errorName}
                />
            </Grid>
            <Grid item xs={12} mb={4}>
                <TextField
                    fullWidth
                    id="description"
                    name="description"
                    label="Description"
                    multiline
                    minRows={4}
                    value={translations.description}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={translations.touchedDescription && Boolean(translations.errorDescription)}
                    helperText={translations.touchedDescription && translations.errorDescription}
                />
            </Grid>
            <GridSubmitButtons
                display={display}
                errors={translations.errorsWithTranslations}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid>
    ), [language, handleAddLanguage, handleLanguageDelete, session, formik.values.translationsUpdate, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, zIndex, translations.name, translations.touchedName, translations.errorName, translations.description, translations.touchedDescription, translations.errorDescription, translations.errorsWithTranslations, onTranslationBlur, onTranslationChange, display, onCancel]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'UpdateResource',
                }}
            />
            <form onSubmit={formik.handleSubmit} style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
            }}
            >
                {loading ? (
                    <Box sx={{
                        position: 'absolute',
                        top: '-5vh', // Half of toolbar height
                        width: '100%',
                        height: '100%',
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center',
                    }}>
                        <CircularProgress size={100} color="secondary" />
                    </Box>
                ) : formInput}
            </form>
        </>
    )
}