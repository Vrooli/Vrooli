import { Checkbox, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material";
import { useMutation } from "@apollo/client";
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { organizationCreate as validationSchema, organizationTranslationCreate } from '@shared/validation';
import { useFormik } from 'formik';
import { organizationCreateMutation } from "graphql/mutation";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, ObjectType, parseSearchParams, removeTranslation, shapeOrganizationCreate, TagShape, usePromptBeforeUnload } from "utils";
import { OrganizationCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession } from "components";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { uuid } from '@shared/uuid';
import { organizationCreateVariables, organizationCreate_organizationCreate } from "graphql/generated/organizationCreate";
import { RelationshipsObject } from "components/inputs/types";
import { getCurrentUser } from "utils/authentication";

export const OrganizationCreate = ({
    onCreated,
    onCancel,
    session,
    zIndex,
}: OrganizationCreateProps) => {

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

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useMutation(organizationCreateMutation);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            isOpenToNewMembers: false,
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                name: '',
                bio: '',
            }]
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper<organizationCreate_organizationCreate, organizationCreateVariables>({
                mutation,
                input: shapeOrganizationCreate({
                    id: values.id,
                    isOpenToNewMembers: values.isOpenToNewMembers,
                    isPrivate: relationships.isPrivate,
                    resourceLists: [resourceList],
                    tags,
                    translations: values.translationsCreate,
                }),
                onSuccess: (data) => { onCreated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const { bio, name, errorBio, errorName, touchedBio, touchedName, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsCreate', language);
        return {
            bio: value?.bio ?? '',
            name: value?.name ?? '',
            errorBio: error?.bio ?? '',
            errorName: error?.name ?? '',
            touchedBio: touched?.bio ?? false,
            touchedName: touched?.name ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsCreate', organizationTranslationCreate),
        }
    }, [formik, language]);
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
        }}
        >
            <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                <Grid item xs={12}>
                    <PageTitle title="Create Organization" />
                </Grid>
                <Grid item xs={12} mb={4}>
                    <RelationshipButtons
                        isEditing={true}
                        objectType={ObjectType.Organization}
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
                        value={name}
                        onBlur={onTranslationBlur}
                        onChange={onTranslationChange}
                        error={touchedName && Boolean(errorName)}
                        helperText={touchedName && errorName}
                    />
                </Grid>
                <Grid item xs={12} mb={4}>
                    <TextField
                        fullWidth
                        id="bio"
                        name="bio"
                        label="Bio"
                        multiline
                        minRows={4}
                        value={bio}
                        onBlur={onTranslationBlur}
                        onChange={onTranslationChange}
                        error={touchedBio && Boolean(errorBio)}
                        helperText={touchedBio && errorBio}
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
                <Grid item xs={12} mb={4}>
                    <Tooltip placement={'top'} title='Indicates if this organization should be displayed when users are looking for an organization to join'>
                        <FormControlLabel
                            label='Open to new members?'
                            control={
                                <Checkbox
                                    id='organization-is-open-to-new-members'
                                    size="medium"
                                    name='isOpenToNewMembers'
                                    color='secondary'
                                    checked={formik.values.isOpenToNewMembers}
                                    onChange={formik.handleChange}
                                />
                            }
                        />
                    </Tooltip>
                </Grid>
                <GridSubmitButtons
                    disabledSubmit={!isLoggedIn}
                    errors={errors}
                    isCreate={true}
                    loading={formik.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </Grid>
        </form >
    )
}