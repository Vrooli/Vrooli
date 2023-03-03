import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useCustomMutation, useCustomLazyQuery } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from 'api/utils';
import { projectValidation, projectVersionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, getUserLanguages, handleTranslationBlur, handleTranslationChange, parseSingleItemUrl, PubSub, removeTranslation, shapeProjectVersion, TagShape, usePromptBeforeUnload, useTranslatedFields, useUpdateActions } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, TagSelector, TopBar } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { ProjectUpdateProps } from "../types";
import { RelationshipsObject } from "components/inputs/types";
import { FindVersionInput, ProjectVersion, ProjectVersionUpdateInput } from "@shared/consts";
import { projectVersionFindOne } from "api/generated/endpoints/projectVersion_findOne";
import { projectVersionUpdate } from "api/generated/endpoints/projectVersion_update";

export const ProjectUpdate = ({
    display = 'page',
    session,
    zIndex = 200,
}: ProjectUpdateProps) => {
    const { onCancel, onUpdated } = useUpdateActions<ProjectVersion>();

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: projectVersion, loading }] = useCustomLazyQuery<ProjectVersion, FindVersionInput>(projectVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: projectVersion?.isComplete ?? false,
            isPrivate: projectVersion?.isPrivate ?? false,
            owner: projectVersion?.root?.owner ?? null,
            parent: projectVersion?.root?.parent ?? null,
            project: null,
        });
        setTags(projectVersion?.root?.tags ?? []);
    }, [projectVersion]);

    // Handle update
    const [mutation] = useCustomMutation<ProjectVersion, ProjectVersionUpdateInput>(projectVersionUpdate);
    const formik = useFormik({
        initialValues: {
            id: projectVersion?.id ?? uuid(),
            translationsUpdate: projectVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                name: '',
                description: '',
            }],
            versionInfo: {
                versionIndex: projectVersion?.root?.versions?.length ?? 0,
                versionLabel: projectVersion?.versionLabel ?? '1.0.0',
                versionNotes: '',
            }
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: projectValidation.update({}),
        onSubmit: (values) => {
            if (!projectVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadProject', severity: 'Error' });
                return;
            }
            mutationWrapper<ProjectVersion, ProjectVersionUpdateInput>({
                mutation,
                input: shapeProjectVersion.update(projectVersion, {
                    id: projectVersion.id,
                    isComplete: relationships.isComplete,
                    isLatest: true,
                    isPrivate: relationships.isPrivate,
                    root: {
                        id: projectVersion.root.id,
                        isPrivate: relationships.isPrivate,
                        owner: relationships.owner,
                        permissions: JSON.stringify({}),
                        tags: tags,
                    },
                    translations: values.translationsUpdate,
                    ...values.versionInfo,
                }),
                onSuccess: (data) => { onUpdated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
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
        validationSchema: projectVersionTranslationValidation.update({}),
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
            <Grid item xs={12} mb={4}>
                <RelationshipButtons
                    isEditing={true}
                    objectType={'Project'}
                    onRelationshipsChange={onRelationshipsChange}
                    relationships={relationships}
                    session={session}
                    zIndex={zIndex}
                />
            </Grid>
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
                    label="description"
                    multiline
                    minRows={4}
                    value={translations.description}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={translations.touchedDescription && Boolean(translations.errorDescription)}
                    helperText={translations.touchedDescription && translations.errorDescription}
                />
            </Grid>
            <Grid item xs={12} mb={4}>
                <TagSelector
                    handleTagsUpdate={handleTagsUpdate}
                    session={session}
                    tags={tags}
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
    ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, formik.values.translationsUpdate, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, onTranslationBlur, onTranslationChange, translations, handleTagsUpdate, tags, onCancel]);


    return (
        <>
           <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'UpdateProject',
                }}
            />
            <form onSubmit={formik.handleSubmit} style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                zIndex,
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