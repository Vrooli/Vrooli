import { DUMMY_ID, endpointGetStandardVersion, endpointPostStandardVersion, endpointPutStandardVersion, noopSubmit, orDefault, Session, StandardVersion, StandardVersionCreateInput, standardVersionTranslationValidation, StandardVersionUpdateInput, standardVersionValidation } from "@local/shared";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { StandardInput } from "components/inputs/standards/StandardInput/StandardInput";
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
import { InputTypeOptions } from "utils/consts";
import { getYou } from "utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { shapeStandardVersion, StandardVersionShape } from "utils/shape/models/standardVersion";
import { validateFormValues } from "utils/validateFormValues";
import { StandardFormProps, StandardUpsertProps } from "../types";

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
        listFor: {
            __typename: "StandardVersion" as const,
            id: DUMMY_ID,
        },
    },
    versionLabel: "1.0.0",
    ...existing,
    root: {
        __typename: "Standard" as const,
        id: DUMMY_ID,
        isInternal: false,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
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

const transformStandardVersionValues = (values: StandardVersionShape, existing: StandardVersionShape, isCreate: boolean) =>
    isCreate ? shapeStandardVersion.create(values) : shapeStandardVersion.update(existing, values);

const StandardForm = ({
    disabled,
    dirty,
    display,
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
}: StandardFormProps) => {
    const session = useContext(SessionContext);
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

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<StandardVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "StandardVersion",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<StandardVersion, StandardVersionCreateInput, StandardVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostStandardVersion,
        endpointUpdate: endpointPutStandardVersion,
    });
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "StandardVersion" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<StandardVersionCreateInput | StandardVersionUpdateInput, StandardVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformStandardVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="standard-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateStandard" : "UpdateStandard")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
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

export const StandardUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: StandardUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<StandardVersion, StandardVersionShape>({
        ...endpointGetStandardVersion,
        isCreate,
        objectType: "StandardVersion",
        overrideObject,
        transform: (existing) => standardInitialValues(session, existing),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformStandardVersionValues, standardVersionValidation)}
        >
            {(formik) => <StandardForm
                disabled={!(isCreate || canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                versions={existing?.root?.versions?.map(v => v.versionLabel) ?? []}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
