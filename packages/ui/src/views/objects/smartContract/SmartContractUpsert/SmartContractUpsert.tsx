import { DUMMY_ID, endpointGetSmartContractVersion, endpointPostSmartContractVersion, endpointPutSmartContractVersion, orDefault, Session, SmartContractVersion, SmartContractVersionCreateInput, smartContractVersionTranslationValidation, SmartContractVersionUpdateInput, smartContractVersionValidation } from "@local/shared";
import { useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CodeInput } from "components/inputs/CodeInput/CodeInput";
import { StandardLanguage } from "components/inputs/CodeInputBase/CodeInputBase";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { SmartContractFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { SmartContractShape } from "utils/shape/models/smartContract";
import { shapeSmartContractVersion, SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { SmartContractUpsertProps } from "../types";

const smartContractInitialValues = (
    session: Session | undefined,
    existing?: Partial<SmartContractVersion> | undefined,
): SmartContractVersionShape => ({
    __typename: "SmartContractVersion" as const,
    id: DUMMY_ID,
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    content: "",
    contractType: "",
    resourceList: {
        __typename: "ResourceList" as const,
        id: DUMMY_ID,
    },
    versionLabel: "1.0.0",
    ...existing,
    root: {
        __typename: "SmartContract" as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
        parent: null,
        tags: [],
        ...existing?.root,
    },
    translations: orDefault(existing?.translations, [{
        __typename: "SmartContractVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        jsonVariable: "",
        name: "",
    }]),
});

const transformSmartContractValues = (values: SmartContractVersionShape, existing: SmartContractVersionShape, isCreate: boolean) =>
    isCreate ? shapeSmartContractVersion.create(values) : shapeSmartContractVersion.update(existing, values);

const validateSmartContractValues = async (values: SmartContractVersionShape, existing: SmartContractVersionShape, isCreate: boolean) => {
    const transformedValues = transformSmartContractValues(values, existing, isCreate);
    const validationSchema = smartContractVersionValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

const SmartContractForm = forwardRef<BaseFormRef | undefined, SmartContractFormProps>(({
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
        fields: ["description", "jsonVariable", "name"],
        validationSchema: smartContractVersionTranslationValidation[isCreate ? "create" : "update"]({}),
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
                        objectType={"SmartContract"}
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
                    <CodeInput
                        disabled={false}
                        limitTo={[StandardLanguage.Solidity, StandardLanguage.Haskell]}
                        name="content"
                    />
                    <ResourceListHorizontalInput
                        isCreate={true}
                        parent={{ __typename: "SmartContractVersion", id: values.id }}
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

export const SmartContractUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: SmartContractUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<SmartContractVersion, SmartContractVersionShape>({
        ...endpointGetSmartContractVersion,
        objectType: "SmartContractVersion",
        overrideObject,
        transform: (existing) => smartContractInitialValues(session, existing),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput>({
        display,
        endpointCreate: endpointPostSmartContractVersion,
        endpointUpdate: endpointPutSmartContractVersion,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="smart-contract-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateSmartContract" : "UpdateSmartContract")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<SmartContractVersionCreateInput | SmartContractVersionUpdateInput, SmartContractVersion>({
                        fetch,
                        inputs: transformSmartContractValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateSmartContractValues(values, existing, isCreate)}
            >
                {(formik) => <SmartContractForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    versions={(existing?.root as SmartContractShape)?.versions?.map(v => v.versionLabel) ?? []}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
