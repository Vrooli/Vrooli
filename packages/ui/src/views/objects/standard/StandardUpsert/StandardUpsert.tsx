import { DUMMY_ID, endpointGetStandardVersion, endpointPostStandardVersion, endpointPutStandardVersion, orDefault, Session, StandardVersion, StandardVersionCreateInput, standardVersionTranslationValidation, StandardVersionUpdateInput, standardVersionValidation } from "@local/shared";
import { useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { StandardInput } from "components/inputs/standards/StandardInput/StandardInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { StandardFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { InputTypeOptions } from "utils/consts";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeStandardVersion, StandardVersionShape } from "utils/shape/models/standardVersion";
import { StandardUpsertProps } from "../types";

export const standardInitialValues = (
    session: Session | undefined,
    existing?: Partial<StandardVersion> | null | undefined,
): StandardVersionShape => ({
    __typename: "StandardVersion" as const,
    id: DUMMY_ID,
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    isFile: false,
    standardType: InputTypeOptions[0].value,
    props: JSON.stringify({}),
    default: JSON.stringify({}),
    yup: JSON.stringify({}),
    resourceList: {
        __typename: "ResourceList" as const,
        id: DUMMY_ID,
    },
    versionLabel: "1.0.0",
    ...existing,
    root: {
        __typename: "Standard" as const,
        id: DUMMY_ID,
        isInternal: false,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session)!.id! },
        parent: null,
        permissions: JSON.stringify({}),
        tags: [],
        ...existing?.root,
    },
    translations: orDefault(existing?.translations, [{
        __typename: "StandardVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        jsonVariable: null, //TODO
        name: "",
    }]),
});

const transformStandardValues = (values: StandardVersionShape, existing: StandardVersionShape, isCreate: boolean) =>
    isCreate ? shapeStandardVersion.create(values) : shapeStandardVersion.update(existing, values);

const validateStandardValues = async (values: StandardVersionShape, existing: StandardVersionShape, isCreate: boolean) => {
    const transformedValues = transformStandardValues(values, existing, isCreate);
    const validationSchema = standardVersionValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

const StandardForm = forwardRef<BaseFormRef | undefined, StandardFormProps>(({
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
    const { palette } = useTheme();
    const { t } = useTranslation();

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
        fields: ["description"],
        validationSchema: standardVersionTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
    });

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
                        objectType={"Standard"}
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
                            name="description"
                            maxChars={2048}
                            minRows={4}
                            maxRows={8}
                            placeholder={t("Description")}
                        />
                    </FormSection>
                    <StandardInput fieldName="preview" />
                    <ResourceListHorizontalInput
                        isCreate={true}
                        parent={{ __typename: "StandardVersion", id: values.id }}
                    />
                    <TagSelector name="root.tags" />
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

export const StandardUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: StandardUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<StandardVersion, StandardVersionShape>({
        ...endpointGetStandardVersion,
        objectType: "StandardVersion",
        overrideObject,
        transform: (existing) => standardInitialValues(session, existing),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<StandardVersion, StandardVersionCreateInput, StandardVersionUpdateInput>({
        display,
        endpointCreate: endpointPostStandardVersion,
        endpointUpdate: endpointPutStandardVersion,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="standard-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateStandard" : "UpdateStandard")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<StandardVersionCreateInput | StandardVersionUpdateInput, StandardVersion>({
                        fetch,
                        inputs: transformStandardValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateStandardValues(values, existing, isCreate)}
            >
                {(formik) => <StandardForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    versions={existing?.root?.versions?.map(v => v.versionLabel) ?? []}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
