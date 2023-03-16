import { Checkbox, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material";
import { FindByIdInput, Organization, OrganizationUpdateInput, ResourceList } from "@shared/consts";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { organizationTranslationValidation, organizationValidation } from '@shared/validation';
import { organizationFindOne } from "api/generated/endpoints/organization_findOne";
import { organizationUpdate } from "api/generated/endpoints/organization_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RelationshipButtons } from "components/inputs/RelationshipButtons/RelationshipButtons";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { RelationshipsObject } from "components/inputs/types";
import { ResourceListHorizontal } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useEffect, useMemo, useState } from "react";
import { defaultRelationships } from "utils/defaults/relationships";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { shapeOrganization } from "utils/shape/models/organization";
import { TagShape } from "utils/shape/models/tag";
import { OrganizationUpdateProps } from "../types";

export const OrganizationUpdate = ({
    display = 'page',
    session,
    zIndex = 200,
}: OrganizationUpdateProps) => {
    const { onCancel, onUpdated } = useUpdateActions<Organization>();

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

    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        onTranslationBlur,
        onTranslationChange,
        setLanguage,
        translations,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['bio', 'name'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: organizationTranslationValidation.update({}),
    });

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
    }, [organization, session, setLanguage]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'UpdateOrganization',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
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
                            handleDelete={handleDeleteLanguage}
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
                        display={display}
                        errors={translations.errorsWithTranslations}
                        isCreate={false}
                        loading={formik.isSubmitting}
                        onCancel={onCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </BaseForm>
        </>
    )
}