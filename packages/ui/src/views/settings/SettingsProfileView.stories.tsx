import { SettingsProfileView } from "./SettingsProfileView.js";

export default {
    title: "Views/Settings/SettingsProfileView",
    component: SettingsProfileView,
};

export function Default() {
    return (
        <SettingsProfileView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings profile view.",
        },
    },
};
