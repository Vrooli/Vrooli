import { Box, Button, Stack, useTheme } from "@mui/material";
import { ProfileUpdateInput, User } from "@shared/consts";
import { SearchIcon } from "@shared/icons";
import { userValidation } from "@shared/validation";
import { Formik } from "formik";
import { SettingsDisplayForm } from "../../../forms/settings";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { mutationWrapper, useCustomMutation } from "../../../api";
import { userProfileUpdate } from "../../../api/generated/endpoints/user_profileUpdate";
import { SettingsList } from "../../../components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "../../../components/navigation/SettingsTopBar/SettingsTopBar";
import { getSiteLanguage } from "../../../utils/authentication/session";
import { useProfileQuery } from "../../../utils/hooks/useProfileQuery";
import { PubSub } from "../../../utils/pubsub";
import { clearSearchHistory } from "../../../utils/search/clearSearchHistory";
import { SessionContext } from "../../../utils/SessionContext";
import { SettingsDisplayViewProps } from "../types";

export const SettingsDisplayView = ({
    display = "page",
}: SettingsDisplayViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [mutation, { loading: isUpdating }] = useCustomMutation<User, ProfileUpdateInput>(userProfileUpdate);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: "Display",
                    helpKey: "DisplaySettingsDescription",
                }}
            />
            <Stack direction="row">
                <SettingsList />
                <Stack direction="column" sx={{
                    margin: "auto",
                    display: "block",
                }}>
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            theme: palette.mode === "dark" ? "dark" : "light",
                        } as ProfileUpdateInput}
                        onSubmit={(values, helpers) => {
                            if (!profile) {
                                PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                                return;
                            }
                            mutationWrapper<User, ProfileUpdateInput>({
                                mutation,
                                input: {
                                    ...values,
                                    languages: [getSiteLanguage(session)],
                                },
                                onSuccess: (data) => { onProfileUpdate(data); },
                                onError: () => { helpers.setSubmitting(false); },
                            });
                        }}
                        validationSchema={userValidation.update({})}
                    >
                        {(formik) => <SettingsDisplayForm
                            display={display}
                            isLoading={isProfileLoading || isUpdating}
                            onCancel={formik.resetForm}
                            {...formik}
                        />}
                    </Formik>
                    <Box sx={{ marginTop: 5, display: "flex", justifyContent: "center", alignItems: "center" }}>
                        <Button id="clear-search-history-button" color="secondary" startIcon={<SearchIcon />} onClick={() => { session && clearSearchHistory(session); }} sx={{
                            marginLeft: "auto",
                            marginRight: "auto",
                        }}>{t("ClearSearchHistory")}</Button>
                    </Box>
                </Stack>
            </Stack>
        </>
    );
};
