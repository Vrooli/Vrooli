import { Checkbox, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material";
import { Organization, OrganizationCreateInput, ResourceList } from "@shared/consts";
import { parseSearchParams } from "@shared/route";
import { uuid } from '@shared/uuid';
import { organizationTranslationValidation, organizationValidation } from '@shared/validation';
import { organizationCreate } from "api/generated/endpoints/organization_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RelationshipButtons } from "components/inputs/RelationshipButtons/RelationshipButtons";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { ResourceListHorizontal } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { checkIfLoggedIn } from "utils/authentication/session";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getUserLanguages } from "utils/display/translationTools";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { shapeOrganization } from "utils/shape/models/organization";
import { TagShape } from "utils/shape/models/tag";
import { OrganizationCreateProps } from "../types";

export const OrganizationCreate = ({
    display = 'page',
    zIndex = 200,
}: OrganizationCreateProps) => {
    const session = useContext(SessionContext);

    const { onCancel, onCreated } = useCreateActions<Organization>();

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useCustomMutation<Organization, OrganizationCreateInput>(organizationCreate);
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
        validationSchema: organizationValidation.create({}),
        onSubmit: (values) => {
            mutationWrapper<Organization, OrganizationCreateInput>({
                mutation,
                input: shapeOrganization.create({
                    id: values.id,
                    isOpenToNewMembers: values.isOpenToNewMembers,
                    isPrivate: relationships.isPrivate,
                    resourceList: resourceList,
                    tags,
                    translations: values.translationsCreate,
                }),
                onSuccess: (data) => { onCreated(data) },
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
        formikField: 'translationsCreate',
        validationSchema: organizationTranslationValidation.create({}),
    });

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateOrganization',
                }}
            />
            <BaseForm onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipButtons
                            isEditing={true}
                            objectType={'Organization'}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
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
                            loading={false}
                            mutate={false}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TagSelector
                            handleTagsUpdate={handleTagsUpdate}
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
                </Grid>
            </BaseForm>
            <GridSubmitButtons
                disabledSubmit={!isLoggedIn}
                display={display}
                errors={translations.errorsWithTranslations}
                isCreate={true}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </>
    )
}