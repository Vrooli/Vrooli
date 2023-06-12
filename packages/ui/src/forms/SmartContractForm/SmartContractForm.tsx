import { DUMMY_ID, orDefault, Session, SmartContractVersion, smartContractVersionTranslationValidation, smartContractVersionValidation } from "@local/shared";
import { Stack, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { JsonStandardInput, StandardLanguage } from "components/inputs/standards";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { SmartContractFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeSmartContractVersion, SmartContractVersionShape } from "utils/shape/models/smartContractVersion";

export const smartContractInitialValues = (
    session: Session | undefined,
    existing?: SmartContractVersion | undefined,
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
    root: {
        __typename: "SmartContract" as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session)!.id! },
        parent: null,
        tags: [],
    },
    versionLabel: "1.0.0",
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "SmartContractVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        jsonVariable: "",
        name: "",
    }]),
});

export const transformSmartContractValues = (values: SmartContractVersionShape, existing?: SmartContractVersionShape) => {
    return existing === undefined
        ? shapeSmartContractVersion.create(values)
        : shapeSmartContractVersion.update(existing, values);
};

export const validateSmartContractValues = async (values: SmartContractVersionShape, existing?: SmartContractVersionShape) => {
    const transformedValues = transformSmartContractValues(values, existing);
    const validationSchema = smartContractVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const SmartContractForm = forwardRef<any, SmartContractFormProps>(({
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
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(700px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <RelationshipList
                        isEditing={true}
                        objectType={"SmartContract"}
                        zIndex={zIndex}
                    />
                    <Stack direction="column" spacing={2} sx={{
                        borderRadius: 2,
                        background: palette.mode === "dark" ? palette.background.paper : palette.background.default,
                        padding: 2,
                    }}>
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
                            name="description"
                            maxChars={2048}
                            minRows={4}
                            maxRows={8}
                            placeholder={t("Description")}
                            zIndex={zIndex}
                        />
                    </Stack>
                    <JsonStandardInput
                        fieldName="content"
                        isEditing={true}
                        limitTo={[StandardLanguage.Solidity, StandardLanguage.Haskell]}
                        zIndex={zIndex}
                    />
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
                </Stack>
                <GridSubmitButtons
                    display={display}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    );
});
