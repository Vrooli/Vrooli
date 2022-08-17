import { Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { projectCreateForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { projectCreateMutation } from "graphql/mutation";
import { getUserLanguages, ProjectTranslationShape, shapeProjectCreate, TagShape, updateArray, useReactSearch } from "utils";
import { ProjectCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { DialogActionItem } from "components/containers/types";
import {
    Add as CreateIcon,
    Restore as CancelIcon,
} from '@mui/icons-material';
import { LanguageInput, ResourceListHorizontal, TagSelector, UserOrganizationSwitch } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { Organization } from "types";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { projectCreate, projectCreateVariables } from "graphql/generated/projectCreate";

export const ProjectCreate = ({
    onCreated,
    onCancel,
    session,
    zIndex,
}: ProjectCreateProps) => {
    const params = useReactSearch(null);

    // Handle user/organization switch
    const [organizationFor, setOrganizationFor] = useState<Organization | null>(null);
    const onSwitchChange = useCallback((organization: Organization | null) => { setOrganizationFor(organization) }, []);

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
    type Translation = ProjectTranslationShape;
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
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, [params]);

    // Handle create
    const [mutation] = useMutation<projectCreate, projectCreateVariables>(projectCreateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            name: '',
        },
        validationSchema,
        onSubmit: (values) => {
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                name: values.name,
            })
            mutationWrapper({
                mutation,
                input: shapeProjectCreate({
                    id: uuid(),
                    isComplete: false, //TODO: values.isComplete,
                    owner: organizationFor ? {
                        __typename: 'Organization',
                        id: organizationFor.id,
                    } : {
                        __typename: 'User',
                        id: session.id ?? '',
                    },
                    resourceLists: [resourceList],
                    tags: tags,
                    translations: allTranslations,
                }),
                onSuccess: (response) => { onCreated(response.data.projectCreate) },
                onError: () => { formik.setSubmitting(false) },
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
            name: existingTranslation?.name ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            name: formik.values.name,
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, formik.values.name, updateFormikTranslation]);
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

    const actions: DialogActionItem[] = useMemo(() => {
        const loggedIn = session?.isLoggedIn === true && uuidValidate(session?.id ?? '');
        return [
            ['Create', CreateIcon, Boolean(!loggedIn || formik.isSubmitting), true, () => { }],
            ['Cancel', CancelIcon, formik.isSubmitting, false, onCancel],
        ] as DialogActionItem[]
    }, [formik, onCancel, session]);
    const [formBottom, setFormBottom] = useState<number>(0);
    const handleResize = useCallback(({ height }: any) => {
        setFormBottom(height);
    }, [setFormBottom]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            paddingBottom: `${formBottom}px`,
            zIndex,
        }}
        >
            <Grid container spacing={2} sx={{ padding: 2, maxWidth: 'min(700px, 100%)' }}>
                <Grid item xs={12}>
                    <UserOrganizationSwitch
                        session={session}
                        selected={organizationFor}
                        onChange={onSwitchChange}
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
                        loading={false}
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
            <DialogActionsContainer actions={actions} onResize={handleResize} />
        </form>
    )
}