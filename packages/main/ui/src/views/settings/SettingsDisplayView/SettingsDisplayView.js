import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { SearchIcon } from "@local/icons";
import { userValidation } from "@local/validation";
import { Box, Button, Stack, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { mutationWrapper, useCustomMutation } from "../../../api";
import { userProfileUpdate } from "../../../api/generated/endpoints/user_profileUpdate";
import { SettingsList } from "../../../components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "../../../components/navigation/SettingsTopBar/SettingsTopBar";
import { SettingsDisplayForm } from "../../../forms/settings";
import { getSiteLanguage } from "../../../utils/authentication/session";
import { useProfileQuery } from "../../../utils/hooks/useProfileQuery";
import { PubSub } from "../../../utils/pubsub";
import { clearSearchHistory } from "../../../utils/search/clearSearchHistory";
import { SessionContext } from "../../../utils/SessionContext";
export const SettingsDisplayView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [mutation, { loading: isUpdating }] = useCustomMutation(userProfileUpdate);
    return (_jsxs(_Fragment, { children: [_jsx(SettingsTopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Display",
                    helpKey: "DisplaySettingsDescription",
                } }), _jsxs(Stack, { direction: "row", children: [_jsx(SettingsList, {}), _jsxs(Stack, { direction: "column", sx: {
                            margin: "auto",
                            display: "block",
                        }, children: [_jsx(Formik, { enableReinitialize: true, initialValues: {
                                    theme: palette.mode === "dark" ? "dark" : "light",
                                }, onSubmit: (values, helpers) => {
                                    if (!profile) {
                                        PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                                        return;
                                    }
                                    mutationWrapper({
                                        mutation,
                                        input: {
                                            ...values,
                                            languages: [getSiteLanguage(session)],
                                        },
                                        onSuccess: (data) => { onProfileUpdate(data); },
                                        onError: () => { helpers.setSubmitting(false); },
                                    });
                                }, validationSchema: userValidation.update({}), children: (formik) => _jsx(SettingsDisplayForm, { display: display, isLoading: isProfileLoading || isUpdating, onCancel: formik.resetForm, ...formik }) }), _jsx(Box, { sx: { marginTop: 5, display: "flex", justifyContent: "center", alignItems: "center" }, children: _jsx(Button, { id: "clear-search-history-button", color: "secondary", startIcon: _jsx(SearchIcon, {}), onClick: () => { session && clearSearchHistory(session); }, sx: {
                                        marginLeft: "auto",
                                        marginRight: "auto",
                                    }, children: t("ClearSearchHistory") }) })] })] })] }));
};
//# sourceMappingURL=SettingsDisplayView.js.map