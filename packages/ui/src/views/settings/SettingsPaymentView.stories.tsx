import { SettingsPaymentView } from "./SettingsPaymentView.js";

export default {
    title: "Views/Settings/SettingsPaymentView",
    component: SettingsPaymentView,
};

export function Default() {
    return (
        <SettingsPaymentView display="page" />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default settings payment view.",
        },
    },
};
