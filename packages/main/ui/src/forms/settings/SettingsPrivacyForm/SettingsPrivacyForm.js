import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { useTranslation } from "react-i18next";
import { ListContainer } from "../../../components/containers/ListContainer/ListContainer";
import { SettingsToggleListItem } from "../../../components/lists/SettingsToggleListItem/SettingsToggleListItem";
import { BaseForm } from "../../BaseForm/BaseForm";
export const SettingsPrivacyForm = ({ display, dirty, isLoading, onCancel, values, ...props }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [isPrivateField] = useField("isPrivate");
    return (_jsxs(BaseForm, { dirty: dirty, isLoading: isLoading, style: {
            width: { xs: "100%", md: "min(100%, 700px)" },
            margin: "auto",
            display: "block",
        }, children: [_jsx(ListContainer, { children: _jsx(SettingsToggleListItem, { title: t("PrivateAccount"), description: t("PushNotificationToggleDescription"), name: "isPrivate" }) }), isPrivateField.value && (_jsx(Typography, { variant: "body2", sx: {
                    marginTop: 4,
                    marginBottom: 1,
                    color: palette.warning.main,
                }, children: "All of your content is private. Turn off private mode to change specific settings." })), _jsxs(ListContainer, { children: [_jsx(SettingsToggleListItem, { disabled: isPrivateField.value, title: t("PrivateApis"), name: "isPrivateApis" }), _jsx(SettingsToggleListItem, { disabled: isPrivateField.value, title: t("PrivateBookmarks"), name: "isPrivateBookmarks" }), _jsx(SettingsToggleListItem, { disabled: isPrivateField.value, title: t("PrivateProjects"), name: "isPrivateProjects" }), _jsx(SettingsToggleListItem, { disabled: isPrivateField.value, title: t("PrivateRoutines"), name: "isPrivateRoutines" }), _jsx(SettingsToggleListItem, { disabled: isPrivateField.value, title: t("PrivateSmartContracts"), name: "isPrivateSmartContracts" }), _jsx(SettingsToggleListItem, { disabled: isPrivateField.value, title: t("PrivateStandards"), name: "isPrivateStandards" })] })] }));
};
//# sourceMappingURL=SettingsPrivacyForm.js.map