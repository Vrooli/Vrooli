import { Grid, TextField } from "@mui/material";
import { FindByIdInput, Resource, ResourceUpdateInput } from "@shared/consts";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { resourceTranslationValidation, resourceValidation } from '@shared/validation';
import { resourceFindOne } from "api/generated/endpoints/resource_findOne";
import { resourceUpdate } from "api/generated/endpoints/resource_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useContext, useEffect, useMemo } from "react";
import { getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ResourceUpdateProps } from "../types";

export const ResourceUpdate = ({
    display = 'dialog',
    listId,
    zIndex = 200,
}: ResourceUpdateProps) => {
    const session = useContext(SessionContext);

    const { onCancel, onUpdated } = useUpdateActions<Resource>();

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: resource, loading }] = useCustomLazyQuery<Resource, FindByIdInput>(resourceFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    // Handle update
    const [mutation] = useCustomMutation<Resource, ResourceUpdateInput>(resourceUpdate);
    const formik = useFormik({
        initialValues: {
            id: resource?.id ?? uuid(),
            translationsUpdate: resource?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                name: '',
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: resourceValidation.update({}),
        onSubmit: (values) => {
            if (!resource) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadResource', severity: 'Error' });
                return;
            }
            // mutationWrapper<Resource, ResourceUpdateInput>({
            //     mutation,
            //     input: shapeResource.update(resource, {
            //         id: resource.id,
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
        fields: ['description', 'name'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: resourceTranslationValidation.update({}),
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'UpdateResource',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
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
                            id="description"
                            name="description"
                            label="Description"
                            multiline
                            minRows={4}
                            value={translations.description}
                            onBlur={onTranslationBlur}
                            onChange={onTranslationChange}
                            error={translations.touchedDescription && Boolean(translations.errorDescription)}
                            helperText={translations.touchedDescription && translations.errorDescription}
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