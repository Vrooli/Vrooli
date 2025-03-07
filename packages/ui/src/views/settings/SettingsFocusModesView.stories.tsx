import { SettingsFocusModesView } from "./SettingsFocusModesView.js";

export default {
    title: "Views/Settings/SettingsFocusModesView",
    component: SettingsFocusModesView,
};

export function Default() {
    return (
        <SettingsFocusModesView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings focus modes view.",
        },
    },
};
