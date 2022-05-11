import { Box, CircularProgress, Grid } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useLazyQuery } from "@apollo/client";
import { standard, standardVariables } from "graphql/generated/standard";
import { standardQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { StandardUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { standardUpdateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { standardUpdateMutation } from "graphql/mutation";
import { formatForUpdate, updateArray } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { LanguageInput, TagSelector } from "components";
import { TagSelectorTag } from "components/inputs/types";
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { NewObject, Organization, Standard } from "types";

export const StandardUpdate = ({
    onCancel,
    onUpdated,
    session,
}: StandardUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Standard}/edit/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchStandards}/edit/:id`);
    const id = params?.id ?? params2?.id;
    // Fetch existing data
    const [getData, { data, loading }] = useLazyQuery<standard, standardVariables>(standardQuery);
    useEffect(() => {
        if (id) getData({ variables: { input: { id } } });
    }, [getData, id])
    const standard = useMemo(() => data?.standard, [data]);

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
    type Translation = NewObject<Standard['translations'][0]>;
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
        setTags(standard?.tags ?? []);
        setTranslations(standard?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            description: t.description ?? '',
        })) ?? []);
    }, [standard]);

    // Handle update
    const [mutation] = useMutation<standard>(standardUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            const tagsUpdate = tags.length > 0 ? {
                tagsCreate: tags.filter(t => !t.id && !standard?.tags?.some(tag => tag.tag === t.tag)).map(t => ({ tag: t.tag })),
                tagsConnect: tags.filter(t => t.id && !standard?.tags?.some(tag => tag.tag === t.tag)).map(t => (t.id)),
                tagsDisconnect: standard?.tags?.filter(t => !tags.some(tag => tag.tag === t.tag)).map(t => (t.id)),
            } : {};
            const allTranslations = getTranslationsUpdate(language, {
                language,
                description: values.description,
            })
            mutationWrapper({
                mutation,
                input: formatForUpdate(standard, {
                    id,
                    organizationId: (standard?.creator as Organization)?.id ?? undefined,
                    ...tagsUpdate,
                    translations: allTranslations,
                }, ['tags'], ['translations']),
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
    const handleLanguageChange = useCallback((oldLanguage: string, newLanguage: string) => {
        // Update translation
        updateTranslation(oldLanguage, {
            language: newLanguage,
            description: formik.values.description,
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
    }, [formik.values, languages, setLanguage, setLanguages, updateTranslation]);
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
            language,
            description: formik.values.description,
        })
        // Update formik
        updateFormikTranslation(newLanguage);
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
            {/* TODO */}
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
    ), [language, handleAddLanguage, handleLanguageChange, handleLanguageDelete, handleLanguageSelect, languages, session, tags, addTag, removeTag, clearTags]);


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