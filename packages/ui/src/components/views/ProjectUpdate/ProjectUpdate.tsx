import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useLazyQuery } from "@apollo/client";
import { project, projectVariables } from "graphql/generated/project";
import { projectQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ProjectUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { projectUpdateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { projectUpdateMutation } from "graphql/mutation";
import { formatForUpdate, getTranslation } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { TagSelectorTag } from "components/inputs/types";
import { LanguageInput, ResourceListHorizontal, TagSelector, UserOrganizationSwitch } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { Organization, Project, ResourceList } from "types";
import { v4 as uuidv4 } from 'uuid';
import { ResourceListUsedFor } from "graphql/generated/globalTypes";

export const ProjectUpdate = ({
    onCancel,
    onUpdated,
    session,
}: ProjectUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Project}/edit/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchProjects}/edit/:id`);
    const id = params?.id ?? params2?.id;
    // Fetch existing data
    const [getData, { data, loading }] = useLazyQuery<project, projectVariables>(projectQuery);
    useEffect(() => {
        if (id) getData({ variables: { input: { id } } });
    }, [id])
    const project = useMemo(() => data?.project, [data]);

    // Handle user/organization switch
    const [organizationFor, setOrganizationFor] = useState<Organization | null>(null);
    const onSwitchChange = useCallback((organization: Organization | null) => { setOrganizationFor(organization) }, [setOrganizationFor]);

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
        name: string;
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
        setResourceList(project?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuidv4(), usedFor: ResourceListUsedFor.Display } as any);
        setTags(project?.tags ?? []);
        setTranslations(project?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            description: t.description ?? '',
            name: t.name ?? '',
        })) ?? []);
    }, [project]);

    // Handle update
    const [mutation] = useMutation<project>(projectUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            name: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            const existingResourceList = Array.isArray(project?.resourceLists) ? (project as Project).resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) : undefined;
            const resourceListUpdate = existingResourceList ? { resourceListsUpdate: formatForUpdate(existingResourceList, resourceList, [], ['resources']) } : {};
            const tagsUpdate = tags.length > 0 ? {
                // Create/connect new tags
                tagsCreate: tags.filter(t => !t.id && !project?.tags?.some(tag => tag.tag === t.tag)).map(t => ({ tag: t.tag })),
                tagsConnect: tags.filter(t => t.id && !project?.tags?.some(tag => tag.tag === t.tag)).map(t => (t.id)),
            } : {};
            const allTranslations = getTranslationsUpdate(language, {
                language,
                description: values.description,
                name: values.name,
            })
            mutationWrapper({
                mutation,
                input: formatForUpdate(project, {
                    id,
                    organizationId: (project?.owner as Organization)?.id ?? undefined,
                    ...resourceListUpdate,
                    ...tagsUpdate,
                    translations: allTranslations,
                }, ['tags'], ['translations']),
                onSuccess: (response) => { onUpdated(response.data.projectUpdate) },
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
                name: translations[0].name,
            })
        }
    }, [languages, setLanguage, setLanguages, translations])
    const handleLanguageChange = useCallback((oldLanguage: string, newLanguage: string) => {
        // Update translation
        updateTranslation(oldLanguage, {
            language: newLanguage,
            description: formik.values.description,
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
    }, [formik.values, languages, translations, setLanguage, setLanguages, updateTranslation]);
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
            name: existingTranslation?.name ?? '',
        });
    }, [formik.setValues, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            language,
            description: formik.values.description,
            name: formik.values.name,
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
                <UserOrganizationSwitch session={session} selected={organizationFor} onChange={onSwitchChange} />
            </Grid>
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
                    id="description"
                    name="description"
                    label="description"
                    multiline
                    minRows={4}
                    value={formik.values.description}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.description && Boolean(formik.errors.description)}
                    helperText={formik.touched.description && formik.errors.description}
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