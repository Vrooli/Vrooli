import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useLazyQuery, useMutation } from "@apollo/client";
import { organization, organizationVariables } from "graphql/generated/organization";
import { organizationQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { OrganizationUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { organizationUpdateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { organizationUpdateMutation } from "graphql/mutation";
import { formatForUpdate, updateArray } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { TagSelectorTag } from "components/inputs/types";
import { LanguageInput, ResourceListHorizontal, TagSelector } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { NewObject, Organization, ResourceList } from "types";
import { v4 as uuidv4 } from 'uuid';
import { ResourceListUsedFor } from "graphql/generated/globalTypes";

export const OrganizationUpdate = ({
    onCancel,
    onUpdated,
    session,
}: OrganizationUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Organization}/edit/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchOrganizations}/edit/:id`);
    const id = params?.id ?? params2?.id;
    // Fetch existing data
    const [getData, { data, loading }] = useLazyQuery<organization, organizationVariables>(organizationQuery);
    useEffect(() => {
        if (id) getData({ variables: { input: { id } } });
    }, [getData, id])
    const organization = useMemo(() => data?.organization, [data]);

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
    type Translation = NewObject<Organization['translations'][0]>;
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
        setResourceList(organization?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuidv4(), usedFor: ResourceListUsedFor.Display } as any);
        setTags(organization?.tags ?? []);
        setTranslations(organization?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            bio: t.bio ?? '',
            name: t.name ?? '',
        })) ?? []);
    }, [organization]);

    // Handle update
    const [mutation] = useMutation<organization>(organizationUpdateMutation);
    const formik = useFormik({
        initialValues: {
            name: '',
            bio: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            const existingResourceList = Array.isArray(organization?.resourceLists) ? (organization as Organization).resourceLists.find(list => list.usedFor === ResourceListUsedFor.Display) : undefined;
            const resourceListUpdate = existingResourceList ? { resourceListsUpdate: formatForUpdate(existingResourceList, resourceList, [], ['resources']) } : {};
            const tagsUpdate = tags.length > 0 ? {
                // Create/connect/disconnect new tags
                tagsCreate: tags.filter(t => !t.id && !organization?.tags?.some(tag => tag.tag === t.tag)).map(t => ({ tag: t.tag })),
                tagsConnect: tags.filter(t => t.id && !organization?.tags?.some(tag => tag.tag === t.tag)).map(t => (t.id)),
                tagsDisconnect: organization?.tags?.filter(t => !tags.some(tag => tag.tag === t.tag)).map(t => (t.id)),
            } : {};
            const allTranslations = getTranslationsUpdate(language, {
                language,
                bio: values.bio,
                name: values.name,
            })
            mutationWrapper({
                mutation,
                input: formatForUpdate(organization, {
                    id,
                    ...resourceListUpdate,
                    ...tagsUpdate,
                    translations: allTranslations,
                }, ['tags'], ['translations']),
                onSuccess: (response) => { onUpdated(response.data.organizationUpdate) },
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
                bio: translations[0].bio ?? '',
                name: translations[0].name,
            })
        }
    }, [formik, languages, setLanguage, setLanguages, translations])
    const handleLanguageChange = useCallback((oldLanguage: string, newLanguage: string) => {
        // Update translation
        updateTranslation(oldLanguage, {
            language: newLanguage,
            bio: formik.values.bio,
            name: formik.values.name,
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
            bio: existingTranslation?.bio ?? '',
            name: existingTranslation?.name ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            language,
            bio: formik.values.bio,
            name: formik.values.name,
        })
        // Update formik
        updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.bio, formik.values.name, updateFormikTranslation]);
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
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="name"
                    name="name"
                    label="Name"
                    value={formik.values.name}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.name && Boolean(formik.errors.name)}
                    helperText={formik.touched.name && formik.errors.name}
                />
            </Grid>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="bio"
                    name="bio"
                    label="Bio"
                    multiline
                    minRows={4}
                    value={formik.values.bio}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.bio && Boolean(formik.errors.bio)}
                    helperText={formik.touched.bio && formik.errors.bio}
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
    ), [language, handleAddLanguage, handleLanguageChange, handleLanguageDelete, handleLanguageSelect, languages, session, formik.values.name, formik.values.bio, formik.handleBlur, formik.handleChange, formik.touched.name, formik.touched.bio, formik.errors.name, formik.errors.bio, resourceList, handleResourcesUpdate, loading, tags, addTag, removeTag, clearTags]);

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