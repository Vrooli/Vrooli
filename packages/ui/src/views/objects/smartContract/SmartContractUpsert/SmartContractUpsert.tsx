import { DUMMY_ID, endpointGetSmartContractVersion, endpointPostSmartContractVersion, endpointPutSmartContractVersion, noopSubmit, orDefault, Session, SmartContractVersion, SmartContractVersionCreateInput, smartContractVersionTranslationValidation, SmartContractVersionUpdateInput, smartContractVersionValidation } from "@local/shared";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CodeInput } from "components/inputs/CodeInput/CodeInput";
import { StandardLanguage } from "components/inputs/CodeInputBase/CodeInputBase";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextInput } from "components/inputs/TranslatedTextInput/TranslatedTextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { getYou } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { SmartContractShape } from "utils/shape/models/smartContract";
import { shapeSmartContractVersion, SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { validateFormValues } from "utils/validateFormValues";
import { SmartContractFormProps, SmartContractUpsertProps } from "../types";

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
        listFor: {
            __typename: "SmartContractVersion" as const,
            id: DUMMY_ID,
        },
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

const transformSmartContractVersionValues = (values: SmartContractVersionShape, existing: SmartContractVersionShape, isCreate: boolean) =>
    isCreate ? shapeSmartContractVersion.create(values) : shapeSmartContractVersion.update(existing, values);

const SmartContractForm = ({
    disabled,
    dirty,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    versions,
    ...props
}: SmartContractFormProps) => {
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);
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
        validationSchema: smartContractVersionTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
    });

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<SmartContractVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "SmartContractVersion",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostSmartContractVersion,
        endpointUpdate: endpointPutSmartContractVersion,
    });
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "SmartContractVersion" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<SmartContractVersionCreateInput | SmartContractVersionUpdateInput, SmartContractVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformSmartContractVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="smart-contract-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateSmartContract" : "UpdateSmartContract")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
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
                        <TranslatedTextInput
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
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const SmartContractUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: SmartContractUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<SmartContractVersion, SmartContractVersionShape>({
        ...endpointGetSmartContractVersion,
        isCreate,
        objectType: "SmartContractVersion",
        overrideObject,
        transform: (existing) => smartContractInitialValues(session, existing),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformSmartContractVersionValues, smartContractVersionValidation)}
        >
            {(formik) => <SmartContractForm
                disabled={!(isCreate || canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                versions={(existing?.root as SmartContractShape)?.versions?.map(v => v.versionLabel) ?? []}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
