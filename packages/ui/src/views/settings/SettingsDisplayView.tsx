import { Box } from "../../components/layout/Box.js";
import { Button } from "../../components/buttons/Button.js";
import { Grid } from "../../components/layout/Grid.js";
import { Typography } from "../../components/text/Typography.js";
import { styled, useTheme } from "@mui/material";
import { endpointsUser, profileValidation, type ProfileUpdateInput, type User } from "@vrooli/shared";
import { Formik, type FormikHelpers } from "formik";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../components/buttons/BottomActionsButtons.js";
import { LanguageSelector } from "../../components/inputs/LanguageSelector/LanguageSelector.js";
import { LeftHandedCheckbox } from "../../components/inputs/LeftHandedCheckbox/LeftHandedCheckbox.js";
import { TextSizeButtons } from "../../components/inputs/TextSizeButtons/TextSizeButtons.js";
import { ThemeSwitch } from "../../components/inputs/ThemeSwitch/ThemeSwitch.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SettingsContent } from "../../components/navigation/SettingsTopBar.js";
import { Title } from "../../components/text/Title.js";
import { SessionContext } from "../../contexts/session.js";
import { InnerForm, OuterForm } from "../../forms/BaseForm/BaseForm.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { FormSection, ScrollBox } from "../../styles.js";
import { SessionService } from "../../utils/authentication/session.js";
import { PubSub } from "../../utils/pubsub.js";
import { SearchHistory } from "../../utils/search/searchHistory.js";
import { type SettingsDisplayFormProps, type SettingsDisplayViewProps } from "./types.js";

// Custom styled component replacement
const ClearSettingBox = ({ children, ...props }) => {
    const theme = useTheme();
    return (
        <Box
            variant="paper"
            padding="sm"
            borderRadius="md"
            style={{
                color: theme.palette.text.primary,
            }}
            {...props}
        >
            {children}
        </Box>
    );
};

function ClearSettingButton({ onClick, title, description }) {
    return (
        <ClearSettingBox>
            <div className="tw-flex tw-items-center tw-gap-4">
                <div className="tw-flex-1">
                    <Typography variant="subtitle1">{title}</Typography>
                    <Typography variant="body2">{description}</Typography>
                </div>
                <div>
                    <Button
                        variant="outline"
                        onClick={onClick}
                    >
                        Clear
                    </Button>
                </div>
            </div>
        </ClearSettingBox>
    );
}

function SettingsDisplayForm({
    display,
    isLoading,
    onCancel,
    ...props
}: SettingsDisplayFormProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const handleClearSearchHistory = useCallback(function handleClearSearchHistoryCallback() {
        if (!session) return;
        SearchHistory.clearSearchHistory(session);
    }, [session]);

    return (
        <OuterForm display={display}>
            <InnerForm
                display={display}
                isLoading={isLoading}
            >
                <Box display="flex" flexDirection="column" gap={2}>
                    <FormSection variant="card">
                        <Title
                            title={t("DisplayAccount")}
                            variant="subheader"
                        />
                        <LanguageSelector />
                    </FormSection>
                    <FormSection variant="card">
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
                    <FormSection variant="card">
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
                    </FormSection>
                </Box>
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
}: SettingsDisplayViewProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointsUser.profileUpdate);

    const onSubmit = useCallback(function onSubmitCallback(values: ProfileUpdateInput, helpers: FormikHelpers<ProfileUpdateInput>) {
        if (!profile) {
            PubSub.get().publish("snack", { message: t("CouldNotReadProfile", { ns: "error" }), severity: "Error" });
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
    }, [fetch, onProfileUpdate, profile, session, t]);

    return (
        <ScrollBox>
            <Navbar
                help={t("DisplaySettingsDescription")}
                title={t("Display")}
            />
            <SettingsContent>
                <SettingsList />
                <Box className="tw-flex tw-flex-col tw-p-1 tw-mx-auto tw-w-full">
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
