/**
 * Displays all search options for a team
 */
import { Button, Divider, Grid, Stack, Typography } from "@mui/material";
import { useFormik } from "formik";
import { useTranslation } from "react-i18next";
import { Z_INDEX } from "../../../utils/consts.js";
import { CookiePreferences, setCookie } from "../../../utils/localStorage.js";
import { HelpButton } from "../../buttons/HelpButton/HelpButton.js";
import { ToggleSwitch } from "../../inputs/ToggleSwitch/ToggleSwitch.js";
import { TopBar } from "../../navigation/TopBar.js";
import { LargeDialog } from "../LargeDialog/LargeDialog.js";
import { CookieSettingsDialogProps } from "../types.js";

const titleId = "cookie-settings-dialog-title";
const strictlyNecessaryUses = ["Authentication"] as const;
const functionalUses = ["DisplayCustomization", "Caching"] as const;

const largeDialogSxs = {
    paper: { width: "min(100vw - 64px, 600px)" },
    root: { zIndex: Z_INDEX.CookieSettingsDialog },
} as const;

export function CookieSettingsDialog({
    handleClose,
    isOpen,
}: CookieSettingsDialogProps) {
    const { t } = useTranslation();

    function setPreferences(preferences: CookiePreferences) {
        // Set preference in local storage
        setCookie("Preferences", preferences);
        // Close dialog
        handleClose(preferences);
    }
    function onCancel() {
        handleClose();
    }

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

    function handleAcceptAllCookies() {
        const preferences: CookiePreferences = {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        };
        setPreferences(preferences);
    }

    return (
        <LargeDialog
            id="cookie-settings-dialog"
            isOpen={isOpen}
            onClose={onCancel}
            titleId={titleId}
            sxs={largeDialogSxs}
        >
            <TopBar
                display="dialog"
                onClose={onCancel}
                title={t("CookieSettings")}
                titleId={titleId}
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
                <Divider />
                {/* Functional */}
                <Stack direction="column" spacing={1} sx={{ marginTop: 2, marginBottom: 2 }}>
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
                <Divider />
                {/* Performance */}
                <Stack direction="column" spacing={1} sx={{ marginTop: 2, marginBottom: 2 }}>
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
                <Divider />
                {/* Targeting */}
                <Stack direction="column" spacing={1} sx={{ marginTop: 2, marginBottom: 4 }}>
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
                            variant="contained"
                        >{t("Confirm")}</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            onClick={handleAcceptAllCookies}
                            variant="contained"
                        >{t("AcceptAll")}</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            onClick={onCancel}
                            variant="outlined"
                        >{t("Cancel")}</Button>
                    </Grid>
                </Grid>
            </form>
        </LargeDialog>
    );
}
