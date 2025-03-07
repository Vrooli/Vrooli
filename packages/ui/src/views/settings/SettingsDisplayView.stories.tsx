import { SettingsDisplayView } from "./SettingsDisplayView.js";

export default {
    title: "Views/Settings/SettingsDisplayView",
    component: SettingsDisplayView,
};

export function Default() {
    return (
        <SettingsDisplayView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings display view.",
        },
    },
};
