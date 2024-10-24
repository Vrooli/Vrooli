import { Box, Button, Grid, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { CookieSettingsDialog } from "components/dialogs/CookieSettingsDialog/CookieSettingsDialog";
import { CloseIcon, LargeCookieIcon } from "icons";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "styles";
import { CookiePreferences, setCookie } from "utils/localStorage";
import { CookiesSnackProps } from "../types";

/**
 * "This site uses cookies" consent dialog
 */
export const CookiesSnack = ({
    handleClose,
}: CookiesSnackProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [isCustomizeOpen, setIsCustomizeOpen] = useState(false);

    const handleAcceptAllCookies = () => {
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
    };

    const handleCustomizeCookies = (preferences?: CookiePreferences) => {
        setIsCustomizeOpen(false);
        if (preferences) {
            setCookie("Preferences", preferences);
            handleClose();
        }
    };

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
                        <LargeCookieIcon width="48px" height="48px" fill={palette.background.textPrimary} />
                        <Typography variant="body1" ml={1}>
                            {t("CookiesDetails")}
                        </Typography>
                    </Box>
                    <IconButton onClick={handleClose}>
                        <CloseIcon width="32px" height="32px" fill={palette.background.textPrimary} />
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
};
