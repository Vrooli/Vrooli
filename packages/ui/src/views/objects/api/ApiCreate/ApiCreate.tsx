import { Grid } from "@mui/material";
import { ApiVersion, ApiVersionCreateInput, ResourceList } from "@shared/consts";
import { parseSearchParams } from "@shared/route";
import { uuid } from '@shared/uuid';
import { apiVersionTranslationValidation, apiVersionValidation } from '@shared/validation';
import { apiVersionCreate } from "api/generated/endpoints/apiVersion_create";
import { useCustomMutation } from "api/hooks";
import { GridSubmitButtons, LanguageInput, RelationshipButtons, TopBar } from "components";
import { RelationshipsObject } from "components/inputs/types";
import { useFormik } from 'formik';
import { BaseForm } from "forms";
import { useCallback, useEffect, useMemo, useState } from "react";
import { defaultRelationships, defaultResourceList, getUserLanguages, TagShape, useCreateActions, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { checkIfLoggedIn } from "utils/authentication";
import { ApiCreateProps } from "../types";

export const ApiCreate = ({
    display = 'page',
    session,
    zIndex = 200,
}: ApiCreateProps) => {
    const { onCancel, onCreated } = useCreateActions<ApiVersion>();

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
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useCustomMutation<ApiVersion, ApiVersionCreateInput>(apiVersionCreate);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                details: '',
                name: '',
                summary: '',
            }]
        },
        validationSchema: apiVersionValidation.create({}),
        onSubmit: (values) => {
            // mutationWrapper<ApiVersion, ApiVersionCreateInput>({
            //     mutation,
            //     input: shapeApiVersion.create({
            //         id: values.id,

            //     }),
            //     onSuccess: (data) => { onCreated(data) },
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
        formikField: 'translationsCreate',
        validationSchema: apiVersionTranslationValidation.create({}),
    });

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'CreateApi',
                }}
            />
            <BaseForm onSubmit={formik.handleSubmit}>
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
                            translations={formik.values.translationsCreate}
                            zIndex={zIndex}
                        />
                    </Grid>
                    {/* TODO */}
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