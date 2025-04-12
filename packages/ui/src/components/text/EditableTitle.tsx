import { useField, useFormikContext } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getTranslationData } from "../../utils/display/translationTools.js";
import { LargeDialog } from "../dialogs/LargeDialog/LargeDialog.js";
import { TopBar } from "../navigation/TopBar.js";
import { Title } from "./Title.js";
import { TitleProps } from "./types.js";

export interface EditableTitleProps extends TitleProps {
    DialogContentForm: (formikContext: ReturnType<typeof useFormikContext>) => React.ReactNode;
    handleDelete?: () => unknown;
    isDeletable?: boolean;
    isEditable?: boolean;
    isTitleTranslated?: boolean;
    isSubtitleTranslated?: boolean;
    language?: string;
    onClose?: () => unknown;
    onSubmit?: () => unknown;
    subtitleField?: string;
    subtitleLabel?: string;
    titleField: string;
    titleLabel?: string;
}

export function EditableTitle({
    DialogContentForm,
    handleDelete,
    isDeletable = false,
    isEditable = true,
    isTitleTranslated = true,
    isSubtitleTranslated = true,
    language,
    onClose,
    onSubmit,
    subtitleField,
    subtitleLabel,
    titleField,
    titleLabel,
    ...titleProps
}: EditableTitleProps) {
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

    const handleOpen = useCallback(() => { setIsDialogOpen(true); }, []);
    const handleClose = useCallback(() => {
        setIsDialogOpen(false);
        onClose && onClose();
    }, [onClose]);

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
                iconInfo: { name: "Edit", type: "Common" } as const,
                label: t("Edit"),
                onClick: handleOpen,
            });
        }
        if (isDeletable) {
            options.push({
                iconInfo: { name: "Delete", type: "Common" } as const,
                label: t("Delete"),
                onClick: () => {
                    if (typeof handleDelete === "function") handleDelete();
                    else console.error("handleDelete is not a function");
                },
            });
        }
        return options;
    }, [isEditable, isDeletable, t, handleOpen, handleDelete]);

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
                onClose={handleClose}
            >
                <TopBar
                    display="dialog"
                    onClose={handleClose}
                />
                {DialogContentForm(formik)}
            </LargeDialog>
        </>
    );
}
