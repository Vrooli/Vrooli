import { Box, CircularProgress, Grid, Typography } from "@mui/material"
import { ResourceListUsedFor } from "@shared/consts";
import { useMutation, useLazyQuery } from "@apollo/client";
import { standard, standardVariables } from "graphql/generated/standard";
import { standardQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { StandardUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { standardUpdateForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { standardUpdateMutation } from "graphql/mutation";
import { PubSub, shapeStandardUpdate, StandardTranslationShape, TagShape, updateArray } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { LanguageInput, ResourceListHorizontal, TagSelector } from "components";
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { ResourceList } from "types";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { standardUpdate, standardUpdateVariables } from "graphql/generated/standardUpdate";

export const StandardUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: StandardUpdateProps) => {
    // Fetch existing data
    const id = useMemo(() => window.location.pathname.split('/').pop() ?? '', []);
    const [getData, { data, loading }] = useLazyQuery<standard, standardVariables>(standardQuery);
    useEffect(() => {
        if (uuidValidate(id)) getData({ variables: { input: { id } } });
    }, [getData, id])
    const standard = useMemo(() => data?.standard, [data]);

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
    type Translation = StandardTranslationShape;
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
        setResourceList(standard?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
        setTags(standard?.tags ?? []);
        setTranslations(standard?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            description: t.description ?? '',
            jsonVariable: t.jsonVariable ?? null,
        })) ?? []);
    }, [standard]);

    // Handle update
    const [mutation] = useMutation<standardUpdate, standardUpdateVariables>(standardUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            if (!standard) {
                PubSub.get().publishSnack({ message: 'Could not find existing standard data.', severity: 'error' });
                return;
            }
            // Update translations with final values
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                jsonVariable: null, //TODO
            })
            // Update
            mutationWrapper({
                mutation,
                input: shapeStandardUpdate(standard, {
                    id: standard.id,
                    translations: allTranslations,
                    resourceLists: [resourceList],
                    tags: tags,
                }),
                onSuccess: (response) => { onUpdated(response.data.standardUpdate) },
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
            })
        }
    }, [formik, languages, setLanguage, setLanguages, translations])
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            jsonVariable: null, //TODO
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, updateFormikTranslation]);
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
                <Typography
                    component="h1"
                    variant="h3"
                    sx={{
                        textAlign: 'center',
                        sx: { marginTop: 2, marginBottom: 2 },
                    }}
                >Update Standard</Typography>
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
            {/* TODO description, versioning */}
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
                    session={session}
                    tags={tags}
                    onTagAdd={addTag}
                    onTagRemove={removeTag}
                    onTagsClear={clearTags}
                />
            </Grid>
        </Grid>
    ), [language, handleAddLanguage, handleLanguageDelete, handleLanguageSelect, languages, session, zIndex, resourceList, handleResourcesUpdate, loading, tags, addTag, removeTag, clearTags]);


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