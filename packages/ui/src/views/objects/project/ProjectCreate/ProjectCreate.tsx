import { Grid, TextField } from "@mui/material";
import { ProjectVersion, ProjectVersionCreateInput, ResourceList } from "@shared/consts";
import { parseSearchParams } from "@shared/route";
import { uuid } from '@shared/uuid';
import { projectVersionTranslationValidation, projectVersionValidation } from '@shared/validation';
import { projectVersionCreate } from "api/generated/endpoints/projectVersion_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { GridSubmitButtons, LanguageInput, RelationshipButtons, ResourceListHorizontal, TagSelector, TopBar, VersionInput } from "components";
import { RelationshipsObject } from "components/inputs/types";
import { useFormik } from 'formik';
import { BaseForm } from "forms";
import { useCallback, useEffect, useMemo, useState } from "react";
import { defaultRelationships, defaultResourceList, getUserLanguages, shapeProjectVersion, TagShape, useCreateActions, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { checkIfLoggedIn } from "utils/authentication";
import { ProjectCreateProps } from "../types";

export const ProjectCreate = ({
    display = 'page',
    session,
    zIndex = 200,
}: ProjectCreateProps) => {
    const { onCancel, onCreated } = useCreateActions<ProjectVersion>();

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
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
    const [mutation] = useCustomMutation<ProjectVersion, ProjectVersionCreateInput>(projectVersionCreate);
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
                versionLabel: '1.0.0',
                versionNotes: '',
            }
        },
        validationSchema: projectVersionValidation.create({}),
        onSubmit: (values) => {
            mutationWrapper<ProjectVersion, ProjectVersionCreateInput>({
                mutation,
                input: shapeProjectVersion.create({
                    id: values.id,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
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
        fields: ['description', 'name'],
        formik,
        formikField: 'translationsCreate',
        validationSchema: projectVersionTranslationValidation.create({}),
    });

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'CreateProject',
                }}
            />
            <BaseForm onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
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
                            handleDelete={handleDeleteLanguage}
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
                            canUpdate={true}
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
                        display={display}
                        errors={translations.errorsWithTranslations}
                        isCreate={true}
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