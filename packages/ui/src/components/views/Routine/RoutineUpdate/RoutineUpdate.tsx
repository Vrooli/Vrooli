import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useMutation, useLazyQuery } from "@apollo/client";
import { routine, routineVariables } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { RoutineUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { routineTranslationUpdate, routineUpdate as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { routineUpdateMutation } from "graphql/mutation";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getLastUrlPart, getTranslationData, handleTranslationBlur, handleTranslationChange, InputShape, ObjectType, OutputShape, PubSub, removeTranslation, shapeRoutineUpdate, TagShape, usePromptBeforeUnload } from "utils";
import { GridSubmitButtons, LanguageInput, MarkdownInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession, VersionInput } from "components";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { ResourceList, RoutineTranslation } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { InputOutputContainer } from "components/lists/inputOutput";
import { routineUpdate, routineUpdateVariables } from "graphql/generated/routineUpdate";
import { RelationshipItemRoutine, RelationshipsObject } from "components/inputs/types";

export const RoutineUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: RoutineUpdateProps) => {
    // Fetch existing data
    const id = useMemo(() => getLastUrlPart(), []);
    const [getData, { data, loading }] = useLazyQuery<routine, routineVariables>(routineQuery);
    useEffect(() => {
        if (uuidValidate(id)) getData({ variables: { input: { id } } });
    }, [getData, id])
    const routine = useMemo(() => data?.routine, [data]);

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

    // Handle inputs
    const [inputsList, setInputsList] = useState<InputShape[]>([]);
    const handleInputsUpdate = useCallback((updatedList: InputShape[]) => {
        setInputsList(updatedList);
    }, [setInputsList]);

    // Handle outputs
    const [outputsList, setOutputsList] = useState<OutputShape[]>([]);
    const handleOutputsUpdate = useCallback((updatedList: OutputShape[]) => {
        setOutputsList(updatedList);
    }, [setOutputsList]);

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
            isComplete: routine?.isComplete ?? false,
            isPrivate: routine?.isPrivate ?? false,
            owner: routine?.owner ?? null,
            parent: null,
            // parent: routine?.parent ?? null, TODO
            project: null, // TODO
        });
        setInputsList(routine?.inputs ?? []);
        setOutputsList(routine?.outputs ?? []);
        setResourceList(routine?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
        setTags(routine?.tags ?? []);
    }, [routine]);

    // Handle languages
    const [language, setLanguage] = useState<string>('');
    const [languages, setLanguages] = useState<string[]>([]);

    // Handle update
    const [mutation] = useMutation<routineUpdate, routineUpdateVariables>(routineUpdateMutation);
    const formik = useFormik({
        initialValues: {
            translationsUpdate: (routine?.translations ?? []) as RoutineTranslation[],
            version: routine?.version ?? '1.0.0',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: validationSchema({ minVersion: routine?.version ?? '0.0.1' }),
        onSubmit: (values) => {
            if (!routine) {
                PubSub.get().publishSnack({ message: 'Could not find existing routine data.', severity: 'error' });
                return;
            }
            mutationWrapper({
                mutation,
                input: shapeRoutineUpdate(routine, {
                    id: routine.id,
                    version: values.version,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    owner: relationships.owner,
                    parent: relationships.parent as RelationshipItemRoutine | null,
                    project: relationships.project,
                    inputs: inputsList,
                    outputs: outputsList,
                    resourceLists: [resourceList],
                    tags: tags,
                    translations: values.translationsUpdate,
                }),
                onSuccess: (response) => { onUpdated(response.data.routineUpdate) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Current description, instructions, and title info, as well as errors
    const { description, instructions, title, errorDescription, errorInstructions, errorTitle, touchedDescription, touchedInstructions, touchedTitle, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            description: value?.description ?? '',
            instructions: value?.instructions ?? '',
            title: value?.title ?? '',
            errorDescription: error?.description ?? '',
            errorInstructions: error?.instructions ?? '',
            errorTitle: error?.title ?? '',
            touchedDescription: touched?.description ?? false,
            touchedInstructions: touched?.instructions ?? false,
            touchedTitle: touched?.title ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', routineTranslationUpdate),
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
                <PageTitle title="Update Routine" />
            </Grid>
            <Grid item xs={12}>
                <RelationshipButtons
                    isFormDirty={formik.dirty}
                    objectType={ObjectType.Routine}
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
                    id="title"
                    name="title"
                    label="title"
                    value={title}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={touchedTitle && Boolean(errorTitle)}
                    helperText={touchedTitle && errorTitle}
                />
            </Grid>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="description"
                    name="description"
                    label="description"
                    value={description}
                    multiline
                    maxRows={3}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={touchedDescription && Boolean(errorDescription)}
                    helperText={touchedDescription && errorDescription}
                />
            </Grid>
            <Grid item xs={12}>
                <MarkdownInput
                    id="instructions"
                    placeholder="Instructions"
                    value={instructions}
                    minRows={4}
                    onChange={(newText: string) => onTranslationChange({ target: { name: 'instructions', value: newText }})}
                    error={touchedInstructions && Boolean(errorInstructions)}
                    helperText={touchedInstructions ? errorInstructions : null}
                />
            </Grid>
            <Grid item xs={12}>
                <VersionInput
                    fullWidth
                    id="version"
                    name="version"
                    value={formik.values.version}
                    onBlur={formik.handleBlur}
                    onChange={(newVersion: string) => {
                        formik.setFieldValue('version', newVersion);
                        setRelationships({
                            ...relationships,
                            isComplete: false,
                        })
                    }}
                    error={formik.touched.version && Boolean(formik.errors.version)}
                    helperText={formik.touched.version ? formik.errors.version : null}
                />
            </Grid>
            <Grid item xs={12}>
                <InputOutputContainer
                    isEditing={true}
                    handleUpdate={handleInputsUpdate}
                    isInput={true}
                    language={language}
                    list={inputsList}
                    session={session}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12}>
                <InputOutputContainer
                    isEditing={true}
                    handleUpdate={handleOutputsUpdate}
                    isInput={false}
                    language={language}
                    list={outputsList}
                    session={session}
                    zIndex={zIndex}
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
    ), [formik, onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, handleLanguageSelect, languages, title, onTranslationBlur, onTranslationChange, touchedTitle, errorTitle, description, touchedDescription, errorDescription, instructions, touchedInstructions, errorInstructions, handleInputsUpdate, inputsList, handleOutputsUpdate, outputsList, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, errors, onCancel]);

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