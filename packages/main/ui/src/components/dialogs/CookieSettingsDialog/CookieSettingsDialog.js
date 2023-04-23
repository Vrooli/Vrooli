import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Button, Grid, Stack, Typography } from "@mui/material";
import { useFormik } from "formik";
import { useTranslation } from "react-i18next";
import { setCookiePreferences } from "../../../utils/cookies";
import { HelpButton } from "../../buttons/HelpButton/HelpButton";
import { ToggleSwitch } from "../../inputs/ToggleSwitch/ToggleSwitch";
import { TopBar } from "../../navigation/TopBar/TopBar";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const titleId = "cookie-settings-dialog-title";
const strictlyNecessaryUses = ["Authentication"];
const functionalUses = ["Theme", "FontSize"];
export const CookieSettingsDialog = ({ handleClose, isOpen, }) => {
    const { t } = useTranslation();
    const setPreferences = (preferences) => {
        setCookiePreferences(preferences);
        handleClose(preferences);
    };
    const onCancel = () => { handleClose(); };
    const formik = useFormik({
        initialValues: {
            strictlyNecessary: true,
            performance: false,
            functional: true,
            targeting: false,
        },
        onSubmit: (values) => {
            setPreferences(values);
        },
    });
    const handleAcceptAllCookies = () => {
        const preferences = {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        };
        setPreferences(preferences);
    };
    return (_jsxs(LargeDialog, { id: "cookie-settings-dialog", isOpen: isOpen, onClose: onCancel, titleId: titleId, zIndex: 30000, children: [_jsx(TopBar, { display: "dialog", onClose: onCancel, titleData: { titleId, titleKey: "CookieSettings" } }), _jsxs("form", { onSubmit: formik.handleSubmit, style: { padding: "16px" }, children: [_jsxs(Stack, { direction: "column", spacing: 2, sx: { marginBottom: 2 }, children: [_jsxs(Stack, { direction: "row", marginRight: "auto", alignItems: "center", children: [_jsx(Typography, { component: "h2", variant: "h5", textAlign: "center", children: t("CookieStrictlyNecessary") }), _jsx(HelpButton, { markdown: t("CookieStrictlyNecessaryDescription") }), _jsx(ToggleSwitch, { checked: formik.values.strictlyNecessary, onChange: formik.handleChange, name: "strictlyNecessary", disabled: true, sx: {
                                            position: "absolute",
                                            right: "16px",
                                        } })] }), _jsxs(Typography, { variant: "body1", children: [t("CurrentUses"), ": ", strictlyNecessaryUses.map((use) => t(use)).join(", ")] })] }), _jsxs(Stack, { direction: "column", spacing: 1, sx: { marginBottom: 2 }, children: [_jsxs(Stack, { direction: "row", marginRight: "auto", alignItems: "center", children: [_jsx(Typography, { component: "h2", variant: "h5", textAlign: "center", children: t("Performance") }), _jsx(HelpButton, { markdown: t("CookiePerformanceDescription") }), _jsx(ToggleSwitch, { checked: formik.values.performance, name: "performance", onChange: formik.handleChange, sx: {
                                            position: "absolute",
                                            right: "16px",
                                        } })] }), _jsxs(Typography, { variant: "body1", children: [t("CurrentUses"), ": ", _jsx("b", { children: t("None") })] })] }), _jsxs(Stack, { direction: "column", spacing: 1, sx: { marginBottom: 2 }, children: [_jsxs(Stack, { direction: "row", marginRight: "auto", alignItems: "center", children: [_jsx(Typography, { component: "h2", variant: "h5", textAlign: "center", children: t("Functional") }), _jsx(HelpButton, { markdown: t("CookieFunctionalDescription") }), _jsx(ToggleSwitch, { checked: formik.values.functional, name: "functional", onChange: formik.handleChange, sx: {
                                            position: "absolute",
                                            right: "16px",
                                        } })] }), _jsxs(Typography, { variant: "body1", children: [t("CurrentUses"), ": ", functionalUses.map((use) => t(use)).join(", ")] })] }), _jsxs(Stack, { direction: "column", spacing: 1, sx: { marginBottom: 4 }, children: [_jsxs(Stack, { direction: "row", marginRight: "auto", alignItems: "center", children: [_jsx(Typography, { component: "h2", variant: "h5", textAlign: "center", children: t("Targeting") }), _jsx(HelpButton, { markdown: t("CookieTargetingDescription") }), _jsx(ToggleSwitch, { checked: formik.values.targeting, name: "targeting", onChange: formik.handleChange, sx: {
                                            position: "absolute",
                                            right: "16px",
                                        } })] }), _jsxs(Typography, { variant: "body1", children: [t("CurrentUses"), ": ", _jsx("b", { children: t("None") })] })] }), _jsxs(Grid, { container: true, spacing: 1, sx: {
                            paddingBottom: "env(safe-area-inset-bottom)",
                        }, children: [_jsx(Grid, { item: true, xs: 4, children: _jsx(Button, { fullWidth: true, type: "submit", children: t("Confirm") }) }), _jsx(Grid, { item: true, xs: 4, children: _jsx(Button, { fullWidth: true, onClick: handleAcceptAllCookies, children: t("AcceptAll") }) }), _jsx(Grid, { item: true, xs: 4, children: _jsx(Button, { fullWidth: true, variant: "text", onClick: onCancel, children: t("Cancel") }) })] })] })] }));
};
//# sourceMappingURL=CookieSettingsDialog.js.map