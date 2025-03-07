import { SettingsView } from "./SettingsView.js";

export default {
    title: "Views/Settings/SettingsView",
    component: SettingsView,
};

export function Default() {
    return (
        <SettingsView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings view.",
        },
    },
};
