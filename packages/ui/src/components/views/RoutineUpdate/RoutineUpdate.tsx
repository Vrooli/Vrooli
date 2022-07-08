import { Box, Checkbox, CircularProgress, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useLazyQuery } from "@apollo/client";
import { routine, routineVariables } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { RoutineUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { routineUpdateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { routineUpdateMutation } from "graphql/mutation";
import { formatForUpdate, InputShape, OutputShape, shapeTagsUpdate, updateArray } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { TagSelectorTag } from "components/inputs/types";
import { DialogActionItem } from "components/containers/types";
import { LanguageInput, MarkdownInput, ResourceListHorizontal, TagSelector, UserOrganizationSwitch } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { v4 as uuid } from 'uuid';
import { NewObject, Organization, ResourceList, Routine } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { InputOutputContainer } from "components/lists/inputOutput";
import { routineUpdate, routineUpdateVariables } from "graphql/generated/routineUpdate";

export const RoutineUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: RoutineUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Routine}/edit/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchRoutines}/edit/:id`);
    const id = params?.id ?? params2?.id;
    // Fetch existing data
    const [getData, { data, loading }] = useLazyQuery<routine, routineVariables>(routineQuery);
    useEffect(() => {
        if (id) getData({ variables: { input: { id } } });
    }, [getData, id])
    const routine = useMemo(() => data?.routine, [data]);

    // Handle user/organization switch
    const [organizationFor, setOrganizationFor] = useState<Organization | null>(null);
    const onSwitchChange = useCallback((organization: Organization | null) => { setOrganizationFor(organization) }, [setOrganizationFor]);

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
    const [tags, setTags] = useState<TagSelectorTag[]>([]);
    const addTag = useCallback((tag: TagSelectorTag) => {
        setTags(t => [...t, tag]);
    }, [setTags]);
    const removeTag = useCallback((tag: TagSelectorTag) => {
        setTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setTags]);
    const clearTags = useCallback(() => {
        setTags([]);
    }, [setTags]);

    // Handle translations
    type Translation = NewObject<Routine['translations'][0]>;
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
        if (routine?.owner?.__typename === 'Organization') setOrganizationFor(routine.owner as Organization);
        else setOrganizationFor(null);
        setInputsList(routine?.inputs ?? []);
        setOutputsList(routine?.outputs ?? []);
        setResourceList(routine?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
        setTags(routine?.tags ?? []);
        setTranslations(routine?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            description: t.description ?? '',
            instructions: t.instructions ?? '',
            title: t.title ?? '',
        })) ?? []);
    }, [routine]);

    // Handle update
    const [mutation] = useMutation<routineUpdate, routineUpdateVariables>(routineUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            instructions: '',
            title: '',
            version: routine?.version ?? '1.0.0',
            isComplete: routine?.isComplete ?? true,
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            const existingResourceList = Array.isArray(routine?.resourceLists) ? (routine as Routine).resourceLists.find(list => list.usedFor === ResourceListUsedFor.Display) : undefined;
            const resourceListUpdate = existingResourceList ? { resourceListsUpdate: formatForUpdate(existingResourceList, resourceList, [], ['resources']) } : {};
            const ownedBy: { organizationId: string; } | { userId: string; } = organizationFor ? { organizationId: organizationFor.id } : { userId: session?.id ?? '' };
            const allTranslations = getTranslationsUpdate(language, {
                language,
                description: values.description,
                instructions: values.instructions,
                title: values.title,
            })
            mutationWrapper({ //TODO for morning: updating inputs/outputs can break because standard. If standard is custom, then it should always display as custom (does not do that yet). when updated, should check if standard is custom and treat it different when formatting for update. standardUpdate is currently an invalid API field, since other people may use your standard. Should change so update is valid, just need to check that it is yours and custom. Then make sure that custom standards are hidden from all search results, so no one else uses them.
                mutation,
                input: formatForUpdate(routine, {
                    id,
                    ...ownedBy,
                    version: values.version,
                    isComplete: values.isComplete,
                    // Handle standard in inputs
                    inputs: inputsList.map(input => {
                        const originalInput = routine?.inputs.find(i => i.id === input.id);
                        // If current input's standard is not internal, set as connect
                        if (input.standard && !input.standard?.isInternal) {
                            return {
                                ...input,
                                standard: undefined,
                                standardConnect: {
                                    id: input.standard.id,
                                }
                            }
                        }
                        // Else if changed, set as create
                        else if (input.standard && originalInput?.standard && JSON.stringify(input.standard) !== JSON.stringify(originalInput.standard)) {
                            return {
                                ...input,
                                standard: undefined,
                                standardCreate: input.standard
                            }
                        }
                        // Otherwise, return unchanged
                        return input;
                    }),
                    // Handle standard in outputs
                    outputs: outputsList.map(output => {
                        const originalOutput = routine?.outputs.find(o => o.id === output.id);
                        // If current output's standard is not internal, set as connect
                        if (output.standard && !output.standard?.isInternal) {
                            return {
                                ...output,
                                standard: undefined,
                                standardConnect: {
                                    id: output.standard.id,
                                }
                            }
                        }
                        // Else if changed, set as create
                        else if (output.standard && originalOutput?.standard && JSON.stringify(output.standard) !== JSON.stringify(originalOutput.standard)) {
                            return {
                                ...output,
                                standard: undefined,
                                standardCreate: output.standard
                            }
                        }
                        // Otherwise, return unchanged
                        return output;
                    }),
                    ...resourceListUpdate,
                    ...shapeTagsUpdate(routine?.tags, tags),
                    translations: allTranslations,
                },
                    ['tags', 'inputs.standard', 'outputs.standard'],
                    ['translations', 'inputs', 'outputs', 'inputs.translations', 'outputs.translations'],
                ),
                onSuccess: (response) => { onUpdated(response.data.routineUpdate) },
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
                instructions: translations[0].instructions,
                title: translations[0].title,
            })
        }
    }, [formik, languages, setLanguage, setLanguages, translations])
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

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, Boolean(formik.isSubmitting || !formik.isValid), true, () => { }],
        ['Cancel', CancelIcon, formik.isSubmitting, false, onCancel],
    ], [formik, onCancel]);
    const [formBottom, setFormBottom] = useState<number>(0);
    const handleResize = useCallback(({ height }: any) => {
        setFormBottom(height);
    }, [setFormBottom]);

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <UserOrganizationSwitch
                    session={session}
                    selected={organizationFor}
                    onChange={onSwitchChange}
                    zIndex={zIndex}
                />
            </Grid>
            {/* TODO add project selector */}
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
                    helperText={formik.touched.instructions ? formik.errors.instructions as string : null}
                />
            </Grid>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="version"
                    name="version"
                    label="version"
                    value={formik.values.version}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.version && Boolean(formik.errors.version)}
                    helperText={formik.touched.version && formik.errors.version}
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
                    session={session}
                    tags={tags}
                    onTagAdd={addTag}
                    onTagRemove={removeTag}
                    onTagsClear={clearTags}
                />
            </Grid>
            <Grid item xs={12} marginBottom={4}>
                <Tooltip placement={'top'} title='Is this routine ready for anyone to use?'>
                    <FormControlLabel
                        label='Complete'
                        control={
                            <Checkbox
                                id='routine-is-complete'
                                name='isComplete'
                                color='secondary'
                                checked={formik.values.isComplete}
                                onChange={formik.handleChange}
                                onBlur={formik.handleBlur}
                            />
                        }
                    />
                </Tooltip>
            </Grid>
        </Grid>
    ), [session, organizationFor, onSwitchChange, zIndex, language, handleAddLanguage, handleLanguageDelete, handleLanguageSelect, languages, formik, handleInputsUpdate, inputsList, handleOutputsUpdate, outputsList, resourceList, handleResourcesUpdate, loading, tags, addTag, removeTag, clearTags]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            paddingBottom: `${formBottom}px`,
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
            <DialogActionsContainer actions={actions} onResize={handleResize} />
        </form>
    )
}