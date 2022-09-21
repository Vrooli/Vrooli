import { Checkbox, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material";
import { useMutation } from "@apollo/client";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { routineCreateForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { routineCreateMutation } from "graphql/mutation";
import { getUserLanguages, InputShape, ObjectType, OutputShape, parseSearchParams, RoutineTranslationShape, shapeRoutineCreate, TagShape, updateArray } from "utils";
import { RoutineCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, MarkdownInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession, VersionInput } from "components";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { InputOutputContainer } from "components/lists/inputOutput";
import { routineCreate, routineCreateVariables } from "graphql/generated/routineCreate";
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

    // Handle translations
    type Translation = RoutineTranslationShape;
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        setTranslations([...translations.filter(t => t.language !== language)]);
        // Also delete translations from inputs and outputs
        setInputsList(inputsList.map(i => {
            const updatedTranslationsList = i.translations.filter(t => t.language !== language);
            return { ...i, translations: updatedTranslationsList };
        }));
        setOutputsList(outputsList.map(o => {
            const updatedTranslationsList = o.translations.filter(t => t.language !== language);
            return { ...o, translations: updatedTranslationsList };
        }));
    }, [translations, inputsList, outputsList]);
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
        const params = parseSearchParams(window.location.search);
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useMutation<routineCreate, routineCreateVariables>(routineCreateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            instructions: 'Fill out the form below.',
            isInternal: false,
            title: '',
            version: '1.0.0',
        },
        validationSchema,
        onSubmit: (values) => {
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                instructions: values.instructions,
                title: values.title,
            })
            mutationWrapper({
                mutation,
                input: shapeRoutineCreate({
                    id: uuid(),
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
                    translations: allTranslations,
                }),
                onSuccess: (response) => { onCreated(response.data.routineCreate) },
                onError: () => { formik.setSubmitting(false) }
            })
        },
    });

    /**
     * On page leave, check if unsaved work. 
     * If so, prompt for confirmation.
     */
    useEffect(() => {
        const handleBeforeUnload = (e: BeforeUnloadEvent) => {
            if (formik.dirty) {
                e.preventDefault()
                e.returnValue = ''
            }
        };
        window.addEventListener('beforeunload', handleBeforeUnload);
        return () => window.removeEventListener('beforeunload', handleBeforeUnload);
    }, [formik.dirty]);

    // Handle languages
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const [languages, setLanguages] = useState<string[]>([getUserLanguages(session)[0]]);
    useEffect(() => {
        if (languages.length === 0) {
            const userLanguage = getUserLanguages(session)[0]
            setLanguage(userLanguage)
            setLanguages([userLanguage])
        }
    }, [languages, session, setLanguage, setLanguages])
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
            instructions: existingTranslation?.instructions ?? '',
            title: existingTranslation?.title ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            instructions: formik.values.instructions,
            title: formik.values.title,
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, formik.values.instructions, formik.values.title, updateFormikTranslation]);
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
                        value={formik.values.title}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.title && Boolean(formik.errors.title)}
                        helperText={formik.touched.title && formik.errors.title}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="description"
                        name="description"
                        label="description"
                        value={formik.values.description}
                        multiline
                        maxRows={3}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.description && Boolean(formik.errors.description)}
                        helperText={formik.touched.description && formik.errors.description}
                    />
                </Grid>
                <Grid item xs={12}>
                    <MarkdownInput
                        id="instructions"
                        placeholder="Instructions"
                        value={formik.values.instructions}
                        minRows={4}
                        onChange={(newText: string) => formik.setFieldValue('instructions', newText)}
                        error={formik.touched.instructions && Boolean(formik.errors.instructions)}
                        helperText={formik.touched.instructions ? formik.errors.instructions : null}
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
                    disabledCancel={formik.isSubmitting}
                    disabledSubmit={Boolean(!isLoggedIn || formik.isSubmitting)}
                    errors={formik.errors}
                    isCreate={true}
                    onCancel={onCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </Grid>
        </form>
    )
}