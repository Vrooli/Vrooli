import { DUMMY_ID, orDefault, Session, StandardVersion, standardVersionTranslationValidation, standardVersionValidation } from "@local/shared";
import { useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { StandardInput } from "components/inputs/standards/StandardInput/StandardInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { StandardFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { InputTypeOptions } from "utils/consts";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeStandardVersion, StandardVersionShape } from "utils/shape/models/standardVersion";

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

export const transformStandardValues = (values: StandardVersionShape, existing: StandardVersionShape, isCreate: boolean) =>
    isCreate ? shapeStandardVersion.create(values) : shapeStandardVersion.update(existing, values);

export const validateStandardValues = async (values: StandardVersionShape, existing: StandardVersionShape, isCreate: boolean) => {
    const transformedValues = transformStandardValues(values, existing, isCreate);
    const validationSchema = standardVersionValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const StandardForm = forwardRef<BaseFormRef | undefined, StandardFormProps>(({
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
        fields: ["description"],
        validationSchema: standardVersionTranslationValidation[isCreate ? "create" : "update"]({}),
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
                            name="description"
                            maxChars={2048}
                            minRows={4}
                            maxRows={8}
                            placeholder={t("Description")}
                            zIndex={zIndex}
                        />
                    </FormSection>
                    <StandardInput
                        fieldName="preview"
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
                </FormContainer>
            </BaseForm>
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
        </>
    );
});
