import { SettingsApiView } from "./SettingsApiView.js";

export default {
    title: "Views/Settings/SettingsApiView",
    component: SettingsApiView,
};

export function Default() {
    return (
        <SettingsApiView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings api view.",
        },
    },
};
