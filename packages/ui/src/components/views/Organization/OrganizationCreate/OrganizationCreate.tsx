import { Checkbox, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material";
import { useMutation } from "@apollo/client";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { organizationCreate as validationSchema, organizationTranslationCreate } from '@shared/validation';
import { useFormik } from 'formik';
import { organizationCreateMutation } from "graphql/mutation";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, ObjectType, parseSearchParams, removeTranslation, shapeOrganizationCreate, TagShape, usePromptBeforeUnload } from "utils";
import { OrganizationCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, userFromSession } from "components";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { organizationCreate, organizationCreateVariables } from "graphql/generated/organizationCreate";
import { RelationshipsObject } from "components/inputs/types";

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
        const params = parseSearchParams(window.location.search);
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle languages
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const [languages, setLanguages] = useState<string[]>([getUserLanguages(session)[0]]);

    // Handle create
    const [mutation] = useMutation<organizationCreate, organizationCreateVariables>(organizationCreateMutation);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            isOpenToNewMembers: false,
            translationsCreate: [{
                id: uuid(),
                language,
                name: '',
                bio: '',
            }]
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: shapeOrganizationCreate({
                    id: values.id,
                    isOpenToNewMembers: values.isOpenToNewMembers,
                    isPrivate: relationships.isPrivate,
                    resourceLists: [resourceList],
                    tags,
                    translations: values.translationsCreate,
                }),
                onSuccess: (response) => { onCreated(response.data.organizationCreate) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Current name and bio info, as well as errors
    const { bio, name, errorBio, errorName, touchedBio, touchedName, errors } = useMemo(() => {
        console.log('org create gettransdata')
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
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsCreate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsCreate', e, language)
    }, [formik, language]);

    // Handle languages
    const handleLanguageSelect = useCallback((newLanguage: string) => { setLanguage(newLanguage) }, []);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
        addEmptyTranslation(formik, 'translationsCreate', newLanguage);
    }, [formik, handleLanguageSelect, languages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        setLanguages(newLanguages);
        removeTranslation(formik, 'translationsCreate', language);
    }, [formik, languages]);

    const isLoggedIn = useMemo(() => session?.isLoggedIn === true && uuidValidate(session?.id ?? ''), [session]);

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
                <Grid item xs={12}>
                    <RelationshipButtons
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
                        value={name}
                        onBlur={onTranslationBlur}
                        onChange={onTranslationChange}
                        error={touchedName && Boolean(errorName)}
                        helperText={touchedName && errorName}
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
                        value={bio}
                        onBlur={onTranslationBlur}
                        onChange={onTranslationChange}
                        error={touchedBio && Boolean(errorBio)}
                        helperText={touchedBio && errorBio}
                    />
                </Grid>
                <Grid item xs={12}>
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
                <GridSubmitButtons
                    disabledCancel={formik.isSubmitting}
                    disabledSubmit={Boolean(!isLoggedIn || formik.isSubmitting)}
                    errors={errors}
                    isCreate={true}
                    onCancel={onCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </Grid>
        </form >
    )
}