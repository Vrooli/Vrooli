import { Box, Stack, Typography } from "@mui/material"
import { SettingsNotificationsProps } from "../types";
import { HelpButton } from "components/buttons";

const helpText =
    `Notification preferences set the types and frequency of notifcations you receive. More customizations will be available in the near future.  
`

const TERTIARY_COLOR = '#95f3cd';

export const SettingsNotifications = ({
    profile,
    onUpdated,
}: SettingsNotificationsProps) => {

    return (
        <form style={{ overflow: 'hidden' }}>
            {/* Title */}
            <Stack direction="row" justifyContent="center" alignItems="center" sx={{
                background: (t) => t.palette.primary.dark,
                color: (t) => t.palette.primary.contrastText,
                padding: 0.5,
                marginBottom: 2,
            }}>
                <Typography component="h1" variant="h3">Notification</Typography>
                <HelpButton markdown={helpText} sx={{ fill: TERTIARY_COLOR }} />
            </Stack>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                Coming soon
            </Box>
        </form>
    )
}