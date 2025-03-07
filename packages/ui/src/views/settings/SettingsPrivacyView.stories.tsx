import { SettingsPrivacyView } from "./SettingsPrivacyView.js";

export default {
    title: "Views/Settings/SettingsPrivacyView",
    component: SettingsPrivacyView,
};

export function Default() {
    return (
        <SettingsPrivacyView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings privacy view.",
        },
    },
};
