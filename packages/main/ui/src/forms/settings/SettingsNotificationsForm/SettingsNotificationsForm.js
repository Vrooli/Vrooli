import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { EmailIcon, PhoneIcon } from "@local/icons";
import { Stack } from "@mui/material";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { ListContainer } from "../../../components/containers/ListContainer/ListContainer";
import { IntegerInput } from "../../../components/inputs/IntegerInput/IntegerInput";
import { PushList } from "../../../components/lists/devices";
import { SettingsToggleListItem } from "../../../components/lists/SettingsToggleListItem/SettingsToggleListItem";
import { Subheader } from "../../../components/text/Subheader/Subheader";
import { BaseForm } from "../../BaseForm/BaseForm";
export const SettingsNotificationForm = ({ display, dirty, isLoading, onCancel, values, ...props }) => {
    const { t } = useTranslation();
    return (_jsxs(BaseForm, { dirty: dirty, isLoading: isLoading, style: {
            width: { xs: "100%", md: "min(100%, 700px)" },
            margin: "auto",
            display: "block",
        }, children: [_jsxs(Stack, { direction: "column", spacing: 4, children: [_jsx(ListContainer, { children: _jsx(SettingsToggleListItem, { title: t("Notification", { count: 2 }), description: t("PushNotificationToggleDescription"), name: "enabled" }) }), _jsx(IntegerInput, { disabled: !values.enabled, label: t("DailyLimit"), min: 0, name: "dailyLimit" }), _jsx(ListContainer, { children: _jsx(SettingsToggleListItem, { title: t("PushNotification", { count: 2 }), description: t("PushNotificationToggleDescription"), disabled: !values.enabled, name: "toPush" }) }), _jsx(Subheader, { Icon: PhoneIcon, title: t("Device", { count: 2 }) }), _jsx(PushList, { handleUpdate: () => { }, list: [] }), _jsx(ListContainer, { children: _jsx(SettingsToggleListItem, { title: t("EmailNotification", { count: 2 }), description: t("EmailNotificationToggleDescription"), disabled: !values.enabled, name: "toEmails" }) }), _jsx(Subheader, { Icon: EmailIcon, title: t("Email", { count: 2 }) })] }), _jsx(GridSubmitButtons, { display: display, errors: props.errors, isCreate: false, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
};
//# sourceMappingURL=SettingsNotificationsForm.js.map