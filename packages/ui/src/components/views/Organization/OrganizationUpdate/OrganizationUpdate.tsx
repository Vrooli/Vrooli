import { Box, Checkbox, CircularProgress, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material"
import { useLazyQuery, useMutation } from "graphql/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { OrganizationUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils';
import { organizationValidation, organizationTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, base36ToUuid, getFormikErrorsWithTranslations, getLastUrlPart, getPreferredLanguage, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, PubSub, removeTranslation, shapeOrganizationUpdate, TagShape, usePromptBeforeUnload } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, SnackSeverity, TagSelector, userFromSession } from "components";
import { DUMMY_ID, uuid, uuidValidate } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindByIdInput, Organization, OrganizationUpdateInput, ResourceList } from "@shared/consts";
import { organizationEndpoint } from "graphql/endpoints";

export const OrganizationUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: OrganizationUpdateProps) => {
    // Fetch existing data
    const id = useMemo(() => base36ToUuid(getLastUrlPart()), []);
    const [getData, { data, loading }] = useLazyQuery<Organization, FindByIdInput, 'organization'>(...organizationEndpoint.findOne);
    useEffect(() => { uuidValidate(id) && getData({ variables: { id } }) }, [getData, id])
    const organization = useMemo(() => data?.organization, [data]);

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
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid() } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // Handle update
    const [mutation] = useMutation<Organization, OrganizationUpdateInput, 'organizationUpdate'>(...organizationEndpoint.update);
    const formik = useFormik({
        initialValues: {
            id: organization?.id ?? uuid(),
            isOpenToNewMembers: organization?.isOpenToNewMembers ?? false,
            translationsUpdate: organization?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                name: '',
                bio: '',
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: organizationValidation.update!(),
        onSubmit: (values) => {
            if (!organization) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadOrganization', severity: SnackSeverity.Error });
                return;
            }
            mutationWrapper<Organization, OrganizationUpdateInput>({
                mutation,
                input: shapeOrganizationUpdate(organization, {
                    id: organization.id,
                    isOpenToNewMembers: values.isOpenToNewMembers,
                    isPrivate: relationships.isPrivate,
                    resourceList: resourceList,
                    tags: tags,
                    translations: values.translationsUpdate.map(t => ({
                        ...t,
                        id: t.id === DUMMY_ID ? uuid() : t.id,
                    })),
                }),
                onSuccess: (data) => { onUpdated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const { bio, name, errorBio, errorName, touchedBio, touchedName, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            bio: value?.bio ?? '',
            name: value?.name ?? '',
            errorBio: error?.bio ?? '',
            errorName: error?.name ?? '',
            touchedBio: touched?.bio ?? false,
            touchedName: touched?.name ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', organizationTranslationValidation.update!()),
        }
    }, [formik, language]);
    const languages = useMemo(() => formik.values.translationsUpdate.map(t => t.language), [formik.values.translationsUpdate]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(formik, 'translationsUpdate', newLanguage);
    }, [formik]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        removeTranslation(formik, 'translationsUpdate', language);
    }, [formik, languages]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => handleTranslationBlur(formik, 'translationsUpdate', e, language), [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => handleTranslationChange(formik, 'translationsUpdate', e, language), [formik, language]);

    useEffect(() => {
        setRelationships({
            isComplete: false,
            isPrivate: organization?.isPrivate ?? false,
            owner: null,
            parent: null,
            project: null,
        });
        setResourceList(organization?.resourceList ?? { id: uuid() } as any);
        setTags(organization?.tags ?? []);
        if (organization?.translations?.length) {
            setLanguage(getPreferredLanguage(organization.translations.map(t => t.language), getUserLanguages(session)));
        }
    }, [organization, session]);

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <PageTitle titleKey='UpdateOrganization' session={session} />
            </Grid>
            <Grid item xs={12} mb={4}>
                <RelationshipButtons
                    isEditing={true}
                    objectType={'Organization'}
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
                    translations={formik.values.translationsUpdate}
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
                    loading={loading}
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
                errors={errors}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid>
    ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, formik.values.translationsUpdate, formik.values.isOpenToNewMembers, formik.handleChange, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, name, onTranslationBlur, onTranslationChange, touchedName, errorName, bio, touchedBio, errorBio, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, errors, onCancel]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
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
        </form>
    )
}