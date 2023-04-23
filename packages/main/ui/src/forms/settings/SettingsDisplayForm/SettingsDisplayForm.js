import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Stack } from "@mui/material";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { FocusModeSelector } from "../../../components/inputs/FocusModeSelector/FocusModeSelector";
import { LanguageSelector } from "../../../components/inputs/LanguageSelector/LanguageSelector";
import { LeftHandedCheckbox } from "../../../components/inputs/LeftHandedCheckbox/LeftHandedCheckbox";
import { TextSizeButtons } from "../../../components/inputs/TextSizeButtons/TextSizeButtons";
import { ThemeSwitch } from "../../../components/inputs/ThemeSwitch/ThemeSwitch";
import { Subheader } from "../../../components/text/Subheader/Subheader";
import { BaseForm } from "../../BaseForm/BaseForm";
export const SettingsDisplayForm = ({ display, dirty, isLoading, onCancel, values, ...props }) => {
    const { t } = useTranslation();
    return (_jsxs(BaseForm, { dirty: dirty, isLoading: isLoading, style: {
            width: { xs: "100%", md: "min(100%, 700px)" },
            margin: "auto",
            display: "block",
        }, children: [_jsx(Subheader, { help: t("DisplayAccountHelp"), title: t("DisplayAccount") }), _jsxs(Stack, { direction: "column", spacing: 2, p: 1, children: [_jsx(LanguageSelector, {}), _jsx(FocusModeSelector, {})] }), _jsx(Subheader, { help: t("DisplayDeviceHelp"), title: t("DisplayDevice") }), _jsxs(Stack, { direction: "column", spacing: 2, p: 1, children: [_jsx(ThemeSwitch, {}), _jsx(TextSizeButtons, {}), _jsx(LeftHandedCheckbox, {})] }), _jsx(GridSubmitButtons, { display: display, errors: props.errors, isCreate: false, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
};
//# sourceMappingURL=SettingsDisplayForm.js.map