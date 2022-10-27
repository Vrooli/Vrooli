import { Box } from "@mui/material"
import { SettingsNotificationsProps } from "../types";
import { PageTitle } from "components/text";
import { SettingsFormData } from "pages";

const helpText =
    `Notification preferences set the types and frequency of notifcations you receive. More customizations will be available in the near future.  
`

export const SettingsNotifications = ({
    profile,
    onUpdated,
}: SettingsNotificationsProps) => {
    return (
        <form style={{ overflow: 'hidden' }}>
            <PageTitle title="Notifications" helpText={helpText} />
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                Coming soon
            </Box>
        </form>
    )
}

export const settingsNotificationsFormData: SettingsFormData = {
    labels: ['Notifications', 'Notification Preferences', 'Alerts', 'Alert Preferences', 'Push Notifications'],
    items: [],
}