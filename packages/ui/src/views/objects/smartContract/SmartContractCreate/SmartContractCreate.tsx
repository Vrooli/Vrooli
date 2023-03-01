import { Checkbox, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { smartContractVersionValidation, smartContractVersionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, defaultResourceList, getUserLanguages, handleTranslationBlur, handleTranslationChange, removeTranslation, shapeSmartContractVersion, TagShape, useCreateActions, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { SmartContractCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector } from "components";
import { uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { checkIfLoggedIn, getCurrentUser } from "utils/authentication";
import { SmartContractVersion, SmartContractVersionCreateInput, ResourceList } from "@shared/consts";
import { smartContractVersionCreate } from "api/generated/endpoints/smartContractVersion_create";
import { parseSearchParams } from "@shared/route";

export const SmartContractCreate = ({
    session,
    zIndex = 200,
}: SmartContractCreateProps) => {
    const { onCancel, onCreated } = useCreateActions<SmartContractVersion>();

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
    const [mutation] = useCustomMutation<SmartContractVersion, SmartContractVersionCreateInput>(smartContractVersionCreate);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: '',
                name: '',
            }]
        },
        validationSchema: smartContractVersionValidation.create({}),
        onSubmit: (values) => {
            // mutationWrapper<SmartContractVersion, SmartContractVersionCreateInput>({
            //     mutation,
            //     input: shapeSmartContractVersion.create({
            //         id: values.id,

            //     }),
            //     onSuccess: (data) => { onCreated(data) },
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const translations = useTranslatedFields({
        fields: ['description', 'jsonVariable', 'name'],
        formik, 
        formikField: 'translationsCreate', 
        language, 
        validationSchema: smartContractVersionTranslationValidation.create({}),
    });
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

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
        }}
        >
            <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                <Grid item xs={12}>
                    <PageTitle titleKey='CreateSmartContract' />
                </Grid>
                <Grid item xs={12} mb={4}>
                    <RelationshipButtons
                        isEditing={true}
                        objectType={'SmartContract'}
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
                {/* TODO */}
                <GridSubmitButtons
                    disabledSubmit={!isLoggedIn}
                    errors={translations.errorsWithTranslations}
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