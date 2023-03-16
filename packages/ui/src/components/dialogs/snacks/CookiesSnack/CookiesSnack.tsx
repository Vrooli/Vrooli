import { Box, Button, Grid, IconButton, Stack, Typography, useTheme } from '@mui/material';
import { CloseIcon, LargeCookieIcon } from '@shared/icons';
import { CookieSettingsDialog } from 'components/dialogs/CookieSettingsDialog/CookieSettingsDialog';
import { useState } from 'react';
import { noSelect } from 'styles';
import { CookiePreferences, setCookiePreferences } from 'utils/cookies';
import { CookiesSnackProps } from '../types';

/**
 * "This site uses cookies" consent dialog
 */
export const CookiesSnack = ({
    handleClose
}: CookiesSnackProps) => {
    const { palette } = useTheme();
    const [isCustomizeOpen, setIsCustomizeOpen] = useState(false);

    const handleAcceptAllCookies = () => {
        console.log('in handleAcceptAllCookies');
        const preferences: CookiePreferences = {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        }
        // Set preference in local storage
        setCookiePreferences(preferences);
        // Close dialog
        setIsCustomizeOpen(false);
        handleClose();
    }

    const handleCustomizeCookies = (preferences?: CookiePreferences) => {
        console.log('in handlecustomizecookies', preferences);
        if (preferences) {
            // Set preference in local storage
            setCookiePreferences(preferences);
        }
        // Close dialog
        setIsCustomizeOpen(false);
        handleClose();
    }

    console.log('in cookies snack renderrr')
    return (
        <>
            {/* Customize preferences dialog */}
            <CookieSettingsDialog handleClose={handleCustomizeCookies} isOpen={isCustomizeOpen} />
            {/* Snack */}
            <Box sx={{
                width: 'min(100%, 500px)',
                zIndex: 20000,
                background: palette.background.paper,
                color: palette.background.textPrimary,
                padding: 2,
                borderRadius: 2,
                boxShadow: 8,
                pointerEvents: 'auto',
                ...noSelect,
            }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center">
                    {/* Cookie icon */}
                    <LargeCookieIcon width="80px" height="80px" fill={palette.background.textPrimary} />
                    {/* Close Icon */}
                    <IconButton onClick={handleClose}>
                        <CloseIcon width="32px" height="32px" fill={palette.background.textPrimary} />
                    </IconButton>
                </Stack>
                {/* Title */}
                <Typography variant="h6" sx={{ mt: 2 }}>
                    This site uses cookies to give you the best experience. Please accept or reject our cookie policy so we can stop asking :)
                </Typography>
                {/* Buttons */}
                <Grid container spacing={2} sx={{ mt: 2 }}>
                    <Grid item xs={12} sm={6}>
                        <Button
                            fullWidth
                            color="secondary"
                            onClick={handleAcceptAllCookies}
                        >
                            Accept all cookies
                        </Button>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                        <Button
                            fullWidth
                            color="secondary"
                            variant="outlined"
                            onClick={() => setIsCustomizeOpen(true)}
                        >
                            Customize cookies
                        </Button>
                    </Grid>
                </Grid>
            </Box>
        </>
    );
}