import { Grid, TextField } from "@mui/material";
import { useMutation } from "graphql/hooks";
import { mutationWrapper } from 'graphql/utils';
import { projectVersionTranslationValidation, projectVersionValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, getUserLanguages, handleTranslationBlur, handleTranslationChange, parseSearchParams, removeTranslation, shapeProjectVersion, TagShape, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { ProjectCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession, VersionInput } from "components";
import { uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { getCurrentUser } from "utils/authentication";
import { projectVersionEndpoint } from "graphql/endpoints";
import { ProjectVersion, ProjectVersionCreateInput, ResourceList } from "@shared/consts";

export const ProjectCreate = ({
    onCreated,
    onCancel,
    session,
    zIndex,
}: ProjectCreateProps) => {

    // Handle root data, public/private status, etc.
    const [relationships, setRelationships] = useState<RelationshipsObject>({
        isComplete: false,
        isPrivate: false,
        owner: userFromSession(session),
        parent: null,
        project: null,
    });
    const onRelationshipsChange = useCallback((newRelationshipsObject: Partial<RelationshipsObject>) =>
        setRelationships({ ...relationships, ...newRelationshipsObject }), [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid() } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // TODO upgrade to pull data from search params like it's done in AdvancedSearchDialog
    useEffect(() => {
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useMutation<ProjectVersion, ProjectVersionCreateInput, 'projectVersionCreate'>(...projectVersionEndpoint.create);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                name: '',
                description: '',
            }],
            versionInfo: {
                versionIndex: 0,
                versionLabel: '1.0.0',
                versionNotes: '',
            }
        },
        validationSchema: projectVersionValidation.create(),
        onSubmit: (values) => {
            mutationWrapper<ProjectVersion, ProjectVersionCreateInput>({
                mutation,
                input: shapeProjectVersion.create({
                    id: values.id,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    isLatest: true,
                    root: {
                        id: uuid(),
                        owner: relationships.owner,
                        parent: relationships.parent,
                    },
                    ...values.versionInfo,
                }),
                onSuccess: (data) => { onCreated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const translations = useTranslatedFields({
        fields: ['description', 'name'],
        formik,
        formikField: 'translationsCreate',
        language,
        validationSchema: projectVersionTranslationValidation.create(),
    });
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

    const isLoggedIn = useMemo(() => Boolean(getCurrentUser(session).id), [session]);

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
                    <PageTitle titleKey='CreateProject' session={session} />
                </Grid>
                <Grid item xs={12} mb={4}>
                    <RelationshipButtons
                        isEditing={true}
                        objectType={'Project'}
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
                        id="name"
                        name="name"
                        label="Name"
                        value={translations.name}
                        onBlur={onTranslationBlur}
                        onChange={onTranslationChange}
                        error={translations.touchedName && Boolean(translations.errorName)}
                        helperText={translations.touchedName && translations.errorName}
                    />
                </Grid>
                <Grid item xs={12} mb={4}>
                    <TextField
                        fullWidth
                        id="description"
                        name="description"
                        label="description"
                        multiline
                        minRows={4}
                        value={translations.description}
                        onBlur={onTranslationBlur}
                        onChange={onTranslationChange}
                        error={translations.touchedDescription && Boolean(translations.errorDescription)}
                        helperText={translations.touchedDescription && translations.errorDescription}
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
                <Grid item xs={12} mb={4}>
                    <TagSelector
                        handleTagsUpdate={handleTagsUpdate}
                        session={session}
                        tags={tags}
                    />
                </Grid>
                <Grid item xs={12} mb={4}>
                    <VersionInput
                        fullWidth
                        id="version"
                        name="version"
                        versionInfo={formik.values.versionInfo}
                        versions={[]}
                        onBlur={formik.handleBlur}
                        onChange={(newVersionInfo) => {
                            formik.setFieldValue('versionInfo', newVersionInfo);
                            setRelationships({
                                ...relationships,
                                isComplete: false,
                            })
                        }}
                        error={formik.touched.versionInfo?.versionLabel && Boolean(formik.errors.versionInfo?.versionLabel)}
                        helperText={formik.touched.versionInfo?.versionLabel ? formik.errors.versionInfo?.versionLabel : null}
                    />
                </Grid>
                <GridSubmitButtons
                    disabledSubmit={!isLoggedIn}
                    errors={translations.errorsWithTranslations}
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