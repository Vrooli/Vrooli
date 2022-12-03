import { Box } from "@mui/material"
import { SettingsNotificationsProps } from "../types";
import { PageTitle } from "components/text";
import { SettingsFormData } from "pages";

export const SettingsNotifications = ({
    profile,
    onUpdated,
    session,
}: SettingsNotificationsProps) => {
    return (
        <form style={{ overflow: 'hidden' }}>
            <PageTitle titleKey='Notifications' session={session} />
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