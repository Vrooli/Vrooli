import { SettingsAuthenticationView } from "./SettingsAuthenticationView.js";

export default {
    title: "Views/Settings/SettingsAuthenticationView",
    component: SettingsAuthenticationView,
};

export function Default() {
    return (
        <SettingsAuthenticationView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings authentication view.",
        },
    },
};
