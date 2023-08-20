import { CommonKey, DUMMY_ID, orDefault, Resource, ResourceListFor, ResourceUsedFor, resourceValidation, Session, userTranslationValidation } from "@local/shared";
import { Stack } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { LinkInput } from "components/inputs/LinkInput/LinkInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ResourceFormProps } from "forms/types";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { forwardRef, useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getResourceIcon } from "utils/display/getResourceIcon";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ResourceShape, shapeResource } from "utils/shape/models/resource";

/** New resources must include a list ID and an index */
export type NewResourceShape = Partial<Omit<Resource, "list">> & {
    index: number,
    list: Partial<Resource["list"]> & ({ id: string } | { listFor: ResourceListFor | `${ResourceListFor}`, listForId: string })
};

export const resourceInitialValues = (
    session: Session | undefined,
    existing: NewResourceShape,
): ResourceShape => ({
    __typename: "Resource" as const,
    id: DUMMY_ID,
    link: "",
    usedFor: ResourceUsedFor.Context,
    ...existing,
    list: {
        __typename: "ResourceList" as const,
        ...existing.list,
        id: existing.list.id ?? DUMMY_ID,
    },
    translations: orDefault(existing.translations, [{
        __typename: "ResourceTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        name: "",
    }]),
});

export const transformResourceValues = (values: ResourceShape, existing: ResourceShape, isCreate: boolean) =>
    isCreate ? shapeResource.create(values) : shapeResource.update(existing, values);

export const validateResourceValues = async (values: ResourceShape, existing: ResourceShape, isCreate: boolean) => {
    const transformedValues = transformResourceValues(values, existing, isCreate);
    const validationSchema = resourceValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const ResourceForm = forwardRef<BaseFormRef | undefined, ResourceFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    console.log("resource form", isCreate, values);

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

    const [field, , helpers] = useField("translations");

    const foundLinkData = useCallback(({ title, subtitle }: { title: string, subtitle: string }) => {
        // If the user has not entered a name or description, set them to the found values
        if (title.length === 0 || subtitle.length === 0) return;
        const currName = field.value.find((t) => t.language === language)?.name;
        const currDescription = field.value.find((t) => t.language === language)?.description;
        if (currName.length === 0) helpers.setValue(field.value.map((t) => t.language === language ? { ...t, name: title } : t));
        if (currDescription.length === 0) helpers.setValue(field.value.map((t) => t.language === language ? { ...t, description: subtitle } : t));
    }, [field, helpers, language]);

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={500}
                ref={ref}
            >
                <Stack direction="column" spacing={2} padding={2}>
                    {/* Language select */}
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                    />
                    {/* Enter link or search for object */}
                    <LinkInput onObjectData={foundLinkData} />
                    <Selector
                        name="usedFor"
                        options={Object.keys(ResourceUsedFor)}
                        getOptionIcon={(i) => getResourceIcon(i as ResourceUsedFor)}
                        getOptionLabel={(l) => t(l as CommonKey, { count: 2 })}
                        fullWidth
                        label={t("Type")}
                    />
                    <TranslatedTextField
                        fullWidth
                        label={t("NameOptional")}
                        language={language}
                        name="name"
                    />
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
