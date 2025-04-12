import { Box, Button, Grid, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { noSelect } from "../../../styles.js";
import { CookiePreferences, setCookie } from "../../../utils/localStorage.js";
import { CookieSettingsDialog } from "../../dialogs/CookieSettingsDialog/CookieSettingsDialog.js";
import { CookiesSnackProps } from "../types.js";

/**
 * "This site uses cookies" consent dialog
 */
export function CookiesSnack({
    handleClose,
}: CookiesSnackProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [isCustomizeOpen, setIsCustomizeOpen] = useState(false);

    function handleAcceptAllCookies() {
        const preferences: CookiePreferences = {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        };
        // Set preference in local storage
        setCookie("Preferences", preferences);
        // Close dialog
        setIsCustomizeOpen(false);
        handleClose();
    }

    function handleCustomizeCookies(preferences?: CookiePreferences) {
        setIsCustomizeOpen(false);
        if (preferences) {
            setCookie("Preferences", preferences);
            handleClose();
        }
    }

    return (
        <>
            {/* Customize preferences dialog */}
            <CookieSettingsDialog handleClose={handleCustomizeCookies} isOpen={isCustomizeOpen} />
            {/* Snack */}
            <Box sx={{
                width: "min(95vw, 300px)",
                zIndex: 20000,
                background: palette.background.paper,
                color: palette.background.textPrimary,
                padding: 1,
                borderRadius: 2,
                boxShadow: 8,
                pointerEvents: "auto",
                ...noSelect,
            }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" pb={1}>
                    <Box display="flex" alignItems="center">
                        <IconCommon
                            decorative
                            fill={palette.background.textPrimary}
                            name="LargeCookie"
                            size={48}
                        />
                        <Typography variant="body1" ml={1}>
                            {t("CookiesDetails")}
                        </Typography>
                    </Box>
                    <IconButton onClick={handleClose}>
                        <IconCommon
                            decorative
                            fill={palette.background.textPrimary}
                            name="Close"
                            size={32}
                        />
                    </IconButton>
                </Stack>
                {/* Buttons */}
                <Grid container spacing={2}>
                    <Grid item xs={6}>
                        <Button
                            fullWidth
                            color="secondary"
                            onClick={handleAcceptAllCookies}
                            variant="contained"
                        >
                            {t("CookiesAcceptAll")}
                        </Button>
                    </Grid>
                    <Grid item xs={6}>
                        <Button
                            fullWidth
                            color="secondary"
                            variant="outlined"
                            onClick={() => setIsCustomizeOpen(true)}
                        >
                            {t("CookiesCustomize")}
                        </Button>
                    </Grid>
                </Grid>
            </Box>
        </>
    );
}
