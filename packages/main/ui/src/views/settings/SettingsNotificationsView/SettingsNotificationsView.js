import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useQuery } from "@apollo/client";
import { Stack } from "@mui/material";
import { Formik } from "formik";
import { mutationWrapper } from "../../../api";
import { notificationSettings } from "../../../api/generated/endpoints/notification_settings";
import { notificationSettingsUpdate } from "../../../api/generated/endpoints/notification_settingsUpdate";
import { useCustomMutation } from "../../../api/hooks";
import { SettingsList } from "../../../components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "../../../components/navigation/SettingsTopBar/SettingsTopBar";
import { SettingsNotificationForm } from "../../../forms/settings/SettingsNotificationsForm/SettingsNotificationsForm";
import { useDisplayApolloError } from "../../../utils/hooks/useDisplayApolloError";
export const SettingsNotificationsView = ({ display = "page", }) => {
    const { data, refetch, loading: isLoading, error } = useQuery(notificationSettings, { errorPolicy: "all" });
    useDisplayApolloError(error);
    const [mutation, { loading: isUpdating }] = useCustomMutation(notificationSettingsUpdate);
    return (_jsxs(_Fragment, { children: [_jsx(SettingsTopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Notification",
                    titleVariables: { count: 2 },
                } }), _jsxs(Stack, { direction: "row", children: [_jsx(SettingsList, {}), _jsx(Formik, { enableReinitialize: true, initialValues: {
                            includedEmails: [],
                            includedSms: [],
                            includedPush: [],
                            toEmails: false,
                            toSms: false,
                            toPush: false,
                            dailyLimit: 0,
                            enabled: true,
                            categories: [],
                        }, onSubmit: (values, helpers) => mutationWrapper({
                            mutation,
                            input: values,
                            onError: () => { helpers.setSubmitting(false); },
                        }), children: (formik) => _jsx(SettingsNotificationForm, { display: display, isLoading: isLoading || isUpdating, onCancel: formik.resetForm, ...formik }) })] })] }));
};
//# sourceMappingURL=SettingsNotificationsView.js.map