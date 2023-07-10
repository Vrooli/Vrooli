import { ApiVersion, apiVersionTranslationValidation, apiVersionValidation, CompleteIcon, DUMMY_ID, orDefault, Session } from "@local/shared";
import { Button, Grid, Stack, TextField } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { CodeInputBase, StandardLanguage } from "components/inputs/CodeInputBase/CodeInputBase";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { Title } from "components/text/Title/Title";
import { Field } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ApiFormProps } from "forms/types";
import { forwardRef, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ApiVersionShape, shapeApiVersion } from "utils/shape/models/apiVersion";

export const apiInitialValues = (
    session: Session | undefined,
    existing?: ApiVersion | null | undefined,
): ApiVersionShape => ({
    __typename: "ApiVersion" as const,
    id: DUMMY_ID,
    callLink: "",
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    resourceList: {
        __typename: "ResourceList" as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: "Api" as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session)!.id! },
        tags: [],
    },
    versionLabel: "1.0.0",
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "ApiVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        details: "",
        name: "",
        summary: "",
    }]),
});

export const transformApiValues = (values: ApiVersionShape, existing?: ApiVersionShape) => {
    return existing === undefined
        ? shapeApiVersion.create(values)
        : shapeApiVersion.update(existing, values);
};

export const validateApiValues = async (values: ApiVersionShape, existing?: ApiVersionShape) => {
    const transformedValues = transformApiValues(values, existing);
    const validationSchema = apiVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const ApiForm = forwardRef<BaseFormRef | undefined, ApiFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    versions,
    zIndex,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    // useEffect(() => {
    //     const params = parseSearchParams();
    //     if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
    //     else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    // }, []);

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["details", "name", "summary"],
        validationSchema: apiVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    const [hasDocUrl, setHasDocUrl] = useState(false);

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={700}
                ref={ref}
            >
                <FormContainer>
                    <RelationshipList
                        isEditing={true}
                        objectType={"Api"}
                        zIndex={zIndex}
                    />
                    <FormSection>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                            zIndex={zIndex + 1}
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Name")}
                            language={language}
                            name="name"
                        />
                        <TranslatedMarkdownInput
                            language={language}
                            name="summary"
                            maxChars={1024}
                            minRows={4}
                            maxRows={8}
                            placeholder={t("Summary")}
                            zIndex={zIndex}
                        />
                        <TranslatedMarkdownInput
                            language={language}
                            name="details"
                            maxChars={8192}
                            minRows={4}
                            maxRows={8}
                            placeholder={t("Details")}
                            zIndex={zIndex}
                        />
                    </FormSection>
                    <FormSection>
                        <Field
                            fullWidth
                            name="callLink"
                            label={"Endpoint URL"}
                            helperText={"The full URL to the endpoint"}
                            as={TextField}
                        />
                        {/* Selector for documentation URL or text */}
                        <Grid item xs={12} pb={2}>
                            {/* Title with help text */}
                            <Title
                                title="Use URL for schema?"
                                help={"Is your API's [OpenAPI](https://swagger.io/specification/) or [GraphQL](https://graphql.org/) schema available at a URL? If so, select \"Yes\" and enter the URL below. If not, select \"No\" and enter the schema text below.\n\n*This field is not required, but recommended.*"}
                                variant="subheader"
                                zIndex={zIndex}
                            />
                            {/* Yes/No buttons */}
                            <Stack direction="row" display="flex" alignItems="center" justifyContent="center" spacing={1} >
                                <Button
                                    fullWidth
                                    color="secondary"
                                    onClick={() => setHasDocUrl(true)}
                                    variant="outlined"
                                    startIcon={hasDocUrl === true ? <CompleteIcon /> : undefined}
                                >{t("Yes")}</Button>
                                <Button
                                    fullWidth
                                    color="secondary"
                                    onClick={() => setHasDocUrl(false)}
                                    variant="outlined"
                                    startIcon={hasDocUrl === false ? <CompleteIcon /> : undefined}
                                >{t("No")}</Button>
                            </Stack >
                        </Grid >
                        {
                            hasDocUrl === true && (
                                <Field
                                    fullWidth
                                    name="documentationLink"
                                    label={"Schema URL (Optional)"}
                                    helperText={"The full URL to the schema"}
                                    as={TextField}
                                />
                            )
                        }
                        {
                            hasDocUrl === false && (
                                <CodeInputBase
                                    disabled={false}
                                    limitTo={[StandardLanguage.Json, StandardLanguage.Graphql]}
                                    name="schemaText"
                                    zIndex={zIndex}
                                />
                            )
                        }
                    </FormSection>
                    <ResourceListHorizontalInput
                        isCreate={true}
                        zIndex={zIndex}
                    />
                    <TagSelector
                        name="root.tags"
                        zIndex={zIndex}
                    />
                    <VersionInput
                        fullWidth
                        versions={versions}
                    />
                </FormContainer>
                <GridSubmitButtons
                    display={display}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                    zIndex={zIndex}
                />
            </BaseForm>
        </>
    );
});
