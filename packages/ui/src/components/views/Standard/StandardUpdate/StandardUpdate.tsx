import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { ResourceListUsedFor } from "@shared/consts";
import { useMutation, useLazyQuery } from "@apollo/client";
import { standard, standardVariables } from "graphql/generated/standard";
import { standardQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { StandardUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { standardTranslationUpdate, standardUpdate as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { standardUpdateMutation } from "graphql/mutation";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getLastUrlPart, getTranslationData, handleTranslationBlur, handleTranslationChange, ObjectType, PubSub, removeTranslation, shapeStandardUpdate, TagShape, usePromptBeforeUnload } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession } from "components";
import { ResourceList, StandardTranslation } from "types";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { standardUpdate, standardUpdateVariables } from "graphql/generated/standardUpdate";
import { RelationshipsObject } from "components/inputs/types";

export const StandardUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: StandardUpdateProps) => {
    // Fetch existing data
    const id = useMemo(() => getLastUrlPart(), []);
    const [getData, { data, loading }] = useLazyQuery<standard, standardVariables>(standardQuery);
    useEffect(() => {
        if (uuidValidate(id)) getData({ variables: { input: { id } } });
    }, [getData, id])
    const standard = useMemo(() => data?.standard, [data]);

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
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: false, //TODO
            isPrivate: standard?.isPrivate ?? false,
            owner: standard?.creator ?? null,
            parent: null,
            // parent: standard?.parent ?? null, TODO
            project: null // TODO
        });
        setResourceList(standard?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
        setTags(standard?.tags ?? []);
    }, [standard]);

    // Handle languages
    const [language, setLanguage] = useState<string>('');
    const [languages, setLanguages] = useState<string[]>([]);

    // Handle update
    const [mutation] = useMutation<standardUpdate, standardUpdateVariables>(standardUpdateMutation);
    const formik = useFormik({
        initialValues: {
            translationsUpdate: (standard?.translations ?? []) as StandardTranslation[],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            if (!standard) {
                PubSub.get().publishSnack({ message: 'Could not find existing standard data.', severity: 'error' });
                return;
            }
            // Update
            mutationWrapper({
                mutation,
                input: shapeStandardUpdate(standard, {
                    id: standard.id,
                    resourceLists: [resourceList],
                    tags: tags,
                    translations: values.translationsUpdate,
                }),
                onSuccess: (response) => { onUpdated(response.data.standardUpdate) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Current description info, as well as errors
    const { description, errorDescription, touchedDescription, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            description: value?.description ?? '',
            errorDescription: error?.description ?? '',
            touchedDescription: touched?.description ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', standardTranslationUpdate),
        }
    }, [formik, language]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsUpdate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsUpdate', e, language)
    }, [formik, language]);

    // Handle languages
    useEffect(() => {
        if (languages.length === 0 && formik.values.translationsUpdate.length > 0) {
            setLanguage(formik.values.translationsUpdate[0].language);
            setLanguages(formik.values.translationsUpdate.map(t => t.language));
        }
    }, [formik, languages, setLanguage, setLanguages])
    const handleLanguageSelect = useCallback((newLanguage: string) => { setLanguage(newLanguage) }, []);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
        addEmptyTranslation(formik, 'translationsUpdate', newLanguage);
    }, [formik, handleLanguageSelect, languages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        setLanguages(newLanguages);
        removeTranslation(formik, 'translationsUpdate', language);
    }, [formik, languages]);

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <PageTitle title="Update Standard" />
            </Grid>
            <Grid item xs={12}>
                <RelationshipButtons
                    objectType={ObjectType.Standard}
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
            {/* TODO versioning */}
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
            <Grid item xs={12} marginBottom={4}>
                <TagSelector
                    handleTagsUpdate={handleTagsUpdate}
                    session={session}
                    tags={tags}
                />
            </Grid>
            <GridSubmitButtons
                disabledCancel={formik.isSubmitting}
                disabledSubmit={formik.isSubmitting || !formik.isValid}
                errors={errors}
                isCreate={false}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid>
    ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, handleLanguageSelect, languages, description, onTranslationBlur, onTranslationChange, touchedDescription, errorDescription, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, formik.isSubmitting, formik.isValid, formik.setSubmitting, formik.handleSubmit, errors, onCancel]);


    return (
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
    )
}