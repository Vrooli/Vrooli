import { Checkbox, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material";
import { useMutation } from "@apollo/client";
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { routineCreate as validationSchema, routineTranslationCreate } from '@shared/validation';
import { useFormik } from 'formik';
import { routineCreateMutation } from "graphql/mutation";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, InputShape, ObjectType, OutputShape, parseSearchParams, removeTranslation, shapeRoutineCreate, TagShape, usePromptBeforeUnload } from "utils";
import { RoutineCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, MarkdownInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession, VersionInput } from "components";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { uuid, uuidValidate } from '@shared/uuid';
import { InputOutputContainer } from "components/lists/inputOutput";
import { routineCreateVariables, routineCreate_routineCreate } from "graphql/generated/routineCreate";
import { RelationshipItemRoutine, RelationshipsObject } from "components/inputs/types";

export const RoutineCreate = ({
    isSubroutine = false,
    onCreated,
    onCancel,
    session,
    zIndex,
}: RoutineCreateProps) => {

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
        const params = parseSearchParams(window.location.search);
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useMutation(routineCreateMutation);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            isInternal: false,
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                description: '',
                instructions: '',
                title: '',
            }],
            version: '1.0.0',
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper<routineCreate_routineCreate, routineCreateVariables>({
                mutation,
                input: shapeRoutineCreate({
                    id: values.id,
                    version: values.version,
                    isInternal: values.isInternal,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    owner: relationships.owner,
                    parent: relationships.parent as RelationshipItemRoutine | null,
                    project: relationships.project,
                    inputs: inputsList,
                    outputs: outputsList,
                    resourceLists: [resourceList],
                    tags: tags,
                    translations: values.translationsCreate,
                }),
                onSuccess: (data) => { onCreated(data) },
                onError: () => { formik.setSubmitting(false) }
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const { description, instructions, title, errorDescription, errorInstructions, errorTitle, touchedDescription, touchedInstructions, touchedTitle, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsCreate', language);
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
            errors: getFormikErrorsWithTranslations(formik, 'translationsCreate', routineTranslationCreate),
        }
    }, [formik, language]);
    const languages = useMemo(() => formik.values.translationsCreate.map(t => t.language), [formik.values.translationsCreate]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(formik, 'translationsCreate', newLanguage);
    }, [formik]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        removeTranslation(formik, 'translationsCreate', language);
    }, [formik, languages]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsCreate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsCreate', e, language)
    }, [formik, language]);

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
                    <PageTitle title="Create Routine" />
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
                        handleCurrent={setLanguage}
                        session={session}
                        translations={formik.values.translationsCreate}
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
                        onBlur={onTranslationBlur}
                        onChange={onTranslationChange}
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
                {/* Is internal checkbox */}
                {isSubroutine && (
                    <Grid item xs={12}>
                        <Tooltip placement={'top'} title='Indicates if this routine is meant to be a subroutine for only one other routine. If so, it will not appear in search resutls.'>
                            <FormControlLabel
                                label='Internal'
                                control={
                                    <Checkbox
                                        id='routine-info-dialog-is-internal'
                                        size="small"
                                        name='isInternal'
                                        color='secondary'
                                        checked={formik.values.isInternal}
                                        onChange={formik.handleChange}
                                    />
                                }
                            />
                        </Tooltip>
                    </Grid>
                )}
                <GridSubmitButtons
                    disabledSubmit={!isLoggedIn}
                    errors={errors}
                    isCreate={true}
                    loading={formik.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </Grid>
        </form>
    )
}