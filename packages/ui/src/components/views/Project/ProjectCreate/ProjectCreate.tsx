import { Grid, TextField, Typography } from "@mui/material";
import { useMutation } from "@apollo/client";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { projectCreateForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { projectCreateMutation } from "graphql/mutation";
import { getUserLanguages, ObjectType, ProjectTranslationShape, shapeProjectCreate, TagShape, updateArray, useReactSearch } from "utils";
import { ProjectCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { DialogActionItem } from "components/containers/types";
import {
    Add as CreateIcon,
    Restore as CancelIcon,
} from '@mui/icons-material';
import { LanguageInput, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { projectCreate, projectCreateVariables } from "graphql/generated/projectCreate";
import { RelationshipOwner, RelationshipParent, RelationshipProject } from "components/inputs/types";

export const ProjectCreate = ({
    onCreated,
    onCancel,
    session,
    zIndex,
}: ProjectCreateProps) => {
    const params = useReactSearch(null);

    // If this object is complete/incomplete
    const [isComplete, setIsComplete] = useState(false);
    const onIsCompleteChange = useCallback((newIsComplete: boolean) => { setIsComplete(newIsComplete) }, [setIsComplete]);
    // If this object is public or private
    const [isPrivate, setIsPrivate] = useState(false);
    const onIsPrivateChange = useCallback((newIsPrivate: boolean) => { setIsPrivate(newIsPrivate) }, [setIsPrivate]);
    // Who can control the project
    const [owner, setOwner] = useState<RelationshipOwner>(userFromSession(session));
    const onOwnerChange = useCallback((owner: RelationshipOwner) => { setOwner(owner); }, [setOwner]);
    // Object this was forked from
    const [parent, setParent] = useState<RelationshipParent>(null);
    const onParentChange = useCallback((parent: RelationshipParent) => { setParent(parent); }, [setParent]);
    // What project this object is a part of
    const [project, setProject] = useState<RelationshipProject>(null);
    const onProjectChange = useCallback((project: RelationshipProject) => { setProject(project); }, [setProject]);

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

    // TODO upgrade to pull data from search params like it's done in AdvancedSearchDialog
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
                    isComplete,
                    isPrivate,
                    owner,
                    parent,
                    // project, //TODO
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
                    <Typography
                        component="h1"
                        variant="h3"
                        sx={{
                            textAlign: 'center',
                            sx: { marginTop: 2, marginBottom: 2 },
                        }}
                    >Create Project</Typography>
                </Grid>
                <Grid item xs={12}>
                    {/* <UserOrganizationSwitch TODO delete switches when relationshipbuttons working
                        session={session}
                        selected={organizationFor}
                        onChange={onSwitchChange}
                        zIndex={zIndex}
                    /> */}
                    <RelationshipButtons
                        isComplete={isComplete}
                        isPrivate={isPrivate}
                        objectType={ObjectType.Project}
                        onIsCompleteChange={onIsCompleteChange}
                        onIsPrivateChange={onIsPrivateChange}
                        onOwnerChange={onOwnerChange}
                        onProjectChange={onProjectChange}
                        onParentChange={onParentChange as any}
                        owner={owner}
                        parent={parent}
                        project={project}
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