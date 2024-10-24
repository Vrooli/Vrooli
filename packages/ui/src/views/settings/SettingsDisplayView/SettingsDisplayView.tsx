import { endpointPutProfile, ProfileUpdateInput, profileValidation, User } from "@local/shared";
import { Box, Button, Divider, Grid, styled, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { FocusModeSelector } from "components/inputs/FocusModeSelector/FocusModeSelector";
import { LanguageSelector } from "components/inputs/LanguageSelector/LanguageSelector";
import { LeftHandedCheckbox } from "components/inputs/LeftHandedCheckbox/LeftHandedCheckbox";
import { TextSizeButtons } from "components/inputs/TextSizeButtons/TextSizeButtons";
import { ThemeSwitch } from "components/inputs/ThemeSwitch/ThemeSwitch";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsContent, SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts";
import { Formik } from "formik";
import { InnerForm, OuterForm } from "forms/BaseForm/BaseForm";
import { useShowBotWarning } from "hooks/subscriptions";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormSection, ScrollBox } from "styles";
import { getSiteLanguage } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { clearSearchHistory } from "utils/search/clearSearchHistory";
import { SettingsDisplayFormProps, SettingsDisplayViewProps } from "../types";

const ClearSettingBox = styled(Box)(({ theme }) => ({
    background: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
    borderRadius: theme.spacing(1),
    padding: theme.spacing(1),
}));

function ClearSettingButton({ onClick, title, description }) {
    return (
        <ClearSettingBox>
            <Grid container spacing={2} alignItems="center">
                <Grid item xs>
                    <Typography variant="subtitle1">{title}</Typography>
                    <Typography variant="body2">{description}</Typography>
                </Grid>
                <Grid item>
                    <Button
                        variant="outlined"
                        color="secondary"
                        onClick={onClick}
                    >
                        Clear
                    </Button>
                </Grid>
            </Grid>
        </ClearSettingBox>
    );
}

const dividerStyle = {
    width: "100%",
    paddingTop: 2,
} as const;

function SettingsDisplayForm({
    display,
    isLoading,
    onCancel,
    ...props
}: SettingsDisplayFormProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const botWarningPreferences = useShowBotWarning();

    const handleClearSearchHistory = useCallback(function handleClearSearchHistoryCallback() {
        if (!session) return;
        clearSearchHistory(session);
    }, [session]);

    const handleClearBotWarning = useCallback(function handleClearBotWarningCallback() {
        botWarningPreferences.handleUpdateShowBotWarning(null);
    }, [botWarningPreferences]);

    return (
        <OuterForm display={display}>
            <InnerForm
                display={display}
                isLoading={isLoading}
            >
                <FormSection variant="transparent">
                    <Title
                        help={t("DisplayAccountHelp")}
                        title={t("DisplayAccount")}
                        variant="subheader"
                    />
                    <LanguageSelector />
                    <FocusModeSelector />
                </FormSection>
                <Divider sx={dividerStyle} />
                <FormSection variant="transparent">
                    <Title
                        help={t("DisplayDeviceHelp")}
                        title={t("DisplayDevice")}
                        variant="subheader"
                    />
                    <Box width="fit-content">
                        <ThemeSwitch updateServer={false} />
                    </Box>
                    <TextSizeButtons />
                    <Box width="fit-content">
                        <LeftHandedCheckbox />
                    </Box>
                </FormSection>
                <Divider sx={dividerStyle} />
                <FormSection variant="transparent">
                    <Title
                        help={"Clear different types of display caches to reset display preferences and reduce storage."}
                        title={"Display Cache"}
                        variant="subheader"
                    />
                    <ClearSettingButton
                        onClick={handleClearSearchHistory}
                        title={t("ClearSearchHistory")}
                        description={"This will clear your search history suggestions when typing in search bars."}
                    />
                    <ClearSettingButton
                        onClick={handleClearBotWarning}
                        title={"Reset \"Hide Bot Warning\""}
                        description={"When starting a new chat with a bot, the warning that the chat is with a bot will be displayed."}
                    />
                </FormSection>
            </InnerForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </OuterForm>
    );
}

export function SettingsDisplayView({
    display,
    onClose,
}: SettingsDisplayViewProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);

    return (
        <ScrollBox>
            <SettingsTopBar
                display={display}
                help={t("DisplaySettingsDescription")}
                onClose={onClose}
                title={t("Display")}
            />
            <SettingsContent>
                <SettingsList />
                <Box display="flex" flexDirection="column" p={1} margin="auto" width="100%">
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            theme: palette.mode === "dark" ? "dark" : "light",
                        } as ProfileUpdateInput}
                        onSubmit={(values, helpers) => {
                            if (!profile) {
                                PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
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
                        validationSchema={profileValidation.update({ env: process.env.NODE_ENV })}
                    >
                        {(formik) => <SettingsDisplayForm
                            display={display}
                            isLoading={isProfileLoading || isUpdating}
                            onCancel={formik.resetForm}
                            {...formik}
                        />}
                    </Formik>
                </Box>
            </SettingsContent>
        </ScrollBox>
    );
}
