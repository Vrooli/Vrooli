import { endpointsUser, LINKS, ProfileUpdateInput, profileValidation, User } from "@local/shared";
import { Box, Button, Divider, Grid, Link, styled, Typography, useTheme } from "@mui/material";
import { Formik, FormikHelpers } from "formik";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../components/buttons/BottomActionsButtons.js";
import { FocusModeSelector } from "../../components/inputs/FocusModeSelector/FocusModeSelector.js";
import { LanguageSelector } from "../../components/inputs/LanguageSelector/LanguageSelector.js";
import { LeftHandedCheckbox } from "../../components/inputs/LeftHandedCheckbox/LeftHandedCheckbox.js";
import { TextSizeButtons } from "../../components/inputs/TextSizeButtons/TextSizeButtons.js";
import { ThemeSwitch } from "../../components/inputs/ThemeSwitch/ThemeSwitch.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { SettingsContent, SettingsTopBar } from "../../components/navigation/SettingsTopBar.js";
import { Title } from "../../components/text/Title.js";
import { SessionContext } from "../../contexts.js";
import { InnerForm, OuterForm } from "../../forms/BaseForm/BaseForm.js";
import { useShowBotWarning } from "../../hooks/subscriptions.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { FormSection, ScrollBox } from "../../styles.js";
import { SessionService } from "../../utils/authentication/session.js";
import { PubSub } from "../../utils/pubsub.js";
import { SearchHistory } from "../../utils/search/searchHistory.js";
import { SettingsDisplayFormProps, SettingsDisplayViewProps } from "./types.js";

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
const seeAllLinkStyle = { textAlign: "right" } as const;
const seeAllLinkTextStyle = { marginRight: "12px", marginBottom: "8px" } as const;

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
        SearchHistory.clearSearchHistory(session);
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
                    <Link
                        href={LINKS.SettingsFocusModes}
                        sx={seeAllLinkStyle}
                    >
                        <Typography variant="body2" sx={seeAllLinkTextStyle}>See all focus modes</Typography>
                    </Link>
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
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointsUser.profileUpdate);

    const onSubmit = useCallback(function onSubmitCallback(values: ProfileUpdateInput, helpers: FormikHelpers<ProfileUpdateInput>) {
        if (!profile) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        fetchLazyWrapper<ProfileUpdateInput, User>({
            fetch,
            inputs: {
                ...values,
                languages: [SessionService.getSiteLanguage(session)],
            },
            onSuccess: (data) => { onProfileUpdate(data); },
            onCompleted: () => { helpers.setSubmitting(false); },
        });
    }, [fetch, onProfileUpdate, profile, session]);

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
                        onSubmit={onSubmit}
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
