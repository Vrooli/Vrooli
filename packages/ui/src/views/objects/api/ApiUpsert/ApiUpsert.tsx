import { ApiVersion, ApiVersionCreateInput, apiVersionTranslationValidation, ApiVersionUpdateInput, apiVersionValidation, DUMMY_ID, endpointGetApiVersion, endpointPostApiVersion, endpointPutApiVersion, orDefault, Session } from "@local/shared";
import { Button, Grid, InputAdornment, Stack, TextField } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CodeInputBase, StandardLanguage } from "components/inputs/CodeInputBase/CodeInputBase";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ApiFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { CompleteIcon, LinkIcon } from "icons";
import { forwardRef, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ApiVersionShape, shapeApiVersion } from "utils/shape/models/apiVersion";
import { ApiUpsertProps } from "../types";

const apiInitialValues = (
    session: Session | undefined,
    existing?: Partial<ApiVersion> | null | undefined,
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
        listFor: {
            __typename: "ApiVersion" as const,
            id: DUMMY_ID,
        },
    },
    versionLabel: "1.0.0",
    ...existing,
    root: {
        __typename: "Api" as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session)!.id! },
        tags: [],
        ...existing?.root,
    },
    translations: orDefault(existing?.translations, [{
        __typename: "ApiVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        details: "",
        name: "",
        summary: "",
    }]),
});

const transformApiValues = (values: ApiVersionShape, existing: ApiVersionShape, isCreate: boolean) =>
    isCreate ? shapeApiVersion.create(values) : shapeApiVersion.update(existing, values);

const validateApiValues = async (values: ApiVersionShape, existing: ApiVersionShape, isCreate: boolean) => {
    const transformedValues = transformApiValues(values, existing, isCreate);
    const validationSchema = apiVersionValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

const ApiForm = forwardRef<BaseFormRef | undefined, ApiFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    versions,
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
        validationSchema: apiVersionTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
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
                    />
                    <FormSection>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Name")}
                            language={language}
                            name="name"
                        />
                        <TranslatedRichInput
                            language={language}
                            name="summary"
                            maxChars={1024}
                            minRows={4}
                            maxRows={8}
                            placeholder={t("Summary")}
                        />
                        <TranslatedRichInput
                            language={language}
                            name="details"
                            maxChars={8192}
                            minRows={4}
                            maxRows={8}
                            placeholder={t("Details")}
                        />
                    </FormSection>
                    <FormSection>
                        <Field
                            fullWidth
                            name="callLink"
                            label={"Endpoint URL"}
                            placeholder={"https://example.com"}
                            as={TextField}
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <LinkIcon />
                                    </InputAdornment>
                                ),
                            }}
                        />
                        {/* Selector for documentation URL or text */}
                        <Grid item xs={12} pb={2}>
                            {/* Title with help text */}
                            <Title
                                title="Use URL for schema?"
                                help={"Is your API's [OpenAPI](https://swagger.io/specification/) or [GraphQL](https://graphql.org/) schema available at a URL? If so, select \"Yes\" and enter the URL below. If not, select \"No\" and enter the schema text below.\n\n*This field is not required, but recommended.*"}
                                variant="subheader"
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
                                />
                            )
                        }
                    </FormSection>
                    <ResourceListHorizontalInput
                        isCreate={true}
                        parent={{ __typename: "ApiVersion", id: values.id }}
                    />
                    <TagSelector
                        name="root.tags"
                    />
                    <VersionInput
                        fullWidth
                        versions={versions}
                    />
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});

export const ApiUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: ApiUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<ApiVersion, ApiVersionShape>({
        ...endpointGetApiVersion,
        objectType: "ApiVersion",
        overrideObject,
        transform: (data) => apiInitialValues(session, data),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<ApiVersion, ApiVersionCreateInput, ApiVersionUpdateInput>({
        display,
        endpointCreate: endpointPostApiVersion,
        endpointUpdate: endpointPutApiVersion,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="api-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateApi" : "UpdateApi")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<ApiVersionCreateInput | ApiVersionUpdateInput, ApiVersion>({
                        fetch,
                        inputs: transformApiValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateApiValues(values, existing, isCreate)}
            >
                {(formik) => <ApiForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    versions={[]}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
