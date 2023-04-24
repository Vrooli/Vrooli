import { Resource, ResourceUsedFor, Session } from "@local/consts";
import { CommonKey } from "@local/translations";
import { orDefault } from "@local/utils";
import { DUMMY_ID } from "@local/uuid";
import { resourceValidation, userTranslationValidation } from "@local/validation";
import { Stack } from "@mui/material";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { LinkInput } from "../../components/inputs/LinkInput/LinkInput";
import { Selector } from "../../components/inputs/Selector/Selector";
import { TranslatedTextField } from "../../components/inputs/TranslatedTextField/TranslatedTextField";
import { getResourceIcon } from "../../utils/display/getResourceIcon";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { ResourceShape, shapeResource } from "../../utils/shape/models/resource";
import { BaseForm } from "../BaseForm/BaseForm";
import { ResourceFormProps } from "../types";

export const resourceInitialValues = (
    session: Session | undefined,
    listId: string | undefined,
    existing?: Resource | null | undefined,
): ResourceShape => ({
    __typename: "Resource" as const,
    id: DUMMY_ID,
    index: 0,
    link: "",
    list: {
        __typename: "ResourceList" as const,
        id: listId ?? DUMMY_ID,
    },
    usedFor: ResourceUsedFor.Context,
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "ResourceTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        name: "",
    }]),
});

export function transformResourceValues(values: ResourceShape, existing?: ResourceShape) {
    return existing === undefined
        ? shapeResource.create(values)
        : shapeResource.update(existing, values);
}

export const validateResourceValues = async (values: ResourceShape, existing?: ResourceShape) => {
    const transformedValues = transformResourceValues(values, existing);
    const validationSchema = resourceValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const ResourceForm = forwardRef<any, ResourceFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {
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
        fields: ["bio"],
        validationSchema: userTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(500px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }}
            >
                <Stack direction="column" spacing={2} padding={2}>
                    {/* Language select */}
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                        zIndex={zIndex + 1}
                    />
                    {/* Enter link or search for object */}
                    <LinkInput zIndex={zIndex} />
                    {/* Select resource type */}
                    <Selector
                        name="usedFor"
                        options={Object.keys(ResourceUsedFor)}
                        getOptionIcon={(i) => getResourceIcon(i as ResourceUsedFor)}
                        getOptionLabel={(l) => t(l as CommonKey, { count: 2 })}
                        fullWidth
                        label={t("Type")}
                    />
                    {/* Enter name */}
                    <TranslatedTextField
                        fullWidth
                        label={t("NameOptional")}
                        language={language}
                        name="name"
                    />
                    {/* Enter description */}
                    <TranslatedTextField
                        fullWidth
                        label={t("DescriptionOptional")}
                        language={language}
                        multiline
                        maxRows={8}
                        name="description"
                    />
                </Stack>
            </BaseForm>
            <GridSubmitButtons
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