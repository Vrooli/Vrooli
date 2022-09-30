import { Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { projectCreate as validationSchema, projectTranslationCreate } from '@shared/validation';
import { useFormik } from 'formik';
import { projectCreateMutation } from "graphql/mutation";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, ObjectType, parseSearchParams, removeTranslation, shapeProjectCreate, TagShape, usePromptBeforeUnload } from "utils";
import { ProjectCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession } from "components";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { projectCreate, projectCreateVariables } from "graphql/generated/projectCreate";
import { RelationshipsObject } from "components/inputs/types";

export const ProjectCreate = ({
    onCreated,
    onCancel,
    session,
    zIndex,
}: ProjectCreateProps) => {

    const [relationships, setRelationships] = useState<RelationshipsObject>({
        isComplete: false,
        isPrivate: false,
        owner: userFromSession(session),
        parent: null,
        project: null,
    });
    const onRelationshipsChange = useCallback((newRelationshipsObject: Partial<RelationshipsObject>) => {
        console.log('onRelationshipsChange', newRelationshipsObject);
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
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // TODO upgrade to pull data from search params like it's done in AdvancedSearchDialog
    useEffect(() => {
        const params = parseSearchParams(window.location.search);
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle languages
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const [languages, setLanguages] = useState<string[]>([getUserLanguages(session)[0]]);

    // Handle create
    const [mutation] = useMutation<projectCreate, projectCreateVariables>(projectCreateMutation);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            translationsCreate: [{
                id: uuid(),
                language,
                name: '',
                description: '',
            }],
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: shapeProjectCreate({
                    id: uuid(),
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    owner: relationships.owner,
                    parent: relationships.parent,
                    resourceLists: [resourceList],
                    tags: tags,
                    translations: values.translationsCreate,
                }),
                onSuccess: (response) => { onCreated(response.data.projectCreate) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Current name and description info, as well as errors
    const { description, name, errorDescription, errorName, touchedDescription, touchedName, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsCreate', language);
        return {
            description: value?.description ?? '',
            name: value?.name ?? '',
            errorDescription: error?.description ?? '',
            errorName: error?.name ?? '',
            touchedDescription: touched?.description ?? false,
            touchedName: touched?.name ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsCreate', projectTranslationCreate),
        }
    }, [formik, language]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: React.FocusEvent<HTMLInputElement>) => {
        handleTranslationBlur(formik, 'translationsCreate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        handleTranslationChange(formik, 'translationsCreate', e, language)
    }, [formik, language]);

    // Handle languages
    const handleLanguageSelect = useCallback((newLanguage: string) => { setLanguage(newLanguage) }, []);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
        addEmptyTranslation(formik, 'translationsCreate', newLanguage);
    }, [formik, handleLanguageSelect, languages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        setLanguages(newLanguages);
        removeTranslation(formik, 'translationsCreate', language);
    }, [formik, languages]);

    const isLoggedIn = useMemo(() => session?.isLoggedIn === true && uuidValidate(session?.id ?? ''), [session]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex,
        }}
        >
            <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                <Grid item xs={12}>
                    <PageTitle title="Create Project" />
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
                        value={name}
                        onBlur={onTranslationBlur}
                        onChange={onTranslationChange}
                        error={touchedName && Boolean(errorName)}
                        helperText={touchedName && errorName}
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
                        loading={false}
                        session={session}
                        mutate={false}
                        zIndex={zIndex}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TagSelector
                        handleTagsUpdate={handleTagsUpdate}
                        session={session}
                        tags={tags}
                    />
                </Grid>
                <GridSubmitButtons
                    disabledCancel={formik.isSubmitting}
                    disabledSubmit={Boolean(!isLoggedIn || formik.isSubmitting)}
                    errors={errors}
                    isCreate={true}
                    onCancel={onCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </Grid>
        </form>
    )
}