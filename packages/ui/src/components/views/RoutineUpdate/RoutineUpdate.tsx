import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS, id } from "@local/shared";
import { useMutation, useLazyQuery } from "@apollo/client";
import { routine, routineVariables } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { RoutineUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { routineUpdateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { routineUpdateMutation } from "graphql/mutation";
import { formatForUpdate, getLanguageSubtag, getTranslation, getUserLanguages, Pubs, updateTranslation } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { TagSelectorTag } from "components/inputs/types";
import { DialogActionItem } from "components/containers/types";
import { LanguageInput, MarkdownInput, ResourceListHorizontal, TagSelector } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { v4 as uuidv4 } from 'uuid';
import { Organization, ResourceList, Routine } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";

export const RoutineUpdate = ({
    onCancel,
    onUpdated,
    session,
}: RoutineUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Run}/edit/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchRoutines}/edit/:id`);
    const id = params?.id ?? params2?.id;
    // Fetch existing data
    const [getData, { data, loading }] = useLazyQuery<routine, routineVariables>(routineQuery);
    useEffect(() => {
        if (id) getData({ variables: { input: { id } } });
    }, [id])
    const routine = useMemo(() => data?.routine, [data]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuidv4(), usedFor: ResourceListUsedFor.Display } as any);
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
    type Translation = {
        language: string;
        description: string;
        instructions: string;
        title: string;
    };
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        setTranslations([...translations.filter(t => t.language !== language)]);
    }, [translations, setTranslations]);
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = translations.findIndex(t => language === t.language);
        // If language exists, update
        if (index >= 0) {
            const newTranslations = [...translations];
            newTranslations[index] = { ...translation };
            return newTranslations;
        }
        // Otherwise, add new
        else {
            return [...translations, translation];
        }
    }, [translations]);
    const updateTranslation = useCallback((language: string, translation: Translation) => {
        setTranslations(getTranslationsUpdate(language, translation));
    }, [translations, setTranslations]);

    useEffect(() => {
        setResourceList(routine?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuidv4(), usedFor: ResourceListUsedFor.Display } as any);
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
    const [mutation] = useMutation<routine>(routineUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            instructions: '',
            title: '',
            version: routine?.version ?? 0,
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            const existingResourceList = Array.isArray(routine?.resourceLists) ? (routine as Routine).resourceLists.find(list => list.usedFor === ResourceListUsedFor.Display) : undefined;
            const resourceListUpdate = existingResourceList ? { resourceListsUpdate: formatForUpdate(existingResourceList, resourceList, [], ['resources']) } : {};
            const tagsUpdate = tags.length > 0 ? {
                // Create/connect new tags
                tagsCreate: tags.filter(t => !t.id && !routine?.tags?.some(tag => tag.tag === t.tag)).map(t => ({ tag: t.tag })),
                tagsConnect: tags.filter(t => t.id && !routine?.tags?.some(tag => tag.tag === t.tag)).map(t => (t.id)),
            } : {};
            const allTranslations = getTranslationsUpdate(language, {
                language,
                description: values.description,
                instructions: values.instructions,
                title: values.title,
            })
            mutationWrapper({
                mutation,
                input: formatForUpdate(routine, {
                    id,
                    organizationId: (routine?.owner as Organization)?.id ?? undefined,
                    version: values.version,
                    ...resourceListUpdate,
                    ...tagsUpdate,
                    translations: allTranslations,
                }, ['tags'], ['translations']),
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
                description: translations[0].description,
                instructions: translations[0].instructions,
                title: translations[0].title,
            })
        }
    }, [languages, setLanguage, setLanguages, translations])
    const handleLanguageChange = useCallback((oldLanguage: string, newLanguage: string) => {
        // Update translation
        updateTranslation(oldLanguage, {
            language: newLanguage,
            description: formik.values.description,
            instructions: formik.values.instructions,
            title: formik.values.title,
        });
        // Change selection
        setLanguage(newLanguage);
        // Update languages
        const newLanguages = [...languages];
        const index = newLanguages.findIndex(l => l === oldLanguage);
        if (index >= 0) {
            newLanguages[index] = newLanguage;
            setLanguages(newLanguages);
        }
    }, [formik.values, languages, translations, setLanguage, setLanguages, updateTranslation]);
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
            instructions: existingTranslation?.instructions ?? '',
            title: existingTranslation?.title ?? '',
        });
    }, [formik.setValues, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            language,
            description: formik.values.description,
            instructions: formik.values.instructions,
            title: formik.values.title,
        })
        // Update formik
        updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [formik.values, formik.setValues, language, translations, setLanguage, updateTranslation]);
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
    }, [deleteTranslation, handleLanguageSelect, languages, setLanguages]);

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, Boolean(formik.isSubmitting || !formik.isValid), true, () => { }],
        ['Cancel', CancelIcon, formik.isSubmitting, false, onCancel],
    ], [formik, onCancel, session]);
    const [formBottom, setFormBottom] = useState<number>(0);
    const handleResize = useCallback(({ height }: any) => {
        setFormBottom(height);
    }, [setFormBottom]);

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <LanguageInput
                    currentLanguage={language}
                    handleAdd={handleAddLanguage}
                    handleChange={handleLanguageChange}
                    handleDelete={handleLanguageDelete}
                    handleSelect={handleLanguageSelect}
                    languages={languages}
                    session={session}
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
                    rows={3}
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
                <ResourceListHorizontal
                    title={'Resources'}
                    list={resourceList}
                    canEdit={true}
                    handleUpdate={handleResourcesUpdate}
                    session={session}
                    mutate={false}
                />
            </Grid>
            <Grid item xs={12} marginBottom={4}>
                <TagSelector
                    session={session}
                    tags={tags}
                    onTagAdd={addTag}
                    onTagRemove={removeTag}
                    onTagsClear={clearTags}
                />
            </Grid>
        </Grid>
    ), [formik, actions, handleResize, formBottom, session, tags, addTag, removeTag, clearTags]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            paddingBottom: `${formBottom}px`,
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