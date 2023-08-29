import { TextField } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RichInput } from "components/inputs/RichInput/RichInput";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { Field, useField, useFormikContext } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { DeleteIcon, EditIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { getTranslationData } from "utils/display/translationTools";
import { Title } from "../Title/Title";
import { TitleProps } from "../types";

export interface EditableTitleProps extends TitleProps {
    handleDelete?: () => unknown;
    isDeletable?: boolean;
    isEditable?: boolean;
    isTitleTranslated?: boolean;
    isSubtitleTranslated?: boolean;
    language?: string;
    subtitleField?: string;
    subtitleLabel?: string;
    titleField: string;
    titleLabel?: string;
    validationEnabled?: boolean;
}

export const EditableTitle = ({
    handleDelete,
    isDeletable = false,
    isEditable = true,
    isTitleTranslated = true,
    isSubtitleTranslated = true,
    language,
    subtitleField,
    subtitleLabel,
    titleField,
    titleLabel,
    validationEnabled = true,
    ...titleProps
}: EditableTitleProps) => {
    const { t } = useTranslation();
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const formik = useFormikContext();
    const [translationsField, translationsMeta] = useField("translations");

    const getFieldValue = useCallback((field: string, isTranslated: boolean) => {
        if (isTranslated && language) {
            const translation = getTranslationData(translationsField, translationsMeta, language).value;
            return translation?.[field] || "";
        }
        return (formik.values as object)?.[field];
    }, [formik.values, language, translationsField, translationsMeta]);

    const setFieldValue = useCallback((field: string, value: string, isTranslated: boolean) => {
        if (isTranslated && language) {
            const translation = getTranslationData(translationsField, translationsMeta, language).value;
            const newTranslation = { ...translation, [field]: value };
            const otherTranslations = (translationsField.value || []).filter((t) => t.language !== language);
            formik.setFieldValue("translations", [...otherTranslations, newTranslation]);
        } else {
            formik.setFieldValue(field, value);
        }
    }, [formik, language, translationsField, translationsMeta]);

    const [initialDialogValues, setInitialDialogValues] = useState<{
        title: string;
        subtitle: string;
    }>({ title: "", subtitle: "" });

    const handleOpenDialog = () => {
        setInitialDialogValues({
            title: getFieldValue(titleField, isTitleTranslated),
            subtitle: subtitleField ? getFieldValue(subtitleField, isSubtitleTranslated) : "",
        });
        setIsDialogOpen(true);
    };

    const handleSubmit = () => {
        setIsDialogOpen(false);
    };

    const mergeTranslations = (field: string, value: string, currentTranslations) => {
        const existingTranslation = currentTranslations.find(t => t.language === language);
        if (existingTranslation) {
            existingTranslation[field] = value;
        } else {
            currentTranslations.push({ language, [field]: value });
        }
        return currentTranslations;
    };
    const handleCancel = () => {
        // Get the current translations array
        const currentTranslations = [...(translationsField.value || [])];
        // Merge title translation
        if (isTitleTranslated) {
            mergeTranslations(titleField, initialDialogValues.title, currentTranslations);
        } else {
            formik.setFieldValue(titleField, initialDialogValues.title);
        }
        // Merge subtitle translation, if subtitleField exists
        if (subtitleField) {
            if (isSubtitleTranslated) {
                mergeTranslations(subtitleField, initialDialogValues.subtitle, currentTranslations);
            } else {
                formik.setFieldValue(subtitleField, initialDialogValues.subtitle);
            }
        }
        // If either title or subtitle or both are translated, update the translations array
        if (isTitleTranslated || (subtitleField && isSubtitleTranslated)) {
            formik.setFieldValue("translations", currentTranslations);
        }
        setIsDialogOpen(false);
    };

    const { title, subtitle } = useMemo(() => {
        return {
            title: getFieldValue(titleField, isTitleTranslated),
            subtitle: subtitleField ? getFieldValue(subtitleField, isSubtitleTranslated) : "",
        };
    }, [getFieldValue, titleField, isTitleTranslated, subtitleField, isSubtitleTranslated]);

    const titleOptions = useMemo(() => {
        const options: TitleProps["options"] = [];
        if (isEditable) {
            options.push({
                Icon: EditIcon,
                label: t("Edit"),
                onClick: handleOpenDialog,
            });
        }
        if (isDeletable) {
            options.push({
                Icon: DeleteIcon,
                label: t("Delete"),
                onClick: () => {
                    if (typeof handleDelete === "function") handleDelete();
                    else console.error("handleDelete is not a function");
                },
            });
        }
        return options;
    }, [isEditable, isDeletable, t, handleOpenDialog, handleDelete]);

    return (
        <>
            <Title
                title={title}
                help={subtitle}
                {...titleProps}
                options={titleOptions}
            />
            <LargeDialog
                id="editable-title-dialog"
                isOpen={isDialogOpen}
                onClose={handleCancel}
            >
                <BaseForm
                    dirty={formik.dirty}
                    display="dialog"
                    style={{
                        width: "min(500px, 100vw)",
                        paddingBottom: "16px",
                    }}
                >
                    <FormContainer>
                        {(language && isTitleTranslated) ? <TranslatedTextField
                            language={language}
                            name={titleField}
                            label={titleLabel || t("Name")}
                        /> : <Field
                            fullWidth
                            name={titleField}
                            label={titleLabel || t("Name")}
                            as={TextField}
                        />}
                        {(language && isSubtitleTranslated && subtitleField) ? <TranslatedRichInput
                            language={language}
                            name={subtitleField}
                            maxChars={1024}
                            minRows={4}
                            maxRows={8}
                            placeholder={subtitleLabel || t("Description")}
                        /> : subtitleField ? <RichInput
                            name={subtitleField}
                            maxChars={1024}
                            minRows={4}
                            maxRows={8}
                            placeholder={subtitleLabel || t("Description")}
                        /> : null}
                    </FormContainer>
                </BaseForm>
                <BottomActionsButtons
                    display="dialog"
                    errors={validationEnabled ? formik.errors : {}}
                    isCreate={false}
                    onCancel={handleCancel}
                    onSubmit={handleSubmit}
                />
            </LargeDialog>
        </>
    );
};
