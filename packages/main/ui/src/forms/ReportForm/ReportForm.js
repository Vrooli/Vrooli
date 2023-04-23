import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { DUMMY_ID } from "@local/uuid";
import { TextField } from "@mui/material";
import { Field, useField } from "formik";
import { forwardRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { SelectLanguageMenu } from "../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { Selector } from "../../components/inputs/Selector/Selector";
import { getUserLanguages } from "../../utils/display/translationTools";
import { BaseForm } from "../BaseForm/BaseForm";
var ReportOptions;
(function (ReportOptions) {
    ReportOptions["Inappropriate"] = "Inappropriate";
    ReportOptions["PII"] = "PII";
    ReportOptions["Scam"] = "Scam";
    ReportOptions["Spam"] = "Spam";
    ReportOptions["Other"] = "Other";
})(ReportOptions || (ReportOptions = {}));
const ReportReasons = {
    [ReportOptions.Inappropriate]: "Inappropriate Content",
    [ReportOptions.PII]: "Includes Personally Identifiable Information (PII)",
    [ReportOptions.Scam]: "Scam",
    [ReportOptions.Spam]: "Spam",
    [ReportOptions.Other]: "Other",
};
export const reportInitialValues = (session, createdFor, createdForId, existing) => ({
    __typename: "Report",
    id: DUMMY_ID,
    createdFor: { __typename: createdFor, id: createdForId },
    reason: "",
    otherReason: "",
    details: "",
    language: getUserLanguages(session)[0],
    ...existing,
});
export const ReportForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, onCancel, values, zIndex, ...props }, ref) => {
    const { t } = useTranslation();
    const [languageField, , languageHelpers] = useField("language");
    const setLanguage = useCallback((language) => {
        languageHelpers.setValue(language);
    }, [languageHelpers]);
    const [reasonField] = useField("reason");
    return (_jsx(_Fragment, { children: _jsxs(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                display: "block",
                width: "min(600px, 100vw - 16px)",
                margin: "auto",
                paddingLeft: "env(safe-area-inset-left)",
                paddingRight: "env(safe-area-inset-right)",
                paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
            }, children: [_jsx(SelectLanguageMenu, { currentLanguage: languageField.value, handleCurrent: setLanguage, languages: [languageField.value], zIndex: zIndex }), _jsx(Selector, { name: "reason", disabled: isLoading, options: Object.keys(ReportReasons), getOptionLabel: (r) => ReportReasons[r], fullWidth: true, inputAriaLabel: "select reason", label: t("Reason") }), reasonField.value === ReportOptions.Other && _jsx(Field, { fullWidth: true, name: "otherReason", label: t("ReasonCustom"), helperText: t("ReasonCustomHelp"), as: TextField }), _jsx(Field, { fullWidth: true, multiline: true, rows: 4, name: "details", label: t("DetailsOptional"), helperText: t("ReportDetailsHelp"), as: TextField }), _jsx(GridSubmitButtons, { display: display, errors: props.errors, isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }) }));
});
//# sourceMappingURL=ReportForm.js.map