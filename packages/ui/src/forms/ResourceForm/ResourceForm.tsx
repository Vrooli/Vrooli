import { Stack } from "@mui/material";
import { ResourceUsedFor } from "@shared/consts";
import { CommonKey } from "@shared/translations";
import { userTranslationValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { LinkInput } from "components/inputs/LinkInput/LinkInput";
import { Selector } from "components/inputs/Selector/Selector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ResourceFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getResourceIcon } from "utils/display/getResourceIcon";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";

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
        fields: ['bio'],
        validationSchema: userTranslationValidation.update({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    paddingBottom: '64px',
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
                        label={t('Type')}
                    />
                    {/* Enter name */}
                    <TranslatedTextField
                        fullWidth
                        label={t('NameOptional')}
                        language={language}
                        name="name"
                    />
                    {/* Enter description */}
                    <TranslatedTextField
                        fullWidth
                        label={t('DescriptionOptional')}
                        language={language}
                        multiline
                        maxRows={8}
                        name="description"
                    />
                </Stack>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={{
                    ...props.errors,
                    ...translationErrors,
                }}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    )
})