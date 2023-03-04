import { Box, Checkbox, CircularProgress, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material"
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiUpdateProps } from "../types";
import { mutationWrapper } from 'api/utils';
import { apiVersionValidation, apiVersionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { defaultRelationships, defaultResourceList, getPreferredLanguage, getUserLanguages, parseSingleItemUrl, PubSub, shapeApiVersion, TagShape, usePromptBeforeUnload, useTranslatedFields, useUpdateActions } from "utils";
import { GridSubmitButtons, LanguageInput, RelationshipButtons, ResourceListHorizontal, TagSelector, TopBar } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindByIdInput, ApiVersion, ApiVersionUpdateInput, ResourceList } from "@shared/consts";
import { apiVersionFindOne } from "api/generated/endpoints/apiVersion_findOne";
import { apiVersionUpdate } from "api/generated/endpoints/apiVersion_update";
import { BaseForm } from "forms";

export const ApiUpdate = ({
    display = 'page',
    session,
    zIndex = 200,
}: ApiUpdateProps) => {
    const { onCancel, onUpdated } = useUpdateActions<ApiVersion>();

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: apiVersion, loading }] = useCustomLazyQuery<ApiVersion, FindByIdInput>(apiVersionFindOne);
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
    const [mutation] = useCustomMutation<ApiVersion, ApiVersionUpdateInput>(apiVersionUpdate);
    const formik = useFormik({
        initialValues: {
            id: apiVersion?.id ?? uuid(),
            translationsUpdate: apiVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                details: '',
                name: '',
                summary: '',
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: apiVersionValidation.update({}),
        onSubmit: (values) => {
            if (!apiVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadApi', severity: 'Error' });
                return;
            }
            // mutationWrapper<ApiVersion, ApiVersionUpdateInput>({
            //     mutation,
            //     input: shapeApiVersion.update(apiVersion, {
            //         id: apiVersion.id,
            //         isOpenToNewMembers: values.isOpenToNewMembers,
            //         isPrivate: relationships.isPrivate,
            //         resourceList: resourceList,
            //         tags: tags,
            //         translations: values.translationsUpdate,
            //     }),
            //     onSuccess: (data) => { onUpdated(data) },
            //     onError: () => { formik.setSubmitting(false) },
            // })
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
        fields: ['details', 'name', 'summary'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: apiVersionTranslationValidation.update({}),
    });

    useEffect(() => {
        setRelationships({
            isComplete: false,
            isPrivate: apiVersion?.isPrivate ?? false,
            owner: null,
            parent: null,
            project: null,
        });
        setResourceList(apiVersion?.resourceList ?? { id: uuid() } as any);
        setTags(apiVersion?.root?.tags ?? []);
        if (apiVersion?.translations?.length) {
            setLanguage(getPreferredLanguage(apiVersion.translations.map(t => t.language), getUserLanguages(session)));
        }
    }, [apiVersion, session, setLanguage]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'UpdateApi',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipButtons
                            isEditing={true}
                            objectType={'Api'}
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
                    {/* TODO */}
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