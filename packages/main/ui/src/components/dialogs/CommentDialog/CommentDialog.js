import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { commentTranslationValidation } from "@local/validation";
import { Box, Typography, useTheme } from "@mui/material";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { BaseForm } from "../../../forms/BaseForm/BaseForm";
import { getDisplay } from "../../../utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools";
import { useTranslatedFields } from "../../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../../utils/SessionContext";
import { GridSubmitButtons } from "../../buttons/GridSubmitButtons/GridSubmitButtons";
import { TranslatedMarkdownInput } from "../../inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TopBar } from "../../navigation/TopBar/TopBar";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const titleId = "comment-dialog-title";
export const CommentDialog = ({ dirty, isCreate, isLoading, isOpen, onCancel, parent, ref, zIndex, ...props }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { language, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["text"],
        validationSchema: commentTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    const { subtitle: parentText } = useMemo(() => getDisplay(parent, [language]), [language, parent]);
    return (_jsxs(LargeDialog, { id: "comment-dialog", isOpen: isOpen, onClose: onCancel, titleId: titleId, zIndex: zIndex, children: [_jsx(TopBar, { display: "dialog", onClose: onCancel, titleData: { titleId, titleKey: isCreate ? "AddComment" : "EditComment" } }), _jsxs(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                    display: "block",
                    paddingBottom: "64px",
                }, children: [_jsx(TranslatedMarkdownInput, { language: language, name: "text", placeholder: t("PleaseBeNice"), minRows: 3, sxs: {
                            bar: {
                                borderRadius: 0,
                                background: palette.primary.main,
                            },
                            textArea: {
                                borderRadius: 0,
                                resize: "none",
                                height: parent ? "70vh" : "100vh",
                                background: palette.background.paper,
                            },
                        } }), parent && (_jsx(Box, { sx: {
                            backgroundColor: palette.background.paper,
                            height: "30vh",
                        }, children: _jsx(Typography, { variant: "body2", children: parentText }) })), _jsx(GridSubmitButtons, { display: "dialog", errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] })] }));
};
//# sourceMappingURL=CommentDialog.js.map