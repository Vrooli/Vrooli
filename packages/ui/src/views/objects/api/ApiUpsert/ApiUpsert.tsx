import { ApiVersion, ApiVersionCreateInput, apiVersionTranslationValidation, ApiVersionUpdateInput, apiVersionValidation, DUMMY_ID, endpointGetApiVersion, endpointPostApiVersion, endpointPutApiVersion, noopSubmit, orDefault, Session } from "@local/shared";
import { Button, Grid, InputAdornment, Stack } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CodeInputBase, StandardLanguage } from "components/inputs/CodeInputBase/CodeInputBase";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextInput } from "components/inputs/TranslatedTextInput/TranslatedTextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { CompleteIcon, LinkIcon } from "icons";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { getYou } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { ApiShape } from "utils/shape/models/api";
import { ApiVersionShape, shapeApiVersion } from "utils/shape/models/apiVersion";
import { validateFormValues } from "utils/validateFormValues";
import { ApiFormProps, ApiUpsertProps } from "../types";

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
        owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
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

const transformApiVersionValues = (values: ApiVersionShape, existing: ApiVersionShape, isCreate: boolean) =>
    isCreate ? shapeApiVersion.create(values) : shapeApiVersion.update(existing, values);

const ApiForm = ({
    disabled,
    dirty,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onClose,
    values,
    versions,
    ...props
}: ApiFormProps) => {
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);
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

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<ApiVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "ApiVersion",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<ApiVersion, ApiVersionCreateInput, ApiVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostApiVersion,
        endpointUpdate: endpointPutApiVersion,
    });
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "ApiVersion" });

    const onSubmit = useSubmitHelper<ApiVersionCreateInput | ApiVersionUpdateInput, ApiVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformApiVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    return (
        <MaybeLargeDialog
            display={display}
            id="api-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateApi" : "UpdateApi")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
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
                        <TranslatedTextInput
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
                            as={TextInput}
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
                                    as={TextInput}
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
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const ApiUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ApiUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<ApiVersion, ApiVersionShape>({
        ...endpointGetApiVersion,
        isCreate,
        objectType: "ApiVersion",
        overrideObject,
        transform: (data) => apiInitialValues(session, data),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformApiVersionValues, apiVersionValidation)}
        >
            {(formik) => <ApiForm
                disabled={!(isCreate || canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                versions={(existing?.root as ApiShape)?.versions?.map(v => v.versionLabel) ?? []}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
