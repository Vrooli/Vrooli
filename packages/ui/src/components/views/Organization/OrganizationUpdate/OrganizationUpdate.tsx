import { Box, Checkbox, CircularProgress, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material"
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { OrganizationUpdateProps } from "../types";
import { mutationWrapper } from 'api/utils';
import { organizationValidation, organizationTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, defaultResourceList, getPreferredLanguage, getUserLanguages, handleTranslationBlur, handleTranslationChange, parseSingleItemUrl, PubSub, removeTranslation, shapeOrganization, TagShape, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindByIdInput, Organization, OrganizationUpdateInput, ResourceList } from "@shared/consts";
import { organizationFindOne } from "api/generated/endpoints/organization_findOne";
import { organizationUpdate } from "api/generated/endpoints/organization_update";

export const OrganizationUpdate = ({
    onCancel,
    onUpdated,
session,
    zIndex,
}: OrganizationUpdateProps) => {
    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: organization, loading }] = useCustomLazyQuery<Organization, FindByIdInput>(organizationFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle resources
   const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
   const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // Handle update
    const [mutation] = useCustomMutation<Organization, OrganizationUpdateInput>(organizationUpdate);
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
        validationSchema: organizationValidation.update({}),
        onSubmit: (values) => {
            if (!organization) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadOrganization', severity: 'Error' });
                return;
            }
            mutationWrapper<Organization, OrganizationUpdateInput>({
                mutation,
                input: shapeOrganization.update(organization, {
                    id: organization.id,
                    isOpenToNewMembers: values.isOpenToNewMembers,
                    isPrivate: relationships.isPrivate,
                    resourceList: resourceList,
                    tags: tags,
                    translations: values.translationsUpdate,
                }),
                onSuccess: (data) => { onUpdated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const translations = useTranslatedFields({
        fields: ['bio', 'name'],
        formik, 
        formikField: 'translationsUpdate', 
        language, 
        validationSchema: organizationTranslationValidation.update({}),
    });
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
                    id="bio"
                    name="bio"
                    label="Bio"
                    multiline
                    minRows={4}
                    value={translations.bio}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={translations.touchedBio && Boolean(translations.errorBio)}
                    helperText={translations.touchedBio && translations.errorBio}
                />
            </Grid>
            <Grid item xs={12}>
                <ResourceListHorizontal
                    title={'Resources'}
                    list={resourceList}
                    canUpdate={true}
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
                errors={translations.errorsWithTranslations}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid>
    ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, formik.values.translationsUpdate, formik.values.isOpenToNewMembers, formik.handleChange, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, onTranslationBlur, onTranslationChange, translations, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, onCancel]);

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