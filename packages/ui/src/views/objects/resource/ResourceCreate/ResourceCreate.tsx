import { Grid } from "@mui/material";
import { Resource, ResourceCreateInput } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { resourceTranslationValidation, resourceValidation } from '@shared/validation';
import { resourceCreate } from "api/generated/endpoints/resource_create";
import { useCustomMutation } from "api/hooks";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useMemo } from "react";
import { checkIfLoggedIn } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { ResourceCreateProps } from "../types";

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
        validationSchema: resourceTranslationValidation.create({}),
    });

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
            <BaseForm onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
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