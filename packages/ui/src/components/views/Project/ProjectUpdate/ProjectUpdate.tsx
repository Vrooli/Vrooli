import { Box, CircularProgress, Grid, TextField, Typography } from "@mui/material"
import { useMutation, useLazyQuery } from "@apollo/client";
import { project, projectVariables } from "graphql/generated/project";
import { projectQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { projectUpdateForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { projectUpdateMutation } from "graphql/mutation";
import { getLastUrlPart, ObjectType, ProjectTranslationShape, PubSub, shapeProjectUpdate, TagShape, updateArray } from "utils";
import { GridSubmitButtons, LanguageInput, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession } from "components";
import { ResourceList } from "types";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { projectUpdate, projectUpdateVariables } from "graphql/generated/projectUpdate";
import { ProjectUpdateProps } from "../types";
import { RelationshipsObject } from "components/inputs/types";

export const ProjectUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: ProjectUpdateProps) => {
    // Fetch existing data
    const id = useMemo(() => getLastUrlPart(), []);
    const [getData, { data, loading }] = useLazyQuery<project, projectVariables>(projectQuery);
    useEffect(() => {
        if (uuidValidate(id)) getData({ variables: { input: { id } } });
    }, [getData, id])
    const project = useMemo(() => data?.project, [data]);

    const [relationships, setRelationships] = useState<RelationshipsObject>({
        isComplete: false,
        isPrivate: false,
        owner: userFromSession(session),
        parent: null,
        project: null,
    });
    const onRelationshipsChange = useCallback((newRelationshipsObject: Partial<RelationshipsObject>) => {
        setRelationships({
            ...relationships,
            ...newRelationshipsObject,
        });
    }, [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const addTag = useCallback((tag: TagShape) => {
        setTags(t => [...t, tag]);
    }, [setTags]);
    const removeTag = useCallback((tag: TagShape) => {
        setTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setTags]);
    const clearTags = useCallback(() => {
        setTags([]);
    }, [setTags]);

    // Handle translations
    type Translation = ProjectTranslationShape;
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
        setRelationships({
            isComplete: project?.isComplete ?? false,
            isPrivate: project?.isPrivate ?? false,
            owner: null,
            // owner: project?.owner ?? null, TODO
            parent: null,
            // parent: project?.parent ?? null, TODO
            project: null,
        });
        setResourceList(project?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
        setTags(project?.tags ?? []);
        setTranslations(project?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            description: t.description ?? '',
            name: t.name ?? '',
        })) ?? []);
    }, [project]);

    // Handle update
    const [mutation] = useMutation<projectUpdate, projectUpdateVariables>(projectUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            name: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            if (!project) {
                PubSub.get().publishSnack({ message: 'Could not find existing project data.', severity: 'error' });
                return;
            }
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                name: values.name,
            })
            mutationWrapper({
                mutation,
                input: shapeProjectUpdate(project, {
                    id: project.id,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    owner: relationships.owner,
                    parent: relationships.parent,
                    resourceLists: [resourceList],
                    tags: tags,
                    translations: allTranslations,
                }),
                onSuccess: (response) => { onUpdated(response.data.projectUpdate) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    // Handle languages
    const [language, setLanguage] = useState<string>('');
    const [languages, setLanguages] = useState<string[]>([]);
    useEffect(() => {
        if (languages.length === 0 && translations.length > 0) {
            setLanguage(translations[0].language);
            setLanguages(translations.map(t => t.language));
            formik.setValues({
                ...formik.values,
                description: translations[0].description ?? '',
                name: translations[0].name,
            })
        }
    }, [formik, languages, setLanguage, setLanguages, translations])
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
            name: existingTranslation?.name ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            name: formik.values.name,
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, formik.values.name, updateFormikTranslation]);
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

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <Typography
                    component="h1"
                    variant="h3"
                    sx={{
                        textAlign: 'center',
                        sx: { marginTop: 2, marginBottom: 2 },
                    }}
                >Update Project</Typography>
            </Grid>
            <Grid item xs={12}>
                <RelationshipButtons
                    objectType={ObjectType.Project}
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
                    handleCurrent={handleLanguageSelect}
                    selectedLanguages={languages}
                    session={session}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="name"
                    name="name"
                    label="Name"
                    value={formik.values.name}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.name && Boolean(formik.errors.name)}
                    helperText={formik.touched.name && formik.errors.name}
                />
            </Grid>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="description"
                    name="description"
                    label="description"
                    multiline
                    minRows={4}
                    value={formik.values.description}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.description && Boolean(formik.errors.description)}
                    helperText={formik.touched.description && formik.errors.description}
                />
            </Grid>
            <Grid item xs={12}>
                <ResourceListHorizontal
                    title={'Resources'}
                    list={resourceList}
                    canEdit={true}
                    handleUpdate={handleResourcesUpdate}
                    loading={loading}
                    session={session}
                    mutate={false}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12}>
                <TagSelector
                    session={session}
                    tags={tags}
                    onTagAdd={addTag}
                    onTagRemove={removeTag}
                    onTagsClear={clearTags}
                />
            </Grid>
            <GridSubmitButtons
                disabledCancel={formik.isSubmitting}
                disabledSubmit={formik.isSubmitting || !formik.isValid}
                errors={formik.errors}
                isCreate={false}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid>
    ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, handleLanguageSelect, languages, formik.values.name, formik.values.description, formik.handleBlur, formik.handleChange, formik.touched.name, formik.touched.description, formik.errors, formik.isSubmitting, formik.isValid, formik.setSubmitting, formik.handleSubmit, resourceList, handleResourcesUpdate, loading, tags, addTag, removeTag, clearTags, onCancel]);


    return (
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
    )
}