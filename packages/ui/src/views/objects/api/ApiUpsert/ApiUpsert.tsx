import { ApiVersion, ApiVersionCreateInput, apiVersionTranslationValidation, ApiVersionUpdateInput, apiVersionValidation, DUMMY_ID, endpointGetApiVersion, endpointPostApiVersion, endpointPutApiVersion, LINKS, noopSubmit, orDefault, Session } from "@local/shared";
import { Button, Divider, Grid, InputAdornment, Stack, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CodeInput, CodeLanguage } from "components/inputs/CodeInput/CodeInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TextInput, TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListInput } from "components/lists/resource/ResourceList/ResourceList";
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
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { SearchPageTabOption } from "utils/search/objectToSearch";
import { ApiShape } from "utils/shape/models/api";
import { ApiVersionShape, shapeApiVersion } from "utils/shape/models/apiVersion";
import { validateFormValues } from "utils/validateFormValues";
import { SearchExistingButton } from "../../../../components/buttons/SearchExistingButton/SearchExistingButton";
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
    schemaText: "",
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

function ApiForm({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onClose,
    values,
    versions,
    ...props
}: ApiFormProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { palette } = useTheme();

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
        validationSchema: apiVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const [hasDocUrl, setHasDocUrl] = useState(false);

    const { handleCancel, handleCompleted } = useUpsertActions<ApiVersion>({
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
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "ApiVersion" });

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
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.Api}"`}
                text="Search existing apis"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <RelationshipList
                            isEditing={true}
                            objectType={"Api"}
                            sx={{ marginBottom: 2 }}
                        />
                        <FormSection sx={{ overflowX: "hidden", marginBottom: 2 }}>
                            <TranslatedTextInput
                                autoFocus
                                fullWidth
                                isRequired={true}
                                label={t("Name")}
                                language={language}
                                name="name"
                                placeholder={t("NamePlaceholder")}
                            />
                            <TranslatedRichInput
                                isRequired={false}
                                language={language}
                                name="summary"
                                maxChars={1024}
                                minRows={4}
                                maxRows={8}
                                placeholder={t("Summary")}
                            />
                            <TranslatedRichInput
                                isRequired={false}
                                language={language}
                                name="details"
                                maxChars={8192}
                                minRows={4}
                                maxRows={8}
                                placeholder={t("Details")}
                            />
                            <LanguageInput
                                currentLanguage={language}
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                languages={languages}
                                sx={{ flexDirection: "row-reverse" }}
                            />
                        </FormSection>
                        <TagSelector name="root.tags" sx={{ marginBottom: 2 }} />
                        <VersionInput
                            fullWidth
                            versions={versions}
                            sx={{ marginBottom: 2 }}
                        />
                        <ResourceListInput
                            horizontal
                            isCreate={true}
                            parent={{ __typename: "ApiVersion", id: values.id }}
                            sxs={{ list: { marginBottom: 2 } }}
                        />
                    </ContentCollapse>
                    <Divider />
                    <ContentCollapse title="Api info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
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
                                    <CodeInput
                                        disabled={false}
                                        limitTo={[CodeLanguage.Json, CodeLanguage.Graphql]}
                                        name="schemaText"
                                    />
                                )
                            }
                        </FormSection>
                    </ContentCollapse>
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
}

export function ApiUpsert({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ApiUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<ApiVersion, ApiVersionShape>({
        ...endpointGetApiVersion,
        isCreate,
        objectType: "ApiVersion",
        overrideObject,
        transform: (data) => apiInitialValues(session, data),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformApiVersionValues, apiVersionValidation)}
        >
            {(formik) => <ApiForm
                disabled={!(isCreate || permissions.canUpdate)}
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
}
