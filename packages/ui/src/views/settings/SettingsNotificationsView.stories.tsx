import { SettingsNotificationsView } from "./SettingsNotificationsView.js";

export default {
    title: "Views/Settings/SettingsNotificationsView",
    component: SettingsNotificationsView,
};

export function Default() {
    return (
        <SettingsNotificationsView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings notifications view.",
        },
    },
};
