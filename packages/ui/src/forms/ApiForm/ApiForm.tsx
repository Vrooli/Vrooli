import { useTheme } from "@mui/material";
import { apiVersionTranslationValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RelationshipButtons } from "components/inputs/RelationshipButtons/RelationshipButtons";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ApiFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";

export const ApiForm = forwardRef<any, ApiFormProps>(({
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
    const { palette } = useTheme();
    const { t } = useTranslation();

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
        fields: ['details', 'name', 'summary'],
        validationSchema: apiVersionTranslationValidation.update({}),
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
                <RelationshipButtons
                    isEditing={true}
                    objectType={'Api'}
                    zIndex={zIndex}
                    sx={{ marginBottom: 4 }}
                />
                <LanguageInput
                    currentLanguage={language}
                    handleAdd={handleAddLanguage}
                    handleDelete={handleDeleteLanguage}
                    handleCurrent={setLanguage}
                    languages={languages}
                    zIndex={zIndex + 1}
                />
                {/* TODO */}
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
            </BaseForm>
        </>
    )
})