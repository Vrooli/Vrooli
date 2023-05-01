/**
 * Displays all search options for an organization
 */
import { Button, Grid, Stack, Typography } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { ToggleSwitch } from "components/inputs/ToggleSwitch/ToggleSwitch";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from "formik";
import { useTranslation } from "react-i18next";
import { CookiePreferences, setCookiePreferences } from "utils/cookies";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { CookieSettingsDialogProps } from "../types";

const titleId = "cookie-settings-dialog-title";
const strictlyNecessaryUses = ["Authentication"] as const;
const functionalUses = ["Theme", "FontSize"] as const;

export const CookieSettingsDialog = ({
    handleClose,
    isOpen,
}: CookieSettingsDialogProps) => {
    const { t } = useTranslation();

    const setPreferences = (preferences: CookiePreferences) => {
        // Set preference in local storage
        setCookiePreferences(preferences);
        // Close dialog
        handleClose(preferences);
    };
    const onCancel = () => { handleClose(); };

    const formik = useFormik({
        initialValues: {
            strictlyNecessary: true,
            performance: false,
            functional: true,
            targeting: false,
        },
        onSubmit: (values) => {
            setPreferences(values);
        },
    });

    const handleAcceptAllCookies = () => {
        const preferences: CookiePreferences = {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        };
        setPreferences(preferences);
    };

    return (
        <LargeDialog
            id="cookie-settings-dialog"
            isOpen={isOpen}
            onClose={onCancel}
            titleId={titleId}
            zIndex={30000}
        >
            <TopBar
                display="dialog"
                onClose={onCancel}
                titleData={{ titleId, titleKey: "CookieSettings" }}
            />
            <form onSubmit={formik.handleSubmit} style={{ padding: "16px" }}>
                {/* Strictly necessary */}
                <Stack direction="column" spacing={2} sx={{ marginBottom: 2 }}>
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">{t("CookieStrictlyNecessary")}</Typography>
                        <HelpButton markdown={t("CookieStrictlyNecessaryDescription")} />
                        <ToggleSwitch
                            checked={formik.values.strictlyNecessary}
                            onChange={formik.handleChange}
                            name="strictlyNecessary"
                            disabled // Can't turn off
                            sx={{
                                position: "absolute",
                                right: "16px",
                            }}
                        />
                    </Stack>
                    <Typography variant="body1">
                        {t("CurrentUses")}: {strictlyNecessaryUses.map((use) => t(use)).join(", ")}
                    </Typography>
                </Stack>
                {/* Performance */}
                <Stack direction="column" spacing={1} sx={{ marginBottom: 2 }}>
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">{t("Performance")}</Typography>
                        <HelpButton markdown={t("CookiePerformanceDescription")} />
                        <ToggleSwitch
                            checked={formik.values.performance}
                            name="performance"
                            onChange={formik.handleChange}
                            sx={{
                                position: "absolute",
                                right: "16px",
                            }}
                        />
                    </Stack>
                    <Typography variant="body1">
                        {t("CurrentUses")}: <b>{t("None")}</b>
                    </Typography>
                </Stack>
                {/* Functional */}
                <Stack direction="column" spacing={1} sx={{ marginBottom: 2 }}>
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">{t("Functional")}</Typography>
                        <HelpButton markdown={t("CookieFunctionalDescription")} />
                        <ToggleSwitch
                            checked={formik.values.functional}
                            name="functional"
                            onChange={formik.handleChange}
                            sx={{
                                position: "absolute",
                                right: "16px",
                            }}
                        />
                    </Stack>
                    <Typography variant="body1">
                        {t("CurrentUses")}: {functionalUses.map((use) => t(use)).join(", ")}
                    </Typography>
                </Stack>
                {/* Targeting */}
                <Stack direction="column" spacing={1} sx={{ marginBottom: 4 }}>
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">{t("Targeting")}</Typography>
                        <HelpButton markdown={t("CookieTargetingDescription")} />
                        <ToggleSwitch
                            checked={formik.values.targeting}
                            name="targeting"
                            onChange={formik.handleChange}
                            sx={{
                                position: "absolute",
                                right: "16px",
                            }}
                        />
                    </Stack>
                    <Typography variant="body1">
                        {t("CurrentUses")}: <b>{t("None")}</b>
                    </Typography>
                </Stack>
                {/* Search/Cancel buttons */}
                <Grid container spacing={1} sx={{
                    paddingBottom: "env(safe-area-inset-bottom)",
                }}>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            type="submit"
                        >{t("Confirm")}</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            onClick={handleAcceptAllCookies}
                        >{t("AcceptAll")}</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            variant="text"
                            onClick={onCancel}
                        >{t("Cancel")}</Button>
                    </Grid>
                </Grid>
            </form>
        </LargeDialog>
    );
};
