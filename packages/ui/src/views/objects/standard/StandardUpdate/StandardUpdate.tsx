import { Grid, TextField } from "@mui/material"
import { useCustomMutation, useCustomLazyQuery } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { StandardUpdateProps } from "../types";
import { mutationWrapper } from 'api/utils';
import { useFormik } from 'formik';
import { defaultRelationships, defaultResourceList, getUserLanguages, parseSingleItemUrl, PubSub, shapeStandardVersion, TagShape, usePromptBeforeUnload, useTranslatedFields, useUpdateActions } from "utils";
import { GridSubmitButtons, LanguageInput, RelationshipButtons, ResourceListHorizontal, TagSelector, TopBar } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindVersionInput, ResourceList, Standard, StandardUpdateInput, StandardVersion, StandardVersionUpdateInput } from "@shared/consts";
import { standardVersionTranslationValidation, standardVersionValidation } from "@shared/validation";
import { standardVersionFindOne } from "api/generated/endpoints/standardVersion_findOne";
import { standardVersionUpdate } from "api/generated/endpoints/standardVersion_update";
import { BaseForm } from "forms";

export const StandardUpdate = ({
    display = 'page',
    session,
    zIndex = 200,
}: StandardUpdateProps) => {
    const { onCancel, onUpdated } = useUpdateActions<StandardVersion>();

    // Fetch existing data
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: standardVersion, loading }] = useCustomLazyQuery<StandardVersion, FindVersionInput>(standardVersionFindOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (urlData.id || urlData.idRoot) getData({ variables: urlData });
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: 'Error' });
    }, [getData, urlData])

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: false, //TODO
            isPrivate: standardVersion?.isPrivate ?? false,
            owner: standardVersion?.root?.owner ?? null,
            parent: null,
            // parent: standard?.parent ?? null, TODO
            project: null // TODO
        });
        setResourceList(standardVersion?.resourceList ?? { id: uuid() } as any);
        setTags(standardVersion?.root?.tags ?? []);
    }, [standardVersion]);

    // Handle update
    const [mutation] = useCustomMutation<StandardVersion, StandardVersionUpdateInput>(standardVersionUpdate);
    const formik = useFormik({
        initialValues: {
            translationsUpdate: standardVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: null, //TODO
            }],
            versionInfo: {
                versionLabel: standardVersion?.versionLabel ?? '1.0.0',
                versionNotes: '',
            }
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: standardVersionValidation.update({}),
        onSubmit: (values) => {
            if (!standardVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadStandard', severity: 'Error' });
                return;
            }
            // Update
            mutationWrapper<StandardVersion, StandardVersionUpdateInput>({
                mutation,
                input: shapeStandardVersion.update(standardVersion, {
                    id: standardVersion.id,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    resourceList: resourceList,
                    root: {
                        id: standardVersion.root.id,
                        isInternal: false,
                        isPrivate: relationships.isPrivate,
                        name: standardVersion.root.name,
                        owner: relationships.owner,
                        permissions: JSON.stringify({}),
                        tags: tags,
                    },
                    standardType: standardVersion.standardType,
                    translations: values.translationsUpdate,
                    ...values.versionInfo,
                } as any), //TODO
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
        fields: ['description'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: standardVersionTranslationValidation.update({}),
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'UpdateStandard',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipButtons
                            isEditing={true}
                            objectType={'Standard'}
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
                    {/* TODO versioning */}
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
                    <Grid item xs={12} marginBottom={4}>
                        <TagSelector
                            handleTagsUpdate={handleTagsUpdate}
                            session={session}
                            tags={tags}
                        />
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