import { endpointPutProfile, ProfileUpdateInput, SearchIcon, User, userValidation } from "@local/shared";
import { Box, Button, Stack, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Formik } from "formik";
import { SettingsDisplayForm } from "forms/settings";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { getSiteLanguage } from "utils/authentication/session";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useProfileQuery } from "utils/hooks/useProfileQuery";
import { PubSub } from "utils/pubsub";
import { clearSearchHistory } from "utils/search/clearSearchHistory";
import { SessionContext } from "utils/SessionContext";
import { SettingsDisplayViewProps } from "../types";

export const SettingsDisplayView = ({
    display = "page",
    onClose,
    zIndex,
}: SettingsDisplayViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);

    return (
        <>
            <SettingsTopBar
                display={display}
                help={t("DisplaySettingsDescription")}
                onClose={onClose}
                title={t("Display")}
                zIndex={zIndex}
            />
            <Stack direction="row">
                <SettingsList />
                <Box m="auto">
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
                            fetchLazyWrapper<ProfileUpdateInput, User>({
                                fetch,
                                inputs: {
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
                            zIndex={zIndex}
                            {...formik}
                        />}
                    </Formik>
                    <Box sx={{ marginTop: 5, display: "flex", justifyContent: "center", alignItems: "center" }}>
                        <Button
                            id="clear-search-history-button"
                            color="secondary"
                            startIcon={<SearchIcon />}
                            onClick={() => { session && clearSearchHistory(session); }}
                            variant="outlined"
                            sx={{
                                marginLeft: "auto",
                                marginRight: "auto",
                            }}>{t("ClearSearchHistory")}</Button>
                    </Box>
                </Box>
            </Stack>
        </>
    );
};
