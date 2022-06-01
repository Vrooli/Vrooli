import { Box, Stack, Typography, useTheme } from "@mui/material"
import { SettingsNotificationsProps } from "../types";
import { HelpButton } from "components/buttons";
import { TERTIARY_COLOR } from "utils";

const helpText =
    `Notification preferences set the types and frequency of notifcations you receive. More customizations will be available in the near future.  
`

export const SettingsNotifications = ({
    profile,
    onUpdated,
}: SettingsNotificationsProps) => {
    const { palette } = useTheme();

    return (
        <form style={{ overflow: 'hidden' }}>
            {/* Title */}
            <Stack direction="row" justifyContent="center" alignItems="center" sx={{
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                marginBottom: 2,
                padding: 0.5,
            }}>
                <Typography component="h1" variant="h4">Notifications</Typography>
                <HelpButton markdown={helpText} sx={{ fill: TERTIARY_COLOR }} />
            </Stack>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                Coming soon
            </Box>
        </form>
    )
}