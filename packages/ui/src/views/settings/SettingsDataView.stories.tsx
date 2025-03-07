import { SettingsDataView } from "./SettingsDataView.js";

export default {
    title: "Views/Settings/SettingsDataView",
    component: SettingsDataView,
};

export function Default() {
    return (
        <SettingsDataView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings data view.",
        },
    },
};
