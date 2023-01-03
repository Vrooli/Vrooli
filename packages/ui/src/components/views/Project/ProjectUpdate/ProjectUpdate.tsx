import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useMutation, useLazyQuery } from "graphql/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils';
import { projectValidation, projectVersionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, base36ToUuid, getFormikErrorsWithTranslations, getLastUrlPart, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, PubSub, removeTranslation, shapeProjectUpdate, TagShape, usePromptBeforeUnload } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, SnackSeverity, TagSelector, userFromSession } from "components";
import { DUMMY_ID, uuid, uuidValidate } from '@shared/uuid';
import { ProjectUpdateProps } from "../types";
import { RelationshipsObject } from "components/inputs/types";
import { projectEndpoint } from "graphql/endpoints";
import { FindByIdInput, Project, ProjectUpdateInput, ResourceList } from "@shared/consts";

export const ProjectUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: ProjectUpdateProps) => {
    // Fetch existing data
    const id = useMemo(() => base36ToUuid(getLastUrlPart()), []);
    const [getData, { data, loading }] = useLazyQuery<Project, FindByIdInput, 'project'>(...projectEndpoint.findOne);
    useEffect(() => { uuidValidate(id) && getData({ variables: { id } }) }, [getData, id])
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
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid() } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

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
        setResourceList(project?.resourceList ?? { id: uuid() } as any);
        setTags(project?.tags ?? []);
    }, [project]);

    // Handle update
    const [mutation] = useMutation<Project, ProjectUpdateInput, 'projectUpdate'>(...projectEndpoint.update);
    const formik = useFormik({
        initialValues: {
            id: project?.id ?? uuid(),
            translationsUpdate: project?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                name: '',
                description: '',
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: projectValidation.update(),
        onSubmit: (values) => {
            if (!project) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadProject', severity: SnackSeverity.Error });
                return;
            }
            mutationWrapper<Project, ProjectUpdateInput>({
                mutation,
                input: shapeProjectUpdate(project, {
                    id: project.id,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    owner: relationships.owner,
                    parent: relationships.parent,
                    resourceLists: [resourceList],
                    tags: tags,
                    translations: values.translationsUpdate.map(t => ({
                        ...t,
                        id: t.id === DUMMY_ID ? uuid() : t.id,
                    })),
                }),
                onSuccess: (data) => { onUpdated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const { description, name, errorDescription, errorName, touchedDescription, touchedName, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            description: value?.description ?? '',
            name: value?.name ?? '',
            errorDescription: error?.description ?? '',
            errorName: error?.name ?? '',
            touchedDescription: touched?.description ?? false,
            touchedName: touched?.name ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', projectVersionTranslationValidation.update()),
        }
    }, [formik, language]);
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
                <PageTitle titleKey='UpdateProject' session={session} />
            </Grid>
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
                    value={name}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={touchedName && Boolean(errorName)}
                    helperText={touchedName && errorName}
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
                    value={description}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={touchedDescription && Boolean(errorDescription)}
                    helperText={touchedDescription && errorDescription}
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
            <Grid item xs={12} mb={4}>
                <TagSelector
                    handleTagsUpdate={handleTagsUpdate}
                    session={session}
                    tags={tags}
                />
            </Grid>
            <GridSubmitButtons
                errors={errors}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid>
    ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, formik.values.translationsUpdate, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, name, onTranslationBlur, onTranslationChange, touchedName, errorName, description, touchedDescription, errorDescription, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, errors, onCancel]);


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