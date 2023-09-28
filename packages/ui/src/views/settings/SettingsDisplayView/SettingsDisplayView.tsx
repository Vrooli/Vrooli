import { endpointPutProfile, ProfileUpdateInput, profileValidation, User } from "@local/shared";
import { Box, Button, Stack, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { FocusModeSelector } from "components/inputs/FocusModeSelector/FocusModeSelector";
import { LanguageSelector } from "components/inputs/LanguageSelector/LanguageSelector";
import { LeftHandedCheckbox } from "components/inputs/LeftHandedCheckbox/LeftHandedCheckbox";
import { TextSizeButtons } from "components/inputs/TextSizeButtons/TextSizeButtons";
import { ThemeSwitch } from "components/inputs/ThemeSwitch/ThemeSwitch";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { SearchIcon } from "icons";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { pagePaddingBottom } from "styles";
import { getSiteLanguage } from "utils/authentication/session";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { clearSearchHistory } from "utils/search/clearSearchHistory";
import { SettingsDisplayFormProps, SettingsDisplayViewProps } from "../types";

const SettingsDisplayForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsDisplayFormProps) => {
    const { t } = useTranslation();

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
            >
                <Title
                    help={t("DisplayAccountHelp")}
                    title={t("DisplayAccount")}
                    variant="subheader"
                />
                <Stack direction="column" spacing={2} p={1}>
                    <LanguageSelector />
                    <FocusModeSelector />
                </Stack>
                <Title
                    help={t("DisplayDeviceHelp")}
                    title={t("DisplayDevice")}
                    variant="subheader"
                />
                <Stack direction="column" spacing={2} p={1}>
                    <ThemeSwitch />
                    <TextSizeButtons />
                    <LeftHandedCheckbox />
                </Stack>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors as any}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
};


export const SettingsDisplayView = ({
    isOpen,
    onClose,
}: SettingsDisplayViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);

    return (
        <>
            <SettingsTopBar
                display={display}
                help={t("DisplaySettingsDescription")}
                onClose={onClose}
                title={t("Display")}
            />
            <Stack direction="row" sx={{ paddingBottom: pagePaddingBottom }}>
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
                                onCompleted: () => { helpers.setSubmitting(false); },
                            });
                        }}
                        validationSchema={profileValidation.update({})}
                    >
                        {(formik) => <SettingsDisplayForm
                            display={display}
                            isLoading={isProfileLoading || isUpdating}
                            onCancel={formik.resetForm}
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
