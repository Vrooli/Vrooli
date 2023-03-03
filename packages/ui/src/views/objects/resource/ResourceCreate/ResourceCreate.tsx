import { Checkbox, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { resourceValidation, resourceTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, defaultResourceList, getUserLanguages, handleTranslationBlur, handleTranslationChange, removeTranslation, shapeResource, TagShape, useCreateActions, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { ResourceCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, TopBar } from "components";
import { uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { checkIfLoggedIn, getCurrentUser } from "utils/authentication";
import { Resource, ResourceCreateInput, ResourceList } from "@shared/consts";
import { resourceCreate } from "api/generated/endpoints/resource_create";
import { parseSearchParams } from "@shared/route";

export const ResourceCreate = ({
    display = 'dialog',
    listId,
    session,
    zIndex = 200,
}: ResourceCreateProps) => {
    const { onCancel, onCreated } = useCreateActions<Resource>();

    // Handle create
    const [mutation] = useCustomMutation<Resource, ResourceCreateInput>(resourceCreate);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                description: '',
                name: '',
            }]
        },
        validationSchema: resourceValidation.create({}),
        onSubmit: (values) => {
            // mutationWrapper<Resource, ResourceCreateInput>({
            //     mutation,
            //     input: shapeResource.create({
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
        fields: ['description', 'name'],
        formik,
        formikField: 'translationsCreate',
        language,
        validationSchema: resourceTranslationValidation.create({}),
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
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'CreateResource',
                }}
            />
            <form onSubmit={formik.handleSubmit} style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
            }}
            >
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
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
                        display={display}
                        errors={translations.errorsWithTranslations}
                        isCreate={true}
                        loading={formik.isSubmitting}
                        onCancel={onCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </form>
        </>
    )
}